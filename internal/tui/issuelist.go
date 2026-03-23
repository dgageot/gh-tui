package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	gh "github.com/dgageot/gh-tui/internal/github"
)

// IssueListModel is the issue list component.
type IssueListModel struct {
	issues  []gh.Issue
	table   table.Model
	err     error
	loading bool
	width   int
	height  int
	focused bool
}

func NewIssueListModel() IssueListModel {
	columns := []table.Column{
		{Title: "#", Width: 6},
		{Title: "Title", Width: 40},
		{Title: "Author", Width: 15},
		{Title: "💬", Width: 3},
		{Title: "Updated", Width: 12},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
	)

	s := table.DefaultStyles()
	s.Header = tableHeaderStyle
	s.Selected = selectedRowStyle
	t.SetStyles(s)

	return IssueListModel{
		table:   t,
		loading: true,
	}
}

func (m *IssueListModel) SetFocused(focused bool) {
	m.focused = focused
	m.table.Focus()
	if !focused {
		m.table.Blur()
	}
}

func (m *IssueListModel) Update(msg tea.Msg) (IssueListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			rowIdx := msg.Y - 3
			if rowIdx >= 0 && rowIdx < len(m.table.Rows()) {
				m.table.SetCursor(rowIdx)
				return *m, func() tea.Msg { return issueClickedMsg{} }
			}
		}
		return *m, nil

	case tea.KeyMsg:
		// pass through to table
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return *m, cmd
}

func (m *IssueListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.table.SetWidth(w)
	m.table.SetHeight(h - 3)
	m.table.SetColumns(m.computeColumns(w))
	m.updateTableRows()
}

func (m *IssueListModel) computeColumns(w int) []table.Column {
	titleW := max(w-6-15-3-12-5*cellPadding, 10)
	return []table.Column{
		{Title: "#", Width: 6},
		{Title: "Title", Width: titleW},
		{Title: "Author", Width: 15},
		{Title: "💬", Width: 3},
		{Title: "Updated", Width: 12},
	}
}

func (m *IssueListModel) SetIssues(issues []gh.Issue) {
	m.issues = issues
	m.loading = false
	m.updateTableRows()
}

func (m *IssueListModel) SetError(err error) {
	m.err = err
	m.loading = false
}

func (m *IssueListModel) updateTableRows() {
	var rows []table.Row
	for _, issue := range m.issues {
		rows = append(rows, table.Row{
			fmt.Sprintf("#%d", issue.Number),
			issue.Title,
			issue.Author,
			strconv.Itoa(issue.Comments),
			issue.UpdatedAt.Format("Jan 02 15:04"),
		})
	}
	m.table.SetRows(rows)
}

// SelectedIssue returns the currently selected issue, if any.
func (m *IssueListModel) SelectedIssue() *gh.Issue {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}

	var num int
	_, _ = fmt.Sscanf(row[0], "#%d", &num)

	for i := range m.issues {
		if m.issues[i].Number == num {
			return &m.issues[i]
		}
	}
	return nil
}

func (m *IssueListModel) View() string {
	var b strings.Builder

	b.WriteString(paneTitleBar("  Issues", m.focused, m.width, ""))
	b.WriteString("\n")

	if m.loading {
		b.WriteString(loadingStyle.Render("  Loading issues…"))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render("  Error: " + m.err.Error()))
		return b.String()
	}

	b.WriteString(m.table.View())

	return b.String()
}
