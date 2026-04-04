package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

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
type mainPageLoadedMsg struct {
	prs    []gh.PR
	issues []gh.Issue
	user   string
}

type mainPageErrorMsg struct{ err error }

type (
	detailLoadedMsg struct{ detail *gh.PRDetail }
	detailFilesMsg  struct{ files []gh.ChangedFile }
	detailErrorMsg  struct{ err error }
	mergeResultMsg  struct{ err error }
	lgtmResultMsg   struct{ err error }
)

type (
	issueDetailLoadedMsg struct {
		issue    *gh.Issue
		comments []gh.IssueComment
	}
	issueDetailErrorMsg struct{ err error }
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
		detail:    PRDetailModel{},
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.loadMainPage()
}

func updateFocus(m *AppModel) {
	m.list.SetFocused(m.pane == PanePRs)
	m.issueList.SetFocused(m.pane == PaneIssues)
}

// topPaneHeight returns the height of the top pane (PRs) in the split layout.
func (m *AppModel) topPaneHeight() int {
	return (m.height - 1) / 2
}

func setSizes(m *AppModel) {
	if m.width == 0 || m.height == 0 {
		return
	}
	topH := m.topPaneHeight()
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

	case mainPageLoadedMsg:
		m.currentUser = msg.user
		m.list.SetPRs(msg.prs, msg.user)
		m.issueList.SetIssues(msg.issues)
		updateFocus(&m)
		return m, nil

	case mainPageErrorMsg:
		m.list.SetError(msg.err)
		m.issueList.SetError(msg.err)
		return m, nil

	case detailLoadedMsg:
		m.detail.SetPR(msg.detail.PR)
		m.detail.SetChecks(msg.detail.Checks)
		m.detail.SetComments(msg.detail.Comments)
		return m, nil

	case detailFilesMsg:
		m.detail.SetFiles(msg.files)
		return m, nil

	case detailErrorMsg:
		m.detail.SetError(msg.err)
		return m, nil

	case issueDetailLoadedMsg:
		m.issueDetail.SetIssue(msg.issue)
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
			return m, m.loadMainPage()
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
		topH := m.topPaneHeight()
		if msg.Y < topH {
			if m.pane != PanePRs {
				m.pane = PanePRs
				updateFocus(&m)
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		if msg.Y == topH {
			// Separator line — ignore
			return m, nil
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
			return m, m.loadMainPage()
		case "M":
			if m.pane == PanePRs {
				if pr := m.list.SelectedPR(); pr != nil {
					model, cmd := m.openSelectedPR()
					app := model.(AppModel)
					app.detail.confirm = "merge"
					return app, cmd
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
		case "b":
			if m.currentPR != nil {
				url := fmt.Sprintf("https://github.com/%s/%s/pull/%d", m.client.Owner, m.client.Repo, m.currentPR.Number)
				_ = exec.Command("open", url).Start()
			}
			return m, nil
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
		case "b":
			if m.currentIssue != nil {
				url := fmt.Sprintf("https://github.com/%s/%s/issues/%d", m.client.Owner, m.client.Repo, m.currentIssue.Number)
				_ = exec.Command("open", url).Start()
			}
			return m, nil
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
	m.detail = PRDetailModel{}
	m.detail.currentUser = m.currentUser
	m.detail.SetSize(m.width, m.height)
	return m, m.loadDetail(pr.Number)
}

func (m AppModel) openSelectedIssue() (tea.Model, tea.Cmd) {
	issue := m.issueList.SelectedIssue()
	if issue == nil {
		return m, nil
	}
	m.currentIssue = issue
	m.screen = ScreenIssueDetail
	m.issueDetail = IssueDetailModel{}
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

func (m AppModel) loadMainPage() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.ListMainPage(context.Background())
		if err != nil {
			return mainPageErrorMsg{err: err}
		}
		return mainPageLoadedMsg{prs: result.PRs, issues: result.Issues, user: result.CurrentUser}
	}
}

func (m AppModel) loadDetail(number int) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			detail, err := m.client.GetPRDetail(context.Background(), number)
			if err != nil {
				return detailErrorMsg{err: err}
			}
			return detailLoadedMsg{detail: detail}
		},
		func() tea.Msg {
			files, _ := m.client.GetChangedFiles(context.Background(), number)
			return detailFilesMsg{files: files}
		},
	)
}

func (m AppModel) loadIssueDetail(number int) tea.Cmd {
	return func() tea.Msg {
		issue, comments, err := m.client.IssueDetail(context.Background(), number)
		if err != nil {
			return issueDetailErrorMsg{err: err}
		}
		return issueDetailLoadedMsg{issue: issue, comments: comments}
	}
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
