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
}

func NewPRListModel() PRListModel {
	columns := []table.Column{
		{Title: "#", Width: 6},
		{Title: "Title", Width: 40},
		{Title: "Author", Width: 15},
		{Title: "State", Width: 8},
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
	if msg, ok := msg.(tea.KeyMsg); ok {
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
	// Reserve space for title, status bar, and filter line
	m.table.SetWidth(w)
	m.table.SetHeight(h - 4)
	// Adjust title column width
	if w > 91 {
		cols := m.table.Columns()
		cols[1] = table.Column{Title: "Title", Width: w - 51 - 6}
		m.table.SetColumns(cols)
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

		rows = append(rows, table.Row{
			fmt.Sprintf("#%d", pr.Number),
			truncate(pr.Title, 50),
			pr.Author,
			state,
			checks,
			pr.UpdatedAt.Format("Jan 02 15:04"),
		})
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

func (m *PRListModel) View() string {
	var b strings.Builder

	title := titleStyle.Render("GitHub PRs")
	filterInfo := m.filterLabel()
	b.WriteString(title + "  " + statusBarStyle.Render(filterInfo))
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
