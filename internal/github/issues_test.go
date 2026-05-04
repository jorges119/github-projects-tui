package github

import (
	"testing"
	"time"

	gogithub "github.com/google/go-github/v67/github"
	"github.com/jhermoso/ghtui/internal/model"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		url       string
		wantOwner string
		wantRepo  string
	}{
		{
			url:       "https://api.github.com/repos/octocat/hello-world",
			wantOwner: "octocat",
			wantRepo:  "hello-world",
		},
		{
			url:       "https://api.github.com/repos/my-org/my-repo",
			wantOwner: "my-org",
			wantRepo:  "my-repo",
		},
		{
			url:       "",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			url:       "single",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tc := range tests {
		owner, repo := parseRepoURL(tc.url)
		if owner != tc.wantOwner || repo != tc.wantRepo {
			t.Errorf("parseRepoURL(%q) = (%q, %q), want (%q, %q)",
				tc.url, owner, repo, tc.wantOwner, tc.wantRepo)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestToModelIssue_Basic(t *testing.T) {
	ts := gogithub.Timestamp{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)}
	i := &gogithub.Issue{
		ID:      ptr(int64(42)),
		Number:  ptr(7),
		Title:   ptr("Fix the bug"),
		Body:    ptr("Description here"),
		State:   ptr("open"),
		HTMLURL: ptr("https://github.com/owner/repo/issues/7"),
		CreatedAt: &ts,
		UpdatedAt: &ts,
	}

	got := toModelIssue(i, "owner", "repo")

	if got.ID != "42" {
		t.Errorf("ID: got %q, want %q", got.ID, "42")
	}
	if got.Number != 7 {
		t.Errorf("Number: got %d, want 7", got.Number)
	}
	if got.Title != "Fix the bug" {
		t.Errorf("Title: got %q, want %q", got.Title, "Fix the bug")
	}
	if got.Body != "Description here" {
		t.Errorf("Body: got %q", got.Body)
	}
	if got.State != "open" {
		t.Errorf("State: got %q, want open", got.State)
	}
	if got.URL != "https://github.com/owner/repo/issues/7" {
		t.Errorf("URL: got %q", got.URL)
	}
	if got.Owner != "owner" || got.Repository != "repo" {
		t.Errorf("Owner/Repo: got %q/%q", got.Owner, got.Repository)
	}
	if !got.CreatedAt.Equal(ts.Time) {
		t.Errorf("CreatedAt: got %v, want %v", got.CreatedAt, ts.Time)
	}
}

func TestToModelIssue_AssigneesAndLabels(t *testing.T) {
	i := &gogithub.Issue{
		ID:    ptr(int64(1)),
		Title: ptr("Test"),
		Assignees: []*gogithub.User{
			{Login: ptr("alice"), AvatarURL: ptr("https://example.com/alice.png")},
			{Login: ptr("bob"), AvatarURL: ptr("https://example.com/bob.png")},
		},
		Labels: []*gogithub.Label{
			{Name: ptr("bug"), Color: ptr("d73a4a")},
			{Name: ptr("enhancement"), Color: ptr("a2eeef")},
		},
	}

	got := toModelIssue(i, "o", "r")

	if len(got.Assignees) != 2 {
		t.Fatalf("Assignees len: got %d, want 2", len(got.Assignees))
	}
	if got.Assignees[0] != (model.Assignee{Login: "alice", AvatarURL: "https://example.com/alice.png"}) {
		t.Errorf("Assignees[0]: got %+v", got.Assignees[0])
	}
	if got.Assignees[1].Login != "bob" {
		t.Errorf("Assignees[1].Login: got %q", got.Assignees[1].Login)
	}

	if len(got.Labels) != 2 {
		t.Fatalf("Labels len: got %d, want 2", len(got.Labels))
	}
	if got.Labels[0] != (model.Label{Name: "bug", Color: "d73a4a"}) {
		t.Errorf("Labels[0]: got %+v", got.Labels[0])
	}
}

func TestToModelIssue_NilAssigneesAndLabels(t *testing.T) {
	i := &gogithub.Issue{
		ID:    ptr(int64(1)),
		Title: ptr("Empty"),
	}
	got := toModelIssue(i, "o", "r")
	if len(got.Assignees) != 0 {
		t.Errorf("expected no assignees, got %d", len(got.Assignees))
	}
	if len(got.Labels) != 0 {
		t.Errorf("expected no labels, got %d", len(got.Labels))
	}
}
