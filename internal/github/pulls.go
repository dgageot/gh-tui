package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v68/github"
)

// PR represents a pull request with relevant fields.
type PR struct {
	Number         int
	Title          string
	Author         string
	State          string
	Body           string
	Labels         []string
	Mergeable      bool
	Draft          bool
	UpdatedAt      time.Time
	Checks         []Check
	HeadRef        string
	ReviewDecision string // APPROVED, CHANGES_REQUESTED, or empty
}

// Check represents a CI status check.
type Check struct {
	Name       string
	Status     string // queued, in_progress, completed
	Conclusion string // success, failure, neutral, cancelled, timed_out, action_required, skipped
}

// Comment represents a PR comment.
type Comment struct {
	Author    string
	Body      string
	CreatedAt time.Time
}

// ChangedFile represents a file changed in a PR.
type ChangedFile struct {
	Filename  string
	Status    string
	Additions int
	Deletions int
	Patch     string
}

// MainPageResult contains the result of the main page query: PRs, issues, and current user.
type MainPageResult struct {
	PRs         []PR
	Issues      []Issue
	CurrentUser string
}

// ListMainPage fetches open PRs, open issues, and the current user in a single GraphQL query.
func (c *Client) ListMainPage(ctx context.Context) (*MainPageResult, error) {
	var result struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
		Repository struct {
			PullRequests struct {
				Nodes []gqlPR `json:"nodes"`
			} `json:"pullRequests"`
			Issues struct {
				Nodes []struct {
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
					} `json:"comments"`
				} `json:"nodes"`
			} `json:"issues"`
		} `json:"repository"`
	}

	err := c.graphql(ctx, `
		query($owner: String!, $repo: String!) {
			viewer { login }
			repository(owner: $owner, name: $repo) {
				pullRequests(first: 50, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
					nodes {
						number title body state isDraft
						author { login }
						updatedAt headRefName reviewDecision
						labels(first: 10) { nodes { name } }
					}
				}
				issues(first: 50, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
					nodes {
						number title body state updatedAt
						author { login }
						labels(first: 10) { nodes { name } }
						comments { totalCount }
					}
				}
			}
		}
	`, map[string]any{"owner": c.Owner, "repo": c.Repo}, &result)
	if err != nil {
		return nil, err
	}

	var prs []PR
	for _, n := range result.Repository.PullRequests.Nodes {
		prs = append(prs, n.toPR())
	}

	var issues []Issue
	for _, n := range result.Repository.Issues.Nodes {
		issues = append(issues, Issue{
			Number:    n.Number,
			Title:     n.Title,
			Author:    n.Author.GetLogin(),
			State:     n.State,
			Body:      n.Body,
			Labels:    extractLabels(n.Labels.Nodes),
			UpdatedAt: n.UpdatedAt,
			Comments:  n.Comments.TotalCount,
		})
	}

	return &MainPageResult{
		PRs:         prs,
		Issues:      issues,
		CurrentUser: result.Viewer.Login,
	}, nil
}

// PRDetail contains the full detail of a PR including checks and comments.
type PRDetail struct {
	PR       *PR
	Checks   []Check
	Comments []Comment
}

// GetPRDetail fetches full PR details including checks, comments, and review info in a single query.
// Files with patches still require REST (GraphQL doesn't expose patch content).
func (c *Client) GetPRDetail(ctx context.Context, number int) (*PRDetail, error) {
	var result struct {
		Repository struct {
			PullRequest struct {
				gqlPR

				Mergeable string `json:"mergeable"`
				Commits   struct {
					Nodes []struct {
						Commit struct {
							StatusCheckRollup *struct {
								Contexts struct {
									Nodes []struct {
										TypeName   string `json:"__typename"`
										Name       string `json:"name"`
										Status     string `json:"status"`
										Conclusion string `json:"conclusion"`
									} `json:"nodes"`
								} `json:"contexts"`
							} `json:"statusCheckRollup"`
						} `json:"commit"`
					} `json:"nodes"`
				} `json:"commits"`
				Comments struct {
					Nodes []struct {
						Author    *actor    `json:"author"`
						Body      string    `json:"body"`
						CreatedAt time.Time `json:"createdAt"`
					} `json:"nodes"`
				} `json:"comments"`
				Reviews struct {
					Nodes []struct {
						Author      *actor    `json:"author"`
						Body        string    `json:"body"`
						State       string    `json:"state"`
						SubmittedAt time.Time `json:"submittedAt"`
					} `json:"nodes"`
				} `json:"reviews"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	err := c.graphql(ctx, `
		query($owner: String!, $repo: String!, $number: Int!) {
			repository(owner: $owner, name: $repo) {
				pullRequest(number: $number) {
					number title body state isDraft
					author { login }
					updatedAt headRefName reviewDecision
					mergeable
					labels(first: 10) { nodes { name } }
					commits(last: 1) {
						nodes {
							commit {
								statusCheckRollup {
									contexts(first: 50) {
										nodes {
											__typename
											... on CheckRun { name status conclusion }
										}
									}
								}
							}
						}
					}
					comments(first: 100) {
						nodes { author { login } body createdAt }
					}
					reviews(first: 100) {
						nodes { author { login } body state submittedAt }
					}
				}
			}
		}
	`, map[string]any{"owner": c.Owner, "repo": c.Repo, "number": number}, &result)
	if err != nil {
		return nil, err
	}

	p := result.Repository.PullRequest
	pr := p.toPR()
	pr.Mergeable = strings.EqualFold(p.Mergeable, "MERGEABLE")

	// Extract checks
	var checks []Check
	if len(p.Commits.Nodes) > 0 {
		commit := p.Commits.Nodes[0].Commit
		if commit.StatusCheckRollup != nil {
			for _, c := range commit.StatusCheckRollup.Contexts.Nodes {
				if c.TypeName == "CheckRun" {
					checks = append(checks, Check{
						Name:       c.Name,
						Status:     strings.ToLower(c.Status),
						Conclusion: strings.ToLower(c.Conclusion),
					})
				}
			}
		}
	}

	// Extract comments (issue comments + review bodies)
	var comments []Comment
	for _, cm := range p.Comments.Nodes {
		comments = append(comments, Comment{
			Author:    cm.Author.GetLogin(),
			Body:      cm.Body,
			CreatedAt: cm.CreatedAt,
		})
	}
	for _, r := range p.Reviews.Nodes {
		if r.Body != "" {
			comments = append(comments, Comment{
				Author:    r.Author.GetLogin(),
				Body:      fmt.Sprintf("[%s] %s", r.State, r.Body),
				CreatedAt: r.SubmittedAt,
			})
		}
	}

	return &PRDetail{
		PR:       &pr,
		Checks:   checks,
		Comments: comments,
	}, nil
}

// GetChangedFiles fetches the list of files changed in a PR (REST, for patch content).
func (c *Client) GetChangedFiles(ctx context.Context, number int) ([]ChangedFile, error) {
	files, _, err := c.inner.PullRequests.ListFiles(ctx, c.Owner, c.Repo, number, &gh.ListOptions{PerPage: 100})
	if err != nil {
		return nil, err
	}

	var result []ChangedFile
	for _, f := range files {
		result = append(result, ChangedFile{
			Filename:  f.GetFilename(),
			Status:    f.GetStatus(),
			Additions: f.GetAdditions(),
			Deletions: f.GetDeletions(),
			Patch:     f.GetPatch(),
		})
	}
	return result, nil
}

// MergePR merges a pull request using the first allowed merge method.
func (c *Client) MergePR(ctx context.Context, number int) error {
	method, err := c.preferredMergeMethod(ctx)
	if err != nil {
		return err
	}

	_, _, err = c.inner.PullRequests.Merge(ctx, c.Owner, c.Repo, number, "", &gh.PullRequestOptions{
		MergeMethod: method,
	})
	return err
}

// preferredMergeMethod returns the preferred merge method allowed by the repository.
func (c *Client) preferredMergeMethod(ctx context.Context) (string, error) {
	repo, _, err := c.inner.Repositories.Get(ctx, c.Owner, c.Repo)
	if err != nil {
		return "", fmt.Errorf("failed to get repository settings: %w", err)
	}

	switch {
	case repo.GetAllowMergeCommit():
		return "merge", nil
	case repo.GetAllowSquashMerge():
		return "squash", nil
	case repo.GetAllowRebaseMerge():
		return "rebase", nil
	default:
		return "merge", nil
	}
}

// ApprovePR submits an approving review with "LGTM" body.
func (c *Client) ApprovePR(ctx context.Context, number int) error {
	_, _, err := c.inner.PullRequests.CreateReview(ctx, c.Owner, c.Repo, number, &gh.PullRequestReviewRequest{
		Body:  gh.Ptr("LGTM"),
		Event: gh.Ptr("APPROVE"),
	})
	return err
}

// gqlPR is the shared GraphQL PR node shape.
type gqlPR struct {
	Number         int       `json:"number"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	State          string    `json:"state"`
	IsDraft        bool      `json:"isDraft"`
	Author         *actor    `json:"author"`
	UpdatedAt      time.Time `json:"updatedAt"`
	HeadRefName    string    `json:"headRefName"`
	ReviewDecision string    `json:"reviewDecision"`
	Labels struct {
		Nodes []labelNode `json:"nodes"`
	} `json:"labels"`
}

func (g *gqlPR) toPR() PR {
	return PR{
		Number:         g.Number,
		Title:          g.Title,
		Author:         g.Author.GetLogin(),
		State:          strings.ToLower(g.State),
		Body:           g.Body,
		Labels:         extractLabels(g.Labels.Nodes),
		Draft:          g.IsDraft,
		UpdatedAt:      g.UpdatedAt,
		HeadRef:        g.HeadRefName,
		ReviewDecision: g.ReviewDecision,
	}
}
