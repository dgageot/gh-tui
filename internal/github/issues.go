package github

import (
	"context"
	"time"

	gh "github.com/google/go-github/v68/github"
)

// Issue represents a GitHub issue.
type Issue struct {
	Number    int
	Title     string
	Author    string
	State     string
	Body      string
	Labels    []string
	UpdatedAt time.Time
	Comments  int
}

// IssueComment represents a comment on an issue.
type IssueComment struct {
	Author    string
	Body      string
	CreatedAt time.Time
}

// ListIssues returns open issues (excluding pull requests) for the repo.
func (c *Client) ListIssues(ctx context.Context) ([]Issue, error) {
	issues, _, err := c.inner.Issues.ListByRepo(ctx, c.Owner, c.Repo, &gh.IssueListByRepoOptions{
		State:     "open",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: gh.ListOptions{
			PerPage: 50,
		},
	})
	if err != nil {
		return nil, err
	}

	var result []Issue
	for _, i := range issues {
		if i.IsPullRequest() {
			continue
		}

		labels := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			labels = append(labels, l.GetName())
		}
		result = append(result, Issue{
			Number:    i.GetNumber(),
			Title:     i.GetTitle(),
			Author:    i.GetUser().GetLogin(),
			State:     i.GetState(),
			Body:      i.GetBody(),
			Labels:    labels,
			UpdatedAt: i.GetUpdatedAt().Time,
			Comments:  i.GetComments(),
		})
	}
	return result, nil
}

// GetIssue fetches a single issue by number.
func (c *Client) GetIssue(ctx context.Context, number int) (*Issue, error) {
	i, _, err := c.inner.Issues.Get(ctx, c.Owner, c.Repo, number)
	if err != nil {
		return nil, err
	}

	labels := make([]string, 0, len(i.Labels))
	for _, l := range i.Labels {
		labels = append(labels, l.GetName())
	}

	return &Issue{
		Number:    i.GetNumber(),
		Title:     i.GetTitle(),
		Author:    i.GetUser().GetLogin(),
		State:     i.GetState(),
		Body:      i.GetBody(),
		Labels:    labels,
		UpdatedAt: i.GetUpdatedAt().Time,
		Comments:  i.GetComments(),
	}, nil
}

// GetIssueComments fetches comments for an issue.
func (c *Client) GetIssueComments(ctx context.Context, number int) ([]IssueComment, error) {
	comments, _, err := c.inner.Issues.ListComments(ctx, c.Owner, c.Repo, number, &gh.IssueListCommentsOptions{
		Sort:      new("created"),
		Direction: new("asc"),
	})
	if err != nil {
		return nil, err
	}

	var result []IssueComment
	for _, c := range comments {
		result = append(result, IssueComment{
			Author:    c.GetUser().GetLogin(),
			Body:      c.GetBody(),
			CreatedAt: c.GetCreatedAt().Time,
		})
	}
	return result, nil
}
