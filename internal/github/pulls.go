package github

import (
	"context"
	"fmt"
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

// ListPRs returns open pull requests for the repo.
func (c *Client) ListPRs(ctx context.Context) ([]PR, error) {
	pulls, _, err := c.inner.PullRequests.List(ctx, c.Owner, c.Repo, &gh.PullRequestListOptions{
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

	var prs []PR
	for _, p := range pulls {
		labels := make([]string, 0, len(p.Labels))
		for _, l := range p.Labels {
			labels = append(labels, l.GetName())
		}
		prs = append(prs, PR{
			Number:    p.GetNumber(),
			Title:     p.GetTitle(),
			Author:    p.GetUser().GetLogin(),
			State:     p.GetState(),
			Body:      p.GetBody(),
			Labels:    labels,
			Draft:     p.GetDraft(),
			UpdatedAt: p.GetUpdatedAt().Time,
			HeadRef:   p.GetHead().GetRef(),
		})
	}
	return prs, nil
}

// GetPRDetail fetches full PR details including mergeable status.
func (c *Client) GetPRDetail(ctx context.Context, number int) (*PR, error) {
	p, _, err := c.inner.PullRequests.Get(ctx, c.Owner, c.Repo, number)
	if err != nil {
		return nil, err
	}

	labels := make([]string, 0, len(p.Labels))
	for _, l := range p.Labels {
		labels = append(labels, l.GetName())
	}

	pr := &PR{
		Number:    p.GetNumber(),
		Title:     p.GetTitle(),
		Author:    p.GetUser().GetLogin(),
		State:     p.GetState(),
		Body:      p.GetBody(),
		Labels:    labels,
		Mergeable: p.GetMergeable(),
		Draft:     p.GetDraft(),
		UpdatedAt: p.GetUpdatedAt().Time,
		HeadRef:   p.GetHead().GetRef(),
	}

	return pr, nil
}

// GetChecks fetches CI check runs for a PR's head ref.
func (c *Client) GetChecks(ctx context.Context, ref string) ([]Check, error) {
	result, _, err := c.inner.Checks.ListCheckRunsForRef(ctx, c.Owner, c.Repo, ref, &gh.ListCheckRunsOptions{})
	if err != nil {
		return nil, err
	}

	var checks []Check
	for _, cr := range result.CheckRuns {
		checks = append(checks, Check{
			Name:       cr.GetName(),
			Status:     cr.GetStatus(),
			Conclusion: cr.GetConclusion(),
		})
	}
	return checks, nil
}

// GetComments fetches issue comments for a PR.
func (c *Client) GetComments(ctx context.Context, number int) ([]Comment, error) {
	comments, _, err := c.inner.Issues.ListComments(ctx, c.Owner, c.Repo, number, &gh.IssueListCommentsOptions{
		Sort:      new("created"),
		Direction: new("asc"),
	})
	if err != nil {
		return nil, err
	}

	var result []Comment
	for _, cm := range comments {
		result = append(result, Comment{
			Author:    cm.GetUser().GetLogin(),
			Body:      cm.GetBody(),
			CreatedAt: cm.GetCreatedAt().Time,
		})
	}

	// Also fetch review comments
	reviews, _, err := c.inner.PullRequests.ListReviews(ctx, c.Owner, c.Repo, number, &gh.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, r := range reviews {
		if r.GetBody() != "" {
			result = append(result, Comment{
				Author:    r.GetUser().GetLogin(),
				Body:      fmt.Sprintf("[%s] %s", r.GetState(), r.GetBody()),
				CreatedAt: r.GetSubmittedAt().Time,
			})
		}
	}

	return result, nil
}

// GetChangedFiles fetches the list of files changed in a PR.
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
// It prefers merge > squash > rebase.
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

// GetReviewDecision computes the review decision for a PR based on the latest review per reviewer.
func (c *Client) GetReviewDecision(ctx context.Context, number int) (string, error) {
	reviews, _, err := c.inner.PullRequests.ListReviews(ctx, c.Owner, c.Repo, number, &gh.ListOptions{PerPage: 100})
	if err != nil {
		return "", err
	}

	// Track the latest review state per user
	latestByUser := map[string]string{}
	for _, r := range reviews {
		state := r.GetState()
		if state == "APPROVED" || state == "CHANGES_REQUESTED" {
			latestByUser[r.GetUser().GetLogin()] = state
		}
	}

	if len(latestByUser) == 0 {
		return "", nil
	}

	for _, state := range latestByUser {
		if state == "CHANGES_REQUESTED" {
			return "CHANGES_REQUESTED", nil
		}
	}

	return "APPROVED", nil
}

// ApprovePR submits an approving review with "LGTM" body.
func (c *Client) ApprovePR(ctx context.Context, number int) error {
	_, _, err := c.inner.PullRequests.CreateReview(ctx, c.Owner, c.Repo, number, &gh.PullRequestReviewRequest{
		Body:  new("LGTM"),
		Event: new("APPROVE"),
	})
	return err
}
