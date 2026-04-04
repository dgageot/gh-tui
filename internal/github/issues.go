package github

import (
	"context"
	"time"
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

// IssueDetail fetches a single issue with its comments in one query.
func (c *Client) IssueDetail(ctx context.Context, number int) (*Issue, []IssueComment, error) {
	var result struct {
		Repository struct {
			Issue struct {
				Number    int       `json:"number"`
				Title     string    `json:"title"`
				Author    *actor    `json:"author"`
				State     string    `json:"state"`
				Body      string    `json:"body"`
				UpdatedAt time.Time `json:"updatedAt"`
				Labels struct {
					Nodes []labelNode `json:"nodes"`
				} `json:"labels"`
				Comments struct {
					TotalCount int `json:"totalCount"`
					Nodes      []struct {
						Author    *actor    `json:"author"`
						Body      string    `json:"body"`
						CreatedAt time.Time `json:"createdAt"`
					} `json:"nodes"`
				} `json:"comments"`
			} `json:"issue"`
		} `json:"repository"`
	}

	err := c.graphql(ctx, `
		query($owner: String!, $repo: String!, $number: Int!) {
			repository(owner: $owner, name: $repo) {
				issue(number: $number) {
					number title body state updatedAt
					author { login }
					labels(first: 10) { nodes { name } }
					comments(first: 100) {
						totalCount
						nodes { author { login } body createdAt }
					}
				}
			}
		}
	`, map[string]any{"owner": c.Owner, "repo": c.Repo, "number": number}, &result)
	if err != nil {
		return nil, nil, err
	}

	n := result.Repository.Issue
	issue := &Issue{
		Number:    n.Number,
		Title:     n.Title,
		Author:    n.Author.GetLogin(),
		State:     n.State,
		Body:      n.Body,
		Labels:    extractLabels(n.Labels.Nodes),
		UpdatedAt: n.UpdatedAt,
		Comments:  n.Comments.TotalCount,
	}

	var comments []IssueComment
	for _, c := range n.Comments.Nodes {
		comments = append(comments, IssueComment{
			Author:    c.Author.GetLogin(),
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}

	return issue, comments, nil
}
