package github

import (
	"context"
	"fmt"
	"os"

	gh "github.com/google/go-github/v68/github"
)

// Client wraps the GitHub API client with owner/repo context.
type Client struct {
	inner *gh.Client
	Owner string
	Repo  string
}

// NewClient creates an authenticated GitHub client.
func NewClient(owner, repo string) (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	client := gh.NewClient(nil).WithAuthToken(token)

	return &Client{
		inner: client,
		Owner: owner,
		Repo:  repo,
	}, nil
}

// CurrentUser returns the authenticated user's login.
func (c *Client) CurrentUser(ctx context.Context) (string, error) {
	user, _, err := c.inner.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}
