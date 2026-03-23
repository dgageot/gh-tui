package tui

import (
	"context"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	gh "github.com/dgageot/gh-tui/internal/github"
)

// Screen represents the current screen.
type Screen int

const (
	ScreenList Screen = iota
	ScreenDetail
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

// AppModel is the root model.
type AppModel struct {
	client      *gh.Client
	screen      Screen
	list        PRListModel
	detail      PRDetailModel
	width       int
	height      int
	statusMsg   string
	currentPR   *gh.PR
	currentUser string
}

func NewAppModel(client *gh.Client) AppModel {
	return AppModel{
		client: client,
		screen: ScreenList,
		list:   NewPRListModel(),
		detail: NewPRDetailModel(),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.loadPRs()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		m.detail.SetSize(msg.Width, msg.Height)
		return m, nil

	case prsLoadedMsg:
		m.currentUser = msg.user
		m.list.SetPRs(msg.prs, msg.user)
		return m, nil

	case prsErrorMsg:
		m.list.SetError(msg.err)
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

	case tea.KeyMsg:
		m.statusMsg = ""

		switch m.screen {
		case ScreenList:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter":
				if pr := m.list.SelectedPR(); pr != nil {
					m.currentPR = pr
					m.screen = ScreenDetail
					m.detail = NewPRDetailModel()
					m.detail.SetSize(m.width, m.height)
					return m, m.loadDetail(pr.Number, pr.HeadRef)
				}
			case "R":
				m.list.loading = true
				return m, m.loadPRs()
			case "M":
				if pr := m.list.SelectedPR(); pr != nil {
					m.currentPR = pr
					m.detail.confirm = "merge"
					m.screen = ScreenDetail
					m.detail = NewPRDetailModel()
					m.detail.SetSize(m.width, m.height)
					m.detail.confirm = "merge"
					return m, m.loadDetail(pr.Number, pr.HeadRef)
				}
			default:
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			}

		case ScreenDetail:
			switch msg.String() {
			case "esc":
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
		}
	}

	return m, nil
}

func (m AppModel) View() string {
	var view string
	switch m.screen {
	case ScreenList:
		view = m.list.View()
	case ScreenDetail:
		view = m.detail.View()
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
