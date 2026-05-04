package github

import (
	"context"
	"fmt"
	"strings"

	gogithub "github.com/google/go-github/v67/github"
	"github.com/jhermoso/ghtui/internal/model"
)

func (c *Client) GetIssue(ctx context.Context, owner, repo string, number int) (*model.Issue, error) {
	issue, _, err := c.REST.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}
	return toModelIssue(issue, owner, repo), nil
}

func (c *Client) CreateIssue(ctx context.Context, input model.CreateIssueInput) (*model.Issue, error) {
	req := &gogithub.IssueRequest{
		Title: gogithub.String(input.Title),
		Body:  gogithub.String(input.Body),
	}
	if len(input.Labels) > 0 {
		req.Labels = &input.Labels
	}
	issue, _, err := c.REST.Issues.Create(ctx, input.Owner, input.Repo, req)
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}
	return toModelIssue(issue, input.Owner, input.Repo), nil
}

func (c *Client) UpdateIssue(ctx context.Context, input model.UpdateIssueInput) (*model.Issue, error) {
	req := &gogithub.IssueRequest{
		Title: gogithub.String(input.Title),
		Body:  gogithub.String(input.Body),
	}
	// only send labels when explicitly provided; nil → omit (preserves existing labels),
	// empty slice → clear all labels
	if input.Labels != nil {
		req.Labels = &input.Labels
	}
	if input.State != "" {
		req.State = gogithub.String(strings.ToLower(input.State))
	}
	issue, _, err := c.REST.Issues.Edit(ctx, input.Owner, input.Repo, input.Number, req)
	if err != nil {
		return nil, fmt.Errorf("updating issue: %w", err)
	}
	return toModelIssue(issue, input.Owner, input.Repo), nil
}

func (c *Client) CloseIssue(ctx context.Context, owner, repo string, number int) error {
	_, _, err := c.REST.Issues.Edit(ctx, owner, repo, number, &gogithub.IssueRequest{
		State: gogithub.String("closed"),
	})
	return err
}

func (c *Client) SearchIssues(ctx context.Context, query string) (*model.SearchResult, error) {
	opts := &gogithub.SearchOptions{ListOptions: gogithub.ListOptions{PerPage: 50}}
	result, _, err := c.REST.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("searching issues: %w", err)
	}
	issues := make([]model.Issue, 0, len(result.Issues))
	for _, issue := range result.Issues {
		// skip PRs
		if issue.PullRequestLinks != nil {
			continue
		}
		owner, repo := parseRepoURL(issue.GetRepositoryURL())
		issues = append(issues, *toModelIssue(issue, owner, repo))
	}
	return &model.SearchResult{Issues: issues, TotalCount: result.GetTotal()}, nil
}

func toModelIssue(i *gogithub.Issue, owner, repo string) *model.Issue {
	var assignees []model.Assignee
	for _, a := range i.Assignees {
		assignees = append(assignees, model.Assignee{
			Login:     a.GetLogin(),
			AvatarURL: a.GetAvatarURL(),
		})
	}
	var labels []model.Label
	for _, l := range i.Labels {
		labels = append(labels, model.Label{
			Name:  l.GetName(),
			Color: l.GetColor(),
		})
	}
	return &model.Issue{
		ID:         fmt.Sprintf("%d", i.GetID()),
		Number:     i.GetNumber(),
		Title:      i.GetTitle(),
		Body:       i.GetBody(),
		State:      i.GetState(),
		URL:        i.GetHTMLURL(),
		Repository: repo,
		Owner:      owner,
		Assignees:  assignees,
		Labels:     labels,
		CreatedAt:  i.GetCreatedAt().Time,
		UpdatedAt:  i.GetUpdatedAt().Time,
	}
}

// parseRepoURL extracts owner/repo from https://api.github.com/repos/{owner}/{repo}
func parseRepoURL(repoURL string) (owner, repo string) {
	parts := strings.Split(repoURL, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "", ""
}
