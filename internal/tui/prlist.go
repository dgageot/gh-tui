package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gh "github.com/dgageot/gh-tui/internal/github"
)

// FilterMode represents the PR list filter.
type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterMine
	FilterReviewRequested
)

// PRListModel is the PR list screen.
type PRListModel struct {
	prs         []gh.PR
	table       table.Model
	filter      FilterMode
	searchQuery string
	searching   bool
	currentUser string
	err         error
	loading     bool
	width       int
	height      int
	focused     bool
}

func NewPRListModel() PRListModel {
	columns := []table.Column{
		{Title: "#", Width: 6},
		{Title: "Title", Width: 40},
		{Title: "Author", Width: 15},
		{Title: "State", Width: 8},
		{Title: "Review", Width: 10},
		{Title: "Checks", Width: 10},
		{Title: "Updated", Width: 12},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = selectedRowStyle
	t.SetStyles(s)

	return PRListModel{
		table:   t,
		loading: true,
	}
}

func (m *PRListModel) Init() tea.Cmd {
	return nil
}

func (m *PRListModel) Update(msg tea.Msg) (PRListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Row 0 is title, row 1 is header, row 2 is header border, rows 3+ are data
			rowIdx := msg.Y - 3
			if rowIdx >= 0 && rowIdx < len(m.table.Rows()) {
				m.table.SetCursor(rowIdx)
				return *m, func() tea.Msg { return prClickedMsg{} }
			}
		}
		return *m, nil

	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "enter", "esc":
				m.searching = false
				return *m, nil
			case "backspace":
				if m.searchQuery != "" {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				}
				m.updateTableRows()
				return *m, nil
			default:
				if len(msg.String()) == 1 {
					m.searchQuery += msg.String()
					m.updateTableRows()
				}
				return *m, nil
			}
		}

		switch msg.String() {
		case "m":
			m.filter = FilterMine
			m.updateTableRows()
			return *m, nil
		case "r":
			m.filter = FilterReviewRequested
			m.updateTableRows()
			return *m, nil
		case "a":
			m.filter = FilterAll
			m.updateTableRows()
			return *m, nil
		case "/":
			m.searching = true
			m.searchQuery = ""
			return *m, nil
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return *m, cmd
}

func (m *PRListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table.SetWidth(w)
	m.table.SetHeight(h - 4)
	m.table.SetColumns(m.computeColumns(w))
	m.updateTableRows()
}

// cellPadding is the horizontal padding the bubbles table adds per column (1 left + 1 right).
const cellPadding = 2

func (m *PRListModel) computeColumns(w int) []table.Column {
	// For very narrow terminals, hide columns progressively.
	// Each column costs its width + cellPadding.
	switch {
	case w < 60:
		// Only show #, Title, State (3 columns)
		titleW := max(w-6-8-3*cellPadding, 10)
		return []table.Column{
			{Title: "#", Width: 6},
			{Title: "Title", Width: titleW},
			{Title: "State", Width: 8},
		}
	case w < 100:
		// Show #, Title, Author, State, Updated (5 columns)
		titleW := max(w-6-15-8-12-5*cellPadding, 10)
		return []table.Column{
			{Title: "#", Width: 6},
			{Title: "Title", Width: titleW},
			{Title: "Author", Width: 15},
			{Title: "State", Width: 8},
			{Title: "Updated", Width: 12},
		}
	default:
		// Show all columns (7 columns), Title gets the remaining space
		titleW := max(w-6-15-8-10-10-12-7*cellPadding, 10)
		return []table.Column{
			{Title: "#", Width: 6},
			{Title: "Title", Width: titleW},
			{Title: "Author", Width: 15},
			{Title: "State", Width: 8},
			{Title: "Review", Width: 10},
			{Title: "Checks", Width: 10},
			{Title: "Updated", Width: 12},
		}
	}
}

func (m *PRListModel) SetPRs(prs []gh.PR, currentUser string) {
	m.prs = prs
	m.currentUser = currentUser
	m.loading = false
	m.updateTableRows()
}

func (m *PRListModel) SetError(err error) {
	m.err = err
	m.loading = false
}

func (m *PRListModel) updateTableRows() {
	numCols := len(m.table.Columns())
	var rows []table.Row
	for _, pr := range m.filteredPRs() {
		checks := "—"
		if len(pr.Checks) > 0 {
			pass := 0
			for _, c := range pr.Checks {
				if c.Conclusion == "success" {
					pass++
				}
			}
			checks = fmt.Sprintf("%d/%d ✓", pass, len(pr.Checks))
		}

		state := pr.State
		if pr.Draft {
			state = "draft"
		}

		review := "—"
		switch pr.ReviewDecision {
		case "APPROVED":
			review = "✓ approved"
		case "CHANGES_REQUESTED":
			review = "✗ changes"
		}

		// Apply gray style to draft PRs
		style := func(s string) string { return s }
		if pr.Draft {
			style = func(s string) string { return draftRowStyle.Render(s) }
		}

		var row table.Row
		switch numCols {
		case 3:
			row = table.Row{
				style(fmt.Sprintf("#%d", pr.Number)),
				style(pr.Title),
				style(state),
			}
		case 5:
			row = table.Row{
				style(fmt.Sprintf("#%d", pr.Number)),
				style(pr.Title),
				style(pr.Author),
				style(state),
				style(pr.UpdatedAt.Format("Jan 02 15:04")),
			}
		default:
			row = table.Row{
				style(fmt.Sprintf("#%d", pr.Number)),
				style(pr.Title),
				style(pr.Author),
				style(state),
				style(review),
				style(checks),
				style(pr.UpdatedAt.Format("Jan 02 15:04")),
			}
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
}

func (m *PRListModel) filteredPRs() []gh.PR {
	var filtered []gh.PR
	for _, pr := range m.prs {
		switch m.filter {
		case FilterMine:
			if pr.Author != m.currentUser {
				continue
			}
		case FilterReviewRequested:
			// For simplicity, show all non-authored PRs as "review requested"
			if pr.Author == m.currentUser {
				continue
			}
		}

		if m.searchQuery != "" {
			q := strings.ToLower(m.searchQuery)
			if !strings.Contains(strings.ToLower(pr.Title), q) &&
				!strings.Contains(strings.ToLower(pr.Author), q) {
				continue
			}
		}

		filtered = append(filtered, pr)
	}
	return filtered
}

// SelectedPR returns the currently selected PR, if any.
func (m *PRListModel) SelectedPR() *gh.PR {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}

	var num int
	_, _ = fmt.Sscanf(row[0], "#%d", &num)

	for i := range m.prs {
		if m.prs[i].Number == num {
			return &m.prs[i]
		}
	}
	return nil
}

func (m *PRListModel) SetFocused(focused bool) {
	m.focused = focused
	m.table.Focus()
	if !focused {
		m.table.Blur()
	}
}

func (m *PRListModel) View() string {
	var b strings.Builder

	filterInfo := m.filterLabel()
	if m.focused {
		b.WriteString(titleStyle.Render("GitHub PRs") + "  " + statusBarStyle.Render(filterInfo))
	} else {
		b.WriteString(dimTextStyle.Render("  PRs") + "  " + statusBarStyle.Render(filterInfo))
	}
	b.WriteString("\n")

	if m.loading {
		b.WriteString(loadingStyle.Render("Loading pull requests..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	b.WriteString(m.table.View())
	b.WriteString("\n")

	if m.searching {
		b.WriteString(statusBarStyle.Render(fmt.Sprintf("Search: %s█", m.searchQuery)))
	} else {
		b.WriteString(statusBarStyle.Render("a:all m:mine r:review /:search R:refresh enter:open M:merge q:quit"))
	}

	return b.String()
}

func (m *PRListModel) filterLabel() string {
	switch m.filter {
	case FilterMine:
		return "[mine]"
	case FilterReviewRequested:
		return "[review requested]"
	default:
		return "[all]"
	}
}
