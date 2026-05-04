package model

import "time"

type AuthMethod int

const (
	AuthPAT AuthMethod = iota
	AuthDeviceFlow
)

type User struct {
	Login     string
	Name      string
	AvatarURL string
}

type Project struct {
	ID     string
	Number int
	Title  string
	URL    string
	Owner  string
}

type Iteration struct {
	ID        string
	Title     string
	StartDate string
	Duration  int
}

type Label struct {
	Name  string
	Color string
}

type Assignee struct {
	Login     string
	AvatarURL string
}

type Issue struct {
	ID         string
	NodeID     string
	Number     int
	Title      string
	Body       string
	State      string
	URL        string
	Repository string
	Owner      string
	Assignees  []Assignee
	Labels     []Label
	CreatedAt  time.Time
	UpdatedAt  time.Time
	IterationID string
}

type ProjectItem struct {
	ID          string
	Issue       Issue
	IterationID string
	Status      string
}

type CreateIssueInput struct {
	Owner  string
	Repo   string
	Title  string
	Body   string
	Labels []string
}

type UpdateIssueInput struct {
	Owner  string
	Repo   string
	Number int
	Title  string
	Body   string
	State  string
	Labels []string
}

type SearchResult struct {
	Issues     []Issue
	TotalCount int
}

type StatusOption struct {
	ID   string
	Name string
}

type ProjectMeta struct {
	Iterations    []Iteration
	StatusFieldID string
	StatusOptions []StatusOption
}
