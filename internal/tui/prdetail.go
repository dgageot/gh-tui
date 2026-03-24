package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

	prLoaded       bool
	checksLoaded   bool
	commentsLoaded bool
	filesLoaded    bool

	selectedFile int
	viewingDiff  bool
}

func NewPRDetailModel() PRDetailModel {
	return PRDetailModel{}
}

func (m *PRDetailModel) Init() tea.Cmd {
	return nil
}

func (m *PRDetailModel) Update(msg tea.Msg) (PRDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if tab, ok := m.tabAtPosition(msg.X, msg.Y); ok {
				m.tab = tab
				if tab != TabFiles {
					m.viewingDiff = false
				}
				m.updateViewport()
				return *m, nil
			}
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return *m, cmd

	case tea.KeyMsg:
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

		if m.viewingDiff {
			switch msg.String() {
			case "esc":
				m.viewingDiff = false
				m.updateViewport()
				return *m, nil
			default:
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return *m, cmd
			}
		}

		if m.tab == TabFiles && m.filesLoaded && len(m.files) > 0 {
			switch msg.String() {
			case "up", "k":
				if m.selectedFile > 0 {
					m.selectedFile--
					m.updateViewport()
				}
				return *m, nil
			case "down", "j":
				if m.selectedFile < len(m.files)-1 {
					m.selectedFile++
					m.updateViewport()
				}
				return *m, nil
			case "enter":
				m.viewingDiff = true
				m.updateViewport()
				return *m, nil
			}
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
	prClickedMsg      struct{}
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

func (m *PRDetailModel) helpBar() string {
	if m.viewingDiff {
		return formatHelpKeys("esc", "back to files")
	}
	pairs := []string{"⇥", "switch tab"}
	if m.tab == TabFiles && m.filesLoaded && len(m.files) > 0 {
		pairs = append(pairs, "↑↓", "navigate", "⏎", "view diff")
	}
	if !m.isOwnPR() {
		pairs = append(pairs, "L", "LGTM")
	}
	if m.isMergeable() {
		pairs = append(pairs, "M", "merge")
	}
	pairs = append(pairs, "b", "browser", "esc", "back")
	return formatHelpKeys(pairs...)
}

func (m *PRDetailModel) loading() bool {
	return !m.prLoaded && m.err == nil
}

func (m *PRDetailModel) tabAtPosition(x, y int) (DetailTab, bool) {
	if y != 1 {
		return 0, false
	}

	pos := 0
	for i, name := range tabNames {
		var rendered string
		if DetailTab(i) == m.tab {
			rendered = activeTabStyle.Render(name)
		} else {
			rendered = tabStyle.Render(name)
		}
		tabWidth := lipgloss.Width(rendered)
		if x >= pos && x < pos+tabWidth {
			return DetailTab(i), true
		}
		pos += tabWidth + 1 // +1 for the space separator
	}
	return 0, false
}

func (m *PRDetailModel) View() string {
	var b strings.Builder

	if m.loading() {
		b.WriteString(loadingStyle.Render("  Loading PR details…"))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render("  Error: " + m.err.Error()))
		return b.String()
	}

	if m.pr == nil {
		return ""
	}

	// Title with number — truncate to fit width
	number := titleStyle.Render(fmt.Sprintf(" #%d", m.pr.Number))
	title := titleStyle.Render(m.pr.Title)
	titleLine := " " + number + "  " + title
	if m.width > 0 {
		titleLine = truncateToWidth(titleLine, m.width)
	}
	b.WriteString(titleLine)
	b.WriteString("\n")

	// Tab bar
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
		prompt := fmt.Sprintf("  Confirm %s? (y/n)", m.confirm)
		b.WriteString(confirmStyle.Render(prompt))
	} else {
		b.WriteString(m.helpBar())
	}

	return b.String()
}

func (m *PRDetailModel) renderDiff() string {
	if m.selectedFile < 0 || m.selectedFile >= len(m.files) {
		return ""
	}

	f := m.files[m.selectedFile]
	var b strings.Builder

	b.WriteString(sectionTitleStyle.Render("  📄 " + f.Filename))
	b.WriteString("\n")
	adds := additionsStyle.Render(fmt.Sprintf("+%d", f.Additions))
	dels := deletionsStyle.Render(fmt.Sprintf("-%d", f.Deletions))
	fmt.Fprintf(&b, "  %s %s\n\n", adds, dels)

	if f.Patch == "" {
		b.WriteString(dimTextStyle.Render("  (binary file or no diff available)"))
		return b.String()
	}

	for line := range strings.SplitSeq(f.Patch, "\n") {
		switch {
		case strings.HasPrefix(line, "+"):
			b.WriteString(diffAddStyle.Render("  " + line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(diffDelStyle.Render("  " + line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString("\n")
			b.WriteString(diffHunkStyle.Render("  " + line))
		default:
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
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
		if m.viewingDiff {
			return m.renderDiff()
		}
		return m.renderFiles()
	default:
		return ""
	}
}

func (m *PRDetailModel) renderOverview() string {
	var b strings.Builder

	b.WriteString("\n")

	// State badge + branch
	badge := stateBadge(m.pr.State, m.pr.Draft)
	fmt.Fprintf(&b, "  %s  %s → main\n\n", badge, branchStyle.Render(m.pr.HeadRef))

	// Info grid
	fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Author"), commentAuthorStyle.Render(m.pr.Author))
	fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Updated"), m.pr.UpdatedAt.Format("Jan 02, 2006 15:04"))

	// Mergeable
	if m.pr.Mergeable {
		fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Mergeable"), mergeableYesStyle.Render("✓ Yes"))
	} else {
		fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Mergeable"), mergeableNoStyle.Render("✗ No"))
	}

	// Review
	switch m.pr.ReviewDecision {
	case "APPROVED":
		fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Review"), reviewApprovedStyle.Render("✓ Approved"))
	case "CHANGES_REQUESTED":
		fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Review"), reviewChangesStyle.Render("✗ Changes requested"))
	default:
		fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Review"), reviewPendingStyle.Render("⏳ Pending"))
	}

	// Labels
	if len(m.pr.Labels) > 0 {
		var labels []string
		for _, l := range m.pr.Labels {
			labels = append(labels, labelStyle.Render(l))
		}
		fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Labels"), strings.Join(labels, " "))
	}

	// Checks summary inline
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
		fmt.Fprintf(&b, "  %-14s %s\n", dimTextStyle.Render("Checks"), checksText)
	}

	// Description
	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render("  Description"))
	b.WriteString("\n")
	if m.pr.Body != "" {
		b.WriteString(bodyStyle.Render(m.pr.Body))
	} else {
		b.WriteString(bodyStyle.Render(dimTextStyle.Render("No description provided.")))
	}

	return b.String()
}

func (m *PRDetailModel) renderChecks() string {
	if !m.checksLoaded {
		return loadingStyle.Render("  Loading checks…")
	}
	if len(m.checks) == 0 {
		return bodyStyle.Render(dimTextStyle.Render("No checks found."))
	}

	var b strings.Builder

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

	b.WriteString("\n")
	fmt.Fprintf(&b, "  %d checks: ", len(m.checks))
	if pass > 0 {
		b.WriteString(checkPassStyle.Render(fmt.Sprintf("%d passed", pass)))
	}
	if fail > 0 {
		if pass > 0 {
			b.WriteString(dimTextStyle.Render(", "))
		}
		b.WriteString(checkFailStyle.Render(fmt.Sprintf("%d failed", fail)))
	}
	if pending > 0 {
		if pass > 0 || fail > 0 {
			b.WriteString(dimTextStyle.Render(", "))
		}
		b.WriteString(checkPendStyle.Render(fmt.Sprintf("%d pending", pending)))
	}
	b.WriteString("\n\n")

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
		return loadingStyle.Render("  Loading comments…")
	}
	if len(m.comments) == 0 {
		return bodyStyle.Render(dimTextStyle.Render("No comments yet."))
	}

	var b strings.Builder
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
	return b.String()
}

func (m *PRDetailModel) renderFiles() string {
	if !m.filesLoaded {
		return loadingStyle.Render("  Loading files…")
	}
	if len(m.files) == 0 {
		return bodyStyle.Render(dimTextStyle.Render("No files changed."))
	}

	var b strings.Builder

	totalAdds, totalDels := 0, 0
	for _, f := range m.files {
		totalAdds += f.Additions
		totalDels += f.Deletions
	}

	b.WriteString("\n")
	fmt.Fprintf(&b, "  %d files changed  ", len(m.files))
	b.WriteString(additionsStyle.Render(fmt.Sprintf("+%d", totalAdds)))
	b.WriteString(dimTextStyle.Render(" / "))
	b.WriteString(deletionsStyle.Render(fmt.Sprintf("-%d", totalDels)))
	b.WriteString("\n\n")

	const maxBarLen = 20
	maxChanges := 0
	for _, f := range m.files {
		if total := f.Additions + f.Deletions; total > maxChanges {
			maxChanges = total
		}
	}

	for i, f := range m.files {
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

		bar := fileBarAdditionsStyle.Render(strings.Repeat("█", addBar)) +
			fileBarDeletionsStyle.Render(strings.Repeat("█", delBar)) +
			dimTextStyle.Render(strings.Repeat("░", maxBarLen-addBar-delBar))

		adds := additionsStyle.Render(fmt.Sprintf("%+4d", f.Additions))
		dels := deletionsStyle.Render(fmt.Sprintf("%+4d", -f.Deletions))

		cursor := "  "
		if i == m.selectedFile {
			cursor = titleStyle.Render("▸ ")
		}
		fmt.Fprintf(&b, "%s%s %s %s %s\n", cursor, adds, dels, bar, f.Filename)
	}
	return b.String()
}
