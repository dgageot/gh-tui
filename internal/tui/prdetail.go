package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	gh "github.com/dgageot/gh-tui/internal/github"
)

// DetailTab represents the active tab in PR detail.
type DetailTab int

const (
	TabOverview DetailTab = iota
	TabChecks
	TabComments
	TabFiles
)

var tabNames = []string{"Overview", "Checks", "Comments", "Files"}

// PRDetailModel is the PR detail screen.
type PRDetailModel struct {
	pr       *gh.PR
	checks   []gh.Check
	comments []gh.Comment
	files    []gh.ChangedFile
	tab      DetailTab
	viewport viewport.Model
	confirm  string // "merge" or "lgtm" or ""
	loading  bool
	err      error
	width    int
	height   int
}

func NewPRDetailModel() PRDetailModel {
	return PRDetailModel{
		loading: true,
	}
}

func (m PRDetailModel) Init() tea.Cmd {
	return nil
}

func (m PRDetailModel) Update(msg tea.Msg) (PRDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirm != "" {
			switch msg.String() {
			case "y", "Y":
				action := m.confirm
				m.confirm = ""
				if action == "merge" {
					return m, func() tea.Msg { return mergeConfirmedMsg{} }
				}
				return m, func() tea.Msg { return lgtmConfirmedMsg{} }
			case "n", "N", "esc":
				m.confirm = ""
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "tab":
			m.tab = (m.tab + 1) % 4
			m.updateViewport()
			return m, nil
		case "shift+tab":
			m.tab = (m.tab + 3) % 4
			m.updateViewport()
			return m, nil
		case "M":
			m.confirm = "merge"
			return m, nil
		case "L":
			m.confirm = "lgtm"
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

type mergeConfirmedMsg struct{}
type lgtmConfirmedMsg struct{}

func (m *PRDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport = viewport.New(w, h-4)
	m.viewport.SetContent(m.renderTabContent())
}

func (m *PRDetailModel) SetData(pr *gh.PR, checks []gh.Check, comments []gh.Comment, files []gh.ChangedFile) {
	m.pr = pr
	m.checks = checks
	m.comments = comments
	m.files = files
	m.loading = false
	m.updateViewport()
}

func (m *PRDetailModel) SetError(err error) {
	m.err = err
	m.loading = false
}

func (m *PRDetailModel) updateViewport() {
	m.viewport.SetContent(m.renderTabContent())
	m.viewport.GotoTop()
}

func (m PRDetailModel) View() string {
	var b strings.Builder

	if m.loading {
		b.WriteString(loadingStyle.Render("Loading PR details..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
		return b.String()
	}

	if m.pr == nil {
		return ""
	}

	// Title
	b.WriteString(titleStyle.Render(fmt.Sprintf("#%d %s", m.pr.Number, m.pr.Title)))
	b.WriteString("\n")

	// Tabs
	var tabs []string
	for i, name := range tabNames {
		if DetailTab(i) == m.tab {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, tabStyle.Render(name))
		}
	}
	b.WriteString(strings.Join(tabs, " "))
	b.WriteString("\n")

	// Viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Confirm dialog or help
	if m.confirm != "" {
		prompt := fmt.Sprintf("Confirm %s? (y/n)", m.confirm)
		b.WriteString(confirmStyle.Render(prompt))
	} else {
		b.WriteString(statusBarStyle.Render("tab:switch L:LGTM M:merge esc:back"))
	}

	return b.String()
}

func (m PRDetailModel) renderTabContent() string {
	if m.pr == nil {
		return ""
	}

	switch m.tab {
	case TabOverview:
		return m.renderOverview()
	case TabChecks:
		return m.renderChecks()
	case TabComments:
		return m.renderComments()
	case TabFiles:
		return m.renderFiles()
	default:
		return ""
	}
}

func (m PRDetailModel) renderOverview() string {
	var b strings.Builder

	state := m.pr.State
	if m.pr.Draft {
		state = "draft"
	}
	b.WriteString(fmt.Sprintf("State: %s  |  Author: %s  |  Mergeable: %v\n", state, m.pr.Author, m.pr.Mergeable))

	if len(m.pr.Labels) > 0 {
		var labels []string
		for _, l := range m.pr.Labels {
			labels = append(labels, labelStyle.Render(l))
		}
		b.WriteString("Labels: " + strings.Join(labels, " ") + "\n")
	}

	b.WriteString("\n")
	if m.pr.Body != "" {
		b.WriteString(bodyStyle.Render(m.pr.Body))
	} else {
		b.WriteString(bodyStyle.Render("(no description)"))
	}

	return b.String()
}

func (m PRDetailModel) renderChecks() string {
	if len(m.checks) == 0 {
		return bodyStyle.Render("No checks found.")
	}

	var b strings.Builder
	for _, c := range m.checks {
		var icon string
		var style func(strs ...string) string
		switch c.Conclusion {
		case "success":
			icon = "✓"
			style = checkPassStyle.Render
		case "failure", "cancelled", "timed_out", "action_required":
			icon = "✗"
			style = checkFailStyle.Render
		default:
			icon = "●"
			style = checkPendStyle.Render
		}
		b.WriteString(style(fmt.Sprintf("  %s %s", icon, c.Name)))
		b.WriteString("\n")
	}
	return b.String()
}

func (m PRDetailModel) renderComments() string {
	if len(m.comments) == 0 {
		return bodyStyle.Render("No comments.")
	}

	var b strings.Builder
	for _, c := range m.comments {
		author := commentAuthorStyle.Render(c.Author)
		date := commentDateStyle.Render(c.CreatedAt.Format("Jan 02 15:04"))
		b.WriteString(fmt.Sprintf("%s  %s\n", author, date))
		b.WriteString(c.Body)
		b.WriteString("\n\n")
	}
	return b.String()
}

func (m PRDetailModel) renderFiles() string {
	if len(m.files) == 0 {
		return bodyStyle.Render("No files changed.")
	}

	var b strings.Builder
	for _, f := range m.files {
		adds := additionsStyle.Render(fmt.Sprintf("+%d", f.Additions))
		dels := deletionsStyle.Render(fmt.Sprintf("-%d", f.Deletions))
		b.WriteString(fmt.Sprintf("  %s %s  %s  %s\n", f.Status, f.Filename, adds, dels))
	}
	return b.String()
}
