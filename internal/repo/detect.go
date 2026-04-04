package repo

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Detect parses the git remote origin URL to extract owner/repo.
// If flagRepo is non-empty (format "owner/repo"), it takes precedence.
func Detect(flagRepo string) (owner, repo string, err error) {
	if flagRepo != "" {
		parts := strings.SplitN(flagRepo, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid --repo format %q, expected owner/repo", flagRepo)
		}
		return parts[0], parts[1], nil
	}

	// Optimistic: try to get the remote URL directly. This avoids
	// extra subprocess calls in the common case (a git repo with an origin).
	if out, err := exec.Command("git", "remote", "get-url", "origin").Output(); err == nil {
		return parseRemoteURL(strings.TrimSpace(string(out)))
	}

	// Slow path: figure out what went wrong and return a helpful error.
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", errors.New("git is not installed — use --repo owner/repo instead")
	}

	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return "", "", errors.New("current directory is not a git repository — use --repo owner/repo instead")
	}

	return "", "", errors.New("no 'origin' remote found — use --repo owner/repo instead")
}

var (
	sshPattern   = regexp.MustCompile(`git@[^:]+:([^/]+)/(.+?)(?:\.git)?$`)
	httpsPattern = regexp.MustCompile(`https?://[^/]+/([^/]+)/(.+?)(?:\.git)?$`)
)

func parseRemoteURL(url string) (string, string, error) {
	if m := sshPattern.FindStringSubmatch(url); m != nil {
		return m[1], m[2], nil
	}
	if m := httpsPattern.FindStringSubmatch(url); m != nil {
		return m[1], m[2], nil
	}
	return "", "", fmt.Errorf("cannot parse git remote URL: %s", url)
}
