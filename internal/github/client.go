package github

import (
	"errors"
	"os"

	gh "github.com/google/go-github/v68/github"
)

// Client wraps the GitHub API client with owner/repo context.
type Client struct {
	inner *gh.Client
	token string
	Owner string
	Repo  string
}

// NewClient creates an authenticated GitHub client.
func NewClient(owner, repo string) (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, errors.New("GITHUB_TOKEN environment variable is required")
	}

	client := gh.NewClient(nil).WithAuthToken(token)

	return &Client{
		inner: client,
		token: token,
		Owner: owner,
		Repo:  repo,
	}, nil
}
