package tui

import (
	"fmt"
	"slices"
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
	pr          *gh.PR
	checks      []gh.Check
	comments    []gh.Comment
	files       []gh.ChangedFile
	tab         DetailTab
	viewport    viewport.Model
	confirm     string // "merge" or "lgtm" or ""
	err         error
	width       int
	height      int
	currentUser string

	// Per-section loading state
	prLoaded       bool
	checksLoaded   bool
	commentsLoaded bool
	filesLoaded    bool
}

func NewPRDetailModel() PRDetailModel {
	return PRDetailModel{}
}

func (m *PRDetailModel) Init() tea.Cmd {
	return nil
}

func (m *PRDetailModel) Update(msg tea.Msg) (PRDetailModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.confirm != "" {
			switch msg.String() {
			case "y", "Y":
				action := m.confirm
				m.confirm = ""
				if action == "merge" {
					return *m, func() tea.Msg { return mergeConfirmedMsg{} }
				}
				return *m, func() tea.Msg { return lgtmConfirmedMsg{} }
			case "n", "N", "esc":
				m.confirm = ""
				return *m, nil
			}
			return *m, nil
		}

		switch msg.String() {
		case "tab":
			m.tab = (m.tab + 1) % 4
			m.updateViewport()
			return *m, nil
		case "shift+tab":
			m.tab = (m.tab + 3) % 4
			m.updateViewport()
			return *m, nil
		case "M":
			if m.pr != nil && !m.pr.Mergeable {
				return *m, nil
			}
			m.confirm = "merge"
			return *m, nil
		case "L":
			if m.isOwnPR() {
				return *m, nil
			}
			m.confirm = "lgtm"
			return *m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return *m, cmd
}

type (
	mergeConfirmedMsg struct{}
	lgtmConfirmedMsg  struct{}
)

func (m *PRDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport = viewport.New(w, h-4)
	m.viewport.SetContent(m.renderTabContent())
}

func (m *PRDetailModel) SetPR(pr *gh.PR) {
	m.pr = pr
	m.prLoaded = true
	m.updateViewport()
}

func (m *PRDetailModel) SetChecks(checks []gh.Check) {
	m.checks = checks
	m.checksLoaded = true
	m.updateViewport()
}

func (m *PRDetailModel) SetComments(comments []gh.Comment) {
	m.comments = comments
	m.commentsLoaded = true
	m.updateViewport()
}

func (m *PRDetailModel) SetFiles(files []gh.ChangedFile) {
	m.files = files
	m.filesLoaded = true
	m.updateViewport()
}

func (m *PRDetailModel) SetReviewDecision(decision string) {
	if m.pr != nil {
		m.pr.ReviewDecision = decision
	}
	m.updateViewport()
}

func (m *PRDetailModel) SetError(err error) {
	m.err = err
}

func (m *PRDetailModel) updateViewport() {
	m.viewport.SetContent(m.renderTabContent())
	m.viewport.GotoTop()
}

func (m *PRDetailModel) isOwnPR() bool {
	return m.pr != nil && m.currentUser != "" && m.pr.Author == m.currentUser
}

func (m *PRDetailModel) isMergeable() bool {
	return m.pr != nil && m.pr.Mergeable
}

func (m *PRDetailModel) helpKeys() string {
	keys := "tab:switch"
	if !m.isOwnPR() {
		keys += " L:LGTM"
	}
	if m.isMergeable() {
		keys += " M:merge"
	}
	keys += " esc:back"
	return keys
}

func (m *PRDetailModel) loading() bool {
	return !m.prLoaded && m.err == nil
}

func (m *PRDetailModel) View() string {
	var b strings.Builder

	if m.loading() {
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
	switch {
	case m.confirm != "":
		prompt := fmt.Sprintf("Confirm %s? (y/n)", m.confirm)
		b.WriteString(confirmStyle.Render(prompt))
	case m.isOwnPR():
		b.WriteString(statusBarStyle.Render(m.helpKeys()))
	default:
		b.WriteString(statusBarStyle.Render(m.helpKeys()))
	}

	return b.String()
}

func (m *PRDetailModel) renderTabContent() string {
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

func (m *PRDetailModel) renderOverview() string {
	var b strings.Builder

	// State and branch
	state := m.pr.State
	if m.pr.Draft {
		state = "draft"
	}
	fmt.Fprintf(&b, "  %-12s %s\n", dimTextStyle.Render("State"), state)
	fmt.Fprintf(&b, "  %-12s %s\n", dimTextStyle.Render("Author"), m.pr.Author)
	fmt.Fprintf(&b, "  %-12s %s\n", dimTextStyle.Render("Branch"), branchStyle.Render(m.pr.HeadRef))
	fmt.Fprintf(&b, "  %-12s %s\n", dimTextStyle.Render("Updated"), m.pr.UpdatedAt.Format("Jan 02, 2006 15:04"))

	// Mergeable
	var mergeText string
	if m.pr.Mergeable {
		mergeText = mergeableYesStyle.Render("✓ Yes")
	} else {
		mergeText = mergeableNoStyle.Render("✗ No")
	}
	fmt.Fprintf(&b, "  %-12s %s\n", dimTextStyle.Render("Mergeable"), mergeText)

	// Review decision
	var reviewText string
	switch m.pr.ReviewDecision {
	case "APPROVED":
		reviewText = reviewApprovedStyle.Render("✓ Approved")
	case "CHANGES_REQUESTED":
		reviewText = reviewChangesStyle.Render("✗ Changes requested")
	default:
		reviewText = reviewPendingStyle.Render("Pending")
	}
	fmt.Fprintf(&b, "  %-12s %s\n", dimTextStyle.Render("Review"), reviewText)

	// Labels
	if len(m.pr.Labels) > 0 {
		var labels []string
		for _, l := range m.pr.Labels {
			labels = append(labels, labelStyle.Render(l))
		}
		fmt.Fprintf(&b, "  %-12s %s\n", dimTextStyle.Render("Labels"), strings.Join(labels, " "))
	}

	// Checks summary
	if m.checksLoaded && len(m.checks) > 0 {
		pass, fail, pending := 0, 0, 0
		for _, c := range m.checks {
			switch c.Conclusion {
			case "success":
				pass++
			case "failure", "cancelled", "timed_out", "action_required":
				fail++
			default:
				pending++
			}
		}
		var checksText string
		switch {
		case fail > 0:
			checksText = checkFailStyle.Render(fmt.Sprintf("%d/%d passing, %d failed", pass, len(m.checks), fail))
		case pending > 0:
			checksText = checkPendStyle.Render(fmt.Sprintf("%d/%d passing, %d pending", pass, len(m.checks), pending))
		default:
			checksText = checkPassStyle.Render(fmt.Sprintf("%d/%d passing", pass, len(m.checks)))
		}
		fmt.Fprintf(&b, "  %-12s %s\n", dimTextStyle.Render("Checks"), checksText)
	}

	// Description
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render("  Description"))
	b.WriteString("\n")
	if m.pr.Body != "" {
		b.WriteString(bodyStyle.Render(m.pr.Body))
	} else {
		b.WriteString(bodyStyle.Render(dimTextStyle.Render("(no description)")))
	}

	return b.String()
}

func (m *PRDetailModel) renderChecks() string {
	if !m.checksLoaded {
		return loadingStyle.Render("Loading checks...")
	}
	if len(m.checks) == 0 {
		return bodyStyle.Render("No checks found.")
	}

	var b strings.Builder

	// Summary
	pass, fail, pending := 0, 0, 0
	for _, c := range m.checks {
		switch c.Conclusion {
		case "success":
			pass++
		case "failure", "cancelled", "timed_out", "action_required":
			fail++
		default:
			pending++
		}
	}
	fmt.Fprintf(&b, "  %d checks: ", len(m.checks))
	if pass > 0 {
		b.WriteString(checkPassStyle.Render(fmt.Sprintf("%d passed", pass)))
	}
	if fail > 0 {
		if pass > 0 {
			b.WriteString(", ")
		}
		b.WriteString(checkFailStyle.Render(fmt.Sprintf("%d failed", fail)))
	}
	if pending > 0 {
		if pass > 0 || fail > 0 {
			b.WriteString(", ")
		}
		b.WriteString(checkPendStyle.Render(fmt.Sprintf("%d pending", pending)))
	}
	b.WriteString("\n\n")

	// Individual checks - failures first, then pending, then passed
	type styledCheck struct {
		icon  string
		name  string
		style func(...string) string
		order int
	}
	var checks []styledCheck
	for _, c := range m.checks {
		var sc styledCheck
		sc.name = c.Name
		switch c.Conclusion {
		case "success":
			sc.icon = "✓"
			sc.style = checkPassStyle.Render
			sc.order = 2
		case "failure", "cancelled", "timed_out", "action_required":
			sc.icon = "✗"
			sc.style = checkFailStyle.Render
			sc.order = 0
		default:
			sc.icon = "●"
			sc.style = checkPendStyle.Render
			sc.order = 1
		}
		checks = append(checks, sc)
	}
	slices.SortFunc(checks, func(a, b styledCheck) int {
		return a.order - b.order
	})
	for _, sc := range checks {
		b.WriteString(sc.style(fmt.Sprintf("  %s %s", sc.icon, sc.name)))
		b.WriteString("\n")
	}
	return b.String()
}

func (m *PRDetailModel) renderComments() string {
	if !m.commentsLoaded {
		return loadingStyle.Render("Loading comments...")
	}
	if len(m.comments) == 0 {
		return bodyStyle.Render("No comments.")
	}

	var b strings.Builder
	for i, c := range m.comments {
		if i > 0 {
			b.WriteString(commentSeparatorStyle.Render("  " + strings.Repeat("─", 60)))
			b.WriteString("\n")
		}
		author := commentAuthorStyle.Render(c.Author)
		date := commentDateStyle.Render(c.CreatedAt.Format("Jan 02, 2006 15:04"))
		fmt.Fprintf(&b, "  %s  %s\n", author, date)
		// Indent the body
		for line := range strings.SplitSeq(c.Body, "\n") {
			fmt.Fprintf(&b, "  %s\n", line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m *PRDetailModel) renderFiles() string {
	if !m.filesLoaded {
		return loadingStyle.Render("Loading files...")
	}
	if len(m.files) == 0 {
		return bodyStyle.Render("No files changed.")
	}

	var b strings.Builder

	// Summary
	totalAdds, totalDels := 0, 0
	for _, f := range m.files {
		totalAdds += f.Additions
		totalDels += f.Deletions
	}
	fmt.Fprintf(&b, "  %d files changed, ", len(m.files))
	b.WriteString(additionsStyle.Render(fmt.Sprintf("+%d", totalAdds)))
	b.WriteString(", ")
	b.WriteString(deletionsStyle.Render(fmt.Sprintf("-%d", totalDels)))
	b.WriteString("\n\n")

	// Files with diff bars
	const maxBarLen = 20
	maxChanges := 0
	for _, f := range m.files {
		if total := f.Additions + f.Deletions; total > maxChanges {
			maxChanges = total
		}
	}

	for _, f := range m.files {
		total := f.Additions + f.Deletions
		barLen := maxBarLen
		if maxChanges > 0 {
			barLen = total * maxBarLen / maxChanges
		}
		if barLen == 0 && total > 0 {
			barLen = 1
		}

		addBar, delBar := 0, 0
		if total > 0 {
			addBar = f.Additions * barLen / total
			delBar = barLen - addBar
		}

		bar := fileBarAdditionsStyle.Render(strings.Repeat("+", addBar)) +
			fileBarDeletionsStyle.Render(strings.Repeat("-", delBar))

		adds := additionsStyle.Render(fmt.Sprintf("%+4d", f.Additions))
		dels := deletionsStyle.Render(fmt.Sprintf("%+4d", -f.Deletions))
		fmt.Fprintf(&b, "  %s %s %s %s\n", adds, dels, bar, f.Filename)
	}
	return b.String()
}
