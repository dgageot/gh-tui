package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#6C6C6C")
	successColor   = lipgloss.Color("#73D216")
	failureColor   = lipgloss.Color("#FF5555")
	warningColor   = lipgloss.Color("#F4BF75")
	dimColor       = lipgloss.Color("#555555")

	// App chrome
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(failureColor).
			Bold(true).
			Padding(0, 1)

	// PR list
	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(primaryColor).
				Bold(true)

	// PR detail
	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(secondaryColor)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(primaryColor).
			Bold(true).
			Underline(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1)

	bodyStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Checks
	checkPassStyle = lipgloss.NewStyle().Foreground(successColor)
	checkFailStyle = lipgloss.NewStyle().Foreground(failureColor)
	checkPendStyle = lipgloss.NewStyle().Foreground(warningColor)

	// Comments
	commentAuthorStyle = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	commentDateStyle   = lipgloss.NewStyle().Foreground(dimColor)

	// Files
	additionsStyle = lipgloss.NewStyle().Foreground(successColor)
	deletionsStyle = lipgloss.NewStyle().Foreground(failureColor)

	// Confirm dialog
	confirmStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warningColor).
			Padding(1, 3)

	// Loading
	loadingStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true).
			Padding(1, 2)
)
