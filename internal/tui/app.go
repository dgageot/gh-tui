package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	gh "github.com/dgageot/gh-tui/internal/github"
)

// Screen represents the current screen.
type Screen int

const (
	ScreenList Screen = iota
	ScreenPRDetail
	ScreenIssueDetail
)

// Pane tracks which pane is focused on the list screen.
type Pane int

const (
	PanePRs Pane = iota
	PaneIssues
)

// Context exposes state for future agent integration.
type Context struct {
	Owner       string
	Repo        string
	CurrentPR   *gh.PR
	FilteredPRs []gh.PR
}

// Messages
type prsLoadedMsg struct {
	prs  []gh.PR
	user string
}

type prsErrorMsg struct{ err error }

type issuesLoadedMsg struct {
	issues []gh.Issue
}

type issuesErrorMsg struct{ err error }

type (
	detailPRMsg       struct{ pr *gh.PR }
	detailChecksMsg   struct{ checks []gh.Check }
	detailCommentsMsg struct{ comments []gh.Comment }
	detailFilesMsg    struct{ files []gh.ChangedFile }
	detailReviewMsg   struct{ decision string }
	detailErrorMsg    struct{ err error }
	mergeResultMsg    struct{ err error }
	lgtmResultMsg     struct{ err error }
)

type (
	issueDetailMsg         struct{ issue *gh.Issue }
	issueDetailCommentsMsg struct{ comments []gh.IssueComment }
	issueDetailErrorMsg    struct{ err error }
)

// AppModel is the root model.
type AppModel struct {
	client       *gh.Client
	screen       Screen
	pane         Pane
	list         PRListModel
	issueList    IssueListModel
	detail       PRDetailModel
	issueDetail  IssueDetailModel
	width        int
	height       int
	statusMsg    string
	currentPR    *gh.PR
	currentIssue *gh.Issue
	currentUser  string
}

func NewAppModel(client *gh.Client) AppModel {
	return AppModel{
		client:    client,
		screen:    ScreenList,
		pane:      PanePRs,
		list:      NewPRListModel(),
		issueList: NewIssueListModel(),
		detail:    NewPRDetailModel(),
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.loadPRs(), m.loadIssues())
}

func updateFocus(m *AppModel) {
	m.list.SetFocused(m.pane == PanePRs)
	m.issueList.SetFocused(m.pane == PaneIssues)
}

func setSizes(m *AppModel) {
	if m.width == 0 || m.height == 0 {
		return
	}
	// Split: top half for PRs, bottom half for issues, 1 line separator
	topH := (m.height - 1) / 2
	botH := m.height - topH - 1
	m.list.SetSize(m.width, topH)
	m.issueList.SetSize(m.width, botH)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		setSizes(&m)
		m.detail.SetSize(msg.Width, msg.Height)
		m.issueDetail.SetSize(msg.Width, msg.Height)
		return m, nil

	case prsLoadedMsg:
		m.currentUser = msg.user
		m.list.SetPRs(msg.prs, msg.user)
		updateFocus(&m)
		return m, nil

	case prsErrorMsg:
		m.list.SetError(msg.err)
		return m, nil

	case issuesLoadedMsg:
		m.issueList.SetIssues(msg.issues)
		updateFocus(&m)
		return m, nil

	case issuesErrorMsg:
		m.issueList.SetError(msg.err)
		return m, nil

	case detailPRMsg:
		m.detail.SetPR(msg.pr)
		return m, nil

	case detailChecksMsg:
		m.detail.SetChecks(msg.checks)
		return m, nil

	case detailCommentsMsg:
		m.detail.SetComments(msg.comments)
		return m, nil

	case detailFilesMsg:
		m.detail.SetFiles(msg.files)
		return m, nil

	case detailReviewMsg:
		m.detail.SetReviewDecision(msg.decision)
		return m, nil

	case detailErrorMsg:
		m.detail.SetError(msg.err)
		return m, nil

	case issueDetailMsg:
		m.issueDetail.SetIssue(msg.issue)
		return m, nil

	case issueDetailCommentsMsg:
		m.issueDetail.SetComments(msg.comments)
		return m, nil

	case issueDetailErrorMsg:
		m.issueDetail.SetError(msg.err)
		return m, nil

	case mergeResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Merge failed: %v", msg.err)
		} else {
			m.statusMsg = "PR merged successfully!"
			m.screen = ScreenList
			return m, m.loadPRs()
		}
		return m, nil

	case lgtmResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("LGTM failed: %v", msg.err)
		} else {
			m.statusMsg = "LGTM submitted!"
		}
		return m, nil

	case mergeConfirmedMsg:
		if m.currentPR != nil {
			return m, m.mergePR(m.currentPR.Number)
		}
		return m, nil

	case lgtmConfirmedMsg:
		if m.currentPR != nil {
			return m, m.approvePR(m.currentPR.Number)
		}
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case prClickedMsg:
		return m.openSelectedPR()

	case issueClickedMsg:
		return m.openSelectedIssue()

	case tea.KeyMsg:
		m.statusMsg = ""
		return m.handleKey(msg)
	}

	return m, nil
}

func (m AppModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenList:
		topH := (m.height - 1) / 2
		if msg.Y < topH {
			if m.pane != PanePRs {
				m.pane = PanePRs
				updateFocus(&m)
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		// Adjust Y for the issue list
		adjusted := tea.MouseMsg{
			X:      msg.X,
			Y:      msg.Y - topH - 1,
			Action: msg.Action,
			Button: msg.Button,
		}
		if m.pane != PaneIssues {
			m.pane = PaneIssues
			updateFocus(&m)
		}
		var cmd tea.Cmd
		m.issueList, cmd = m.issueList.Update(adjusted)
		return m, cmd
	case ScreenPRDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	case ScreenIssueDetail:
		var cmd tea.Cmd
		m.issueDetail, cmd = m.issueDetail.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenList:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.pane == PanePRs {
				m.pane = PaneIssues
			} else {
				m.pane = PanePRs
			}
			updateFocus(&m)
			return m, nil
		case "enter":
			if m.pane == PanePRs {
				return m.openSelectedPR()
			}
			return m.openSelectedIssue()
		case "R":
			m.list.loading = true
			return m, tea.Batch(m.loadPRs(), m.loadIssues())
		case "M":
			if m.pane == PanePRs {
				if pr := m.list.SelectedPR(); pr != nil {
					m.currentPR = pr
					m.detail = NewPRDetailModel()
					m.detail.currentUser = m.currentUser
					m.detail.SetSize(m.width, m.height)
					m.detail.confirm = "merge"
					m.screen = ScreenPRDetail
					return m, m.loadDetail(pr.Number, pr.HeadRef)
				}
			}
			return m, nil
		default:
			if m.pane == PanePRs {
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			}
			var cmd tea.Cmd
			m.issueList, cmd = m.issueList.Update(msg)
			return m, cmd
		}

	case ScreenPRDetail:
		switch msg.String() {
		case "esc":
			if m.detail.viewingDiff {
				var cmd tea.Cmd
				m.detail, cmd = m.detail.Update(msg)
				return m, cmd
			}
			m.screen = ScreenList
			m.currentPR = nil
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}

	case ScreenIssueDetail:
		switch msg.String() {
		case "esc":
			m.screen = ScreenList
			m.currentIssue = nil
			return m, nil
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			var cmd tea.Cmd
			m.issueDetail, cmd = m.issueDetail.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m AppModel) openSelectedPR() (tea.Model, tea.Cmd) {
	pr := m.list.SelectedPR()
	if pr == nil {
		return m, nil
	}
	m.currentPR = pr
	m.screen = ScreenPRDetail
	m.detail = NewPRDetailModel()
	m.detail.currentUser = m.currentUser
	m.detail.SetSize(m.width, m.height)
	return m, m.loadDetail(pr.Number, pr.HeadRef)
}

func (m AppModel) openSelectedIssue() (tea.Model, tea.Cmd) {
	issue := m.issueList.SelectedIssue()
	if issue == nil {
		return m, nil
	}
	m.currentIssue = issue
	m.screen = ScreenIssueDetail
	m.issueDetail = NewIssueDetailModel()
	m.issueDetail.SetSize(m.width, m.height)
	return m, m.loadIssueDetail(issue.Number)
}

func (m AppModel) View() string {
	var view string
	switch m.screen {
	case ScreenList:
		sep := separatorStyle.Render(strings.Repeat("─", m.width))
		view = m.list.View() + "\n" + sep + "\n" + m.issueList.View()
	case ScreenPRDetail:
		view = m.detail.View()
	case ScreenIssueDetail:
		view = m.issueDetail.View()
	}

	if m.statusMsg != "" {
		view += "\n" + statusBarStyle.Render(m.statusMsg)
	}

	return view
}

// AgentContext returns the current context for agent integration.
func (m AppModel) AgentContext() Context {
	return Context{
		Owner:       m.client.Owner,
		Repo:        m.client.Repo,
		CurrentPR:   m.currentPR,
		FilteredPRs: m.list.prs,
	}
}

// Commands

func (m AppModel) loadPRs() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		user, err := m.client.CurrentUser(ctx)
		if err != nil {
			return prsErrorMsg{err: err}
		}

		prs, err := m.client.ListPRs(ctx)
		if err != nil {
			return prsErrorMsg{err: err}
		}

		// Fetch review decisions concurrently
		var wg sync.WaitGroup
		for i := range prs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				decision, _ := m.client.GetReviewDecision(ctx, prs[i].Number)
				prs[i].ReviewDecision = decision
			}(i)
		}
		wg.Wait()

		return prsLoadedMsg{prs: prs, user: user}
	}
}

func (m AppModel) loadIssues() tea.Cmd {
	return func() tea.Msg {
		issues, err := m.client.ListIssues(context.Background())
		if err != nil {
			return issuesErrorMsg{err: err}
		}
		return issuesLoadedMsg{issues: issues}
	}
}

func (m AppModel) loadDetail(number int, ref string) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			pr, err := m.client.GetPRDetail(context.Background(), number)
			if err != nil {
				return detailErrorMsg{err: err}
			}
			return detailPRMsg{pr: pr}
		},
		func() tea.Msg {
			checks, _ := m.client.GetChecks(context.Background(), ref)
			return detailChecksMsg{checks: checks}
		},
		func() tea.Msg {
			comments, _ := m.client.GetComments(context.Background(), number)
			return detailCommentsMsg{comments: comments}
		},
		func() tea.Msg {
			files, _ := m.client.GetChangedFiles(context.Background(), number)
			return detailFilesMsg{files: files}
		},
		func() tea.Msg {
			decision, _ := m.client.GetReviewDecision(context.Background(), number)
			return detailReviewMsg{decision: decision}
		},
	)
}

func (m AppModel) loadIssueDetail(number int) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			issue, err := m.client.GetIssue(context.Background(), number)
			if err != nil {
				return issueDetailErrorMsg{err: err}
			}
			return issueDetailMsg{issue: issue}
		},
		func() tea.Msg {
			comments, err := m.client.GetIssueComments(context.Background(), number)
			if err != nil {
				return issueDetailErrorMsg{err: err}
			}
			return issueDetailCommentsMsg{comments: comments}
		},
	)
}

func (m AppModel) mergePR(number int) tea.Cmd {
	return func() tea.Msg {
		return mergeResultMsg{err: m.client.MergePR(context.Background(), number)}
	}
}

func (m AppModel) approvePR(number int) tea.Cmd {
	return func() tea.Msg {
		return lgtmResultMsg{err: m.client.ApprovePR(context.Background(), number)}
	}
}
