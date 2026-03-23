package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	gh "github.com/dgageot/gh-tui/internal/github"
)

type issueClickedMsg struct{}

// IssueDetailModel is the issue detail screen.
type IssueDetailModel struct {
	issue          *gh.Issue
	comments       []gh.IssueComment
	viewport       viewport.Model
	err            error
	width          int
	height         int
	issueLoaded    bool
	commentsLoaded bool
}

func NewIssueDetailModel() IssueDetailModel {
	return IssueDetailModel{}
}

func (m *IssueDetailModel) Update(msg tea.Msg) (IssueDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return *m, cmd

	case tea.KeyMsg:
		// viewport handles scrolling
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return *m, cmd
}

func (m *IssueDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport = viewport.New(w, h-3)
	m.viewport.SetContent(m.renderContent())
}

func (m *IssueDetailModel) SetIssue(issue *gh.Issue) {
	m.issue = issue
	m.issueLoaded = true
	m.updateViewport()
}

func (m *IssueDetailModel) SetComments(comments []gh.IssueComment) {
	m.comments = comments
	m.commentsLoaded = true
	m.updateViewport()
}

func (m *IssueDetailModel) SetError(err error) {
	m.err = err
}

func (m *IssueDetailModel) updateViewport() {
	m.viewport.SetContent(m.renderContent())
	m.viewport.GotoTop()
}

func (m *IssueDetailModel) loading() bool {
	return !m.issueLoaded && m.err == nil
}

func (m *IssueDetailModel) View() string {
	var b strings.Builder

	if m.loading() {
		b.WriteString(loadingStyle.Render("  Loading issue details…"))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render("  Error: " + m.err.Error()))
		return b.String()
	}

	if m.issue == nil {
		return ""
	}

	b.WriteString(" " + titleStyle.Render(fmt.Sprintf(" #%d", m.issue.Number)) + "  " + titleStyle.Render(m.issue.Title))
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	b.WriteString(formatHelpKeys("esc", "back", "q", "quit"))

	return b.String()
}

func (m *IssueDetailModel) renderContent() string {
	if m.issue == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n")

	// State badge
	badge := stateBadge(m.issue.State, false)
	b.WriteString("  " + badge + "\n\n")

	fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Author"), commentAuthorStyle.Render(m.issue.Author))
	fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Updated"), m.issue.UpdatedAt.Format("Jan 02, 2006 15:04"))

	if len(m.issue.Labels) > 0 {
		var labels []string
		for _, l := range m.issue.Labels {
			labels = append(labels, labelStyle.Render(l))
		}
		fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Labels"), strings.Join(labels, " "))
	}

	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render("  Description"))
	b.WriteString("\n")
	if m.issue.Body != "" {
		b.WriteString(bodyStyle.Render(m.issue.Body))
	} else {
		b.WriteString(bodyStyle.Render(dimTextStyle.Render("No description provided.")))
	}

	// Comments
	switch {
	case m.commentsLoaded && len(m.comments) > 0:
		b.WriteString("\n")
		b.WriteString(sectionTitleStyle.Render(fmt.Sprintf("  💬 Comments (%d)", len(m.comments))))
		b.WriteString("\n")
		for i, c := range m.comments {
			if i > 0 {
				b.WriteString(commentSeparatorStyle.Render("  " + strings.Repeat("─", min(60, m.width-4))))
				b.WriteString("\n")
			}
			author := commentAuthorStyle.Render(c.Author)
			date := commentDateStyle.Render(c.CreatedAt.Format("Jan 02, 2006 15:04"))
			fmt.Fprintf(&b, "  %s  %s\n", author, date)
			for line := range strings.SplitSeq(c.Body, "\n") {
				fmt.Fprintf(&b, "  %s\n", line)
			}
			b.WriteString("\n")
		}
	case m.commentsLoaded:
		b.WriteString("\n")
		b.WriteString(bodyStyle.Render(dimTextStyle.Render("No comments yet.")))
	default:
		b.WriteString("\n")
		b.WriteString(loadingStyle.Render("  Loading comments…"))
	}

	return b.String()
}
