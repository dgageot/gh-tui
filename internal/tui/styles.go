package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	// Color palette — soft, cohesive dark-theme inspired by Catppuccin Mocha
	accentColor  = lipgloss.Color("#CBA6F7") // mauve
	subtleColor  = lipgloss.Color("#6C7086") // overlay0
	textColor    = lipgloss.Color("#CDD6F4") // text
	subTextColor = lipgloss.Color("#A6ADC8") // subtext0
	surfaceColor = lipgloss.Color("#313244") // surface0
	baseColor    = lipgloss.Color("#1E1E2E") // base
	greenColor   = lipgloss.Color("#A6E3A1") // green
	redColor     = lipgloss.Color("#F38BA8") // red
	yellowColor  = lipgloss.Color("#F9E2AF") // yellow
	blueColor    = lipgloss.Color("#89B4FA") // blue
	peachColor   = lipgloss.Color("#FAB387") // peach
	tealColor    = lipgloss.Color("#94E2D5") // teal

	// ── App chrome ──────────────────────────────────────────────

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	paneTitleBarStyle = lipgloss.NewStyle().
				Background(surfaceColor).
				Padding(0, 1)

	paneTitleFocusedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				Background(surfaceColor)

	paneTitleDimStyle = lipgloss.NewStyle().
				Foreground(subtleColor).
				Background(surfaceColor)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(subTextColor).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(redColor).
			Bold(true).
			Padding(0, 1)

	separatorStyle = lipgloss.NewStyle().
			Foreground(surfaceColor)

	// ── Tables ──────────────────────────────────────────────────

	tableHeaderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(surfaceColor).
				BorderBottom(true).
				Foreground(subTextColor).
				Bold(true).
				Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(baseColor).
				Background(accentColor).
				Bold(true)

	// ── PR detail tabs ──────────────────────────────────────────

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(subtleColor)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(accentColor).
			Bold(true).
			Underline(true)

	// ── Labels ──────────────────────────────────────────────────

	labelStyle = lipgloss.NewStyle().
			Foreground(baseColor).
			Background(accentColor).
			Padding(0, 1).
			Bold(true)

	bodyStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(textColor)

	// ── Checks ──────────────────────────────────────────────────

	checkPassStyle = lipgloss.NewStyle().Foreground(greenColor)
	checkFailStyle = lipgloss.NewStyle().Foreground(redColor)
	checkPendStyle = lipgloss.NewStyle().Foreground(yellowColor)

	// ── Comments ────────────────────────────────────────────────

	commentAuthorStyle    = lipgloss.NewStyle().Bold(true).Foreground(blueColor)
	commentDateStyle      = lipgloss.NewStyle().Foreground(subtleColor)
	commentSeparatorStyle = lipgloss.NewStyle().Foreground(surfaceColor)

	// ── Files ───────────────────────────────────────────────────

	additionsStyle        = lipgloss.NewStyle().Foreground(greenColor)
	deletionsStyle        = lipgloss.NewStyle().Foreground(redColor)
	fileBarAdditionsStyle = lipgloss.NewStyle().Foreground(greenColor)
	fileBarDeletionsStyle = lipgloss.NewStyle().Foreground(redColor)

	// ── Detail sections ─────────────────────────────────────────

	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				MarginTop(1).
				MarginBottom(1)

	dimTextStyle = lipgloss.NewStyle().Foreground(subtleColor)

	branchStyle = lipgloss.NewStyle().
			Foreground(tealColor).
			Background(surfaceColor).
			Padding(0, 1)

	reviewApprovedStyle = lipgloss.NewStyle().Foreground(greenColor).Bold(true)
	reviewChangesStyle  = lipgloss.NewStyle().Foreground(redColor).Bold(true)
	reviewPendingStyle  = lipgloss.NewStyle().Foreground(yellowColor)
	mergeableYesStyle   = lipgloss.NewStyle().Foreground(greenColor)
	mergeableNoStyle    = lipgloss.NewStyle().Foreground(redColor)

	// ── Diff ────────────────────────────────────────────────────

	diffAddStyle  = lipgloss.NewStyle().Foreground(greenColor)
	diffDelStyle  = lipgloss.NewStyle().Foreground(redColor)
	diffHunkStyle = lipgloss.NewStyle().Foreground(blueColor).Bold(true)

	// ── Confirm dialog ──────────────────────────────────────────

	confirmStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(peachColor).
			Foreground(textColor).
			Padding(1, 3)

	// ── Loading ─────────────────────────────────────────────────

	loadingStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Italic(true).
			Padding(1, 2)

	// ── State badges ────────────────────────────────────────────

	stateBadgeOpen = lipgloss.NewStyle().
			Foreground(baseColor).
			Background(greenColor).
			Padding(0, 1).
			Bold(true)

	stateBadgeDraft = lipgloss.NewStyle().
			Foreground(baseColor).
			Background(subtleColor).
			Padding(0, 1)

	stateBadgeClosed = lipgloss.NewStyle().
				Foreground(baseColor).
				Background(redColor).
				Padding(0, 1).
				Bold(true)

	stateBadgeMerged = lipgloss.NewStyle().
				Foreground(baseColor).
				Background(accentColor).
				Padding(0, 1).
				Bold(true)
)

// formatHelpKeys renders key bindings as styled "key:desc" pairs.
func formatHelpKeys(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, helpKeyStyle.Render(pairs[i])+helpDescStyle.Render(":"+pairs[i+1]))
	}
	var b strings.Builder
	b.WriteString(" ")
	for i, p := range parts {
		if i > 0 {
			b.WriteString(dimTextStyle.Render("  "))
		}
		b.WriteString(p)
	}
	return b.String()
}

// paneTitleBar renders a full-width title row with a filled background.
func paneTitleBar(title string, focused bool, width int, extra string) string {
	var label string
	if focused {
		label = paneTitleFocusedStyle.Render(title)
	} else {
		label = paneTitleDimStyle.Render(title)
	}
	if extra != "" {
		label += paneTitleDimStyle.Render("  ") + extra
	}
	return paneTitleBarStyle.Width(width).Render(label)
}

// stateBadge returns a styled badge for a PR/issue state.
func stateBadge(state string, draft bool) string {
	if draft {
		return stateBadgeDraft.Render("draft")
	}
	switch state {
	case "closed":
		return stateBadgeClosed.Render("closed")
	case "merged":
		return stateBadgeMerged.Render("merged")
	default:
		return stateBadgeOpen.Render("open")
	}
}

// truncateToWidth truncates an ANSI-styled string to fit within maxWidth.
func truncateToWidth(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	return ansi.Truncate(s, maxWidth-1, "…")
}
