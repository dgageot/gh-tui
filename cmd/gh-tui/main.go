package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	gh "github.com/dgageot/gh-tui/internal/github"
	"github.com/dgageot/gh-tui/internal/repo"
	"github.com/dgageot/gh-tui/internal/tui"
)

func main() {
	repoFlag := flag.String("repo", "", "GitHub repository in owner/repo format")
	flag.Parse()

	owner, repoName, err := repo.Detect(*repoFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	client, err := gh.NewClient(owner, repoName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		tui.NewAppModel(client),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
