package views

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	gh "github.com/jhermoso/ghtui/internal/github"
	"github.com/jhermoso/ghtui/internal/model"
)

// ─── Utility functions ────────────────────────────────────────────────────────

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"todo", "#45475a"},
		{"TODO", "#45475a"},
		{"backlog", "#45475a"},
		{"in progress", "#fab387"},
		{"In Progress", "#fab387"},
		{"done", "#a6e3a1"},
		{"closed", "#a6e3a1"},
		{"DONE", "#a6e3a1"},
		{"review", "#89b4fa"},
		{"", "#89b4fa"},
		{"unknown-status", "#89b4fa"},
	}
	for _, tc := range tests {
		got := statusColor(tc.status)
		if got != tc.want {
			t.Errorf("statusColor(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestAnyContains(t *testing.T) {
	logins := []model.Assignee{
		{Login: "alice"},
		{Login: "bob"},
	}
	extract := func(a model.Assignee) string { return a.Login }

	if !anyContains(logins, extract, "ali") {
		t.Error("expected match for 'ali'")
	}
	if !anyContains(logins, extract, "bob") {
		t.Error("expected lowercase match for 'bob'")
	}
	if anyContains(logins, extract, "charlie") {
		t.Error("unexpected match for 'charlie'")
	}
	if anyContains([]model.Assignee{}, extract, "ali") {
		t.Error("unexpected match on empty slice")
	}
}

func TestClamp(t *testing.T) {
	tests := []struct{ v, lo, hi, want int }{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 0, 0},
		{3, 3, 3, 3},
	}
	for _, tc := range tests {
		got := clamp(tc.v, tc.lo, tc.hi)
		if got != tc.want {
			t.Errorf("clamp(%d,%d,%d) = %d, want %d", tc.v, tc.lo, tc.hi, got, tc.want)
		}
	}
}

func TestImin(t *testing.T) {
	if imin(3, 7) != 3 {
		t.Error("imin(3,7) should be 3")
	}
	if imin(7, 3) != 3 {
		t.Error("imin(7,3) should be 3")
	}
	if imin(5, 5) != 5 {
		t.Error("imin(5,5) should be 5")
	}
}

func TestImax(t *testing.T) {
	if imax(3, 7) != 7 {
		t.Error("imax(3,7) should be 7")
	}
	if imax(7, 3) != 7 {
		t.Error("imax(7,3) should be 7")
	}
	if imax(5, 5) != 5 {
		t.Error("imax(5,5) should be 5")
	}
}

// ─── Dashboard helpers ────────────────────────────────────────────────────────

func newTestDashboard(items []model.ProjectItem) Dashboard {
	d := Dashboard{
		allItems:      items,
		orgOptions:    []string{"me", "myorg"},
		allProjects:   nil,
		projects:      nil,
		iterations:    []model.Iteration{{ID: "", Title: "All"}, {ID: gh.BacklogIterationID, Title: "Backlog"}},
		statusOptions: []model.StatusOption{
			{ID: "s1", Name: "Todo"},
			{ID: "s2", Name: "In Progress"},
			{ID: "s3", Name: "Done"},
		},
	}
	d.assigneeInput = textinput.New()
	d.labelInput = textinput.New()
	d.titleInput = textinput.New()
	d.descInput = textinput.New()
	return d
}

func items(issues ...model.Issue) []model.ProjectItem {
	out := make([]model.ProjectItem, len(issues))
	for i, iss := range issues {
		out[i] = model.ProjectItem{ID: iss.ID, Issue: iss}
	}
	return out
}

// ─── applyFilters ─────────────────────────────────────────────────────────────

func TestApplyFilters_NoFilter_ReturnsAll(t *testing.T) {
	d := newTestDashboard(items(
		model.Issue{ID: "1", Title: "Alpha", State: "open"},
		model.Issue{ID: "2", Title: "Beta", State: "closed"},
	))
	d.applyFilters()
	if len(d.filtered) != 2 {
		t.Errorf("expected 2 items, got %d", len(d.filtered))
	}
}

func TestApplyFilters_Empty(t *testing.T) {
	d := newTestDashboard(nil)
	d.applyFilters()
	if len(d.filtered) != 0 {
		t.Errorf("expected 0 items, got %d", len(d.filtered))
	}
}

func TestApplyFilters_TitleFilter(t *testing.T) {
	d := newTestDashboard(items(
		model.Issue{ID: "1", Title: "Fix login bug", State: "open"},
		model.Issue{ID: "2", Title: "Add dark mode", State: "open"},
		model.Issue{ID: "3", Title: "LOGIN timeout", State: "open"},
	))
	d.titleInput.SetValue("login")
	d.applyFilters()

	if len(d.filtered) != 2 {
		t.Errorf("expected 2 items matching 'login', got %d", len(d.filtered))
	}
	for _, it := range d.filtered {
		if it.ID != "1" && it.ID != "3" {
			t.Errorf("unexpected item ID %q in filtered result", it.ID)
		}
	}
}

func TestApplyFilters_DescFilter(t *testing.T) {
	d := newTestDashboard(items(
		model.Issue{ID: "1", Title: "A", Body: "contains keyword here", State: "open"},
		model.Issue{ID: "2", Title: "B", Body: "nothing relevant", State: "open"},
	))
	d.descInput.SetValue("Keyword")
	d.applyFilters()

	if len(d.filtered) != 1 || d.filtered[0].ID != "1" {
		t.Errorf("expected only item 1, got %+v", d.filtered)
	}
}

func TestApplyFilters_AssigneeFilter(t *testing.T) {
	d := newTestDashboard(items(
		model.Issue{ID: "1", Title: "A", State: "open", Assignees: []model.Assignee{{Login: "alice"}}},
		model.Issue{ID: "2", Title: "B", State: "open", Assignees: []model.Assignee{{Login: "bob"}}},
		model.Issue{ID: "3", Title: "C", State: "open"},
	))
	d.assigneeInput.SetValue("ali")
	d.applyFilters()

	if len(d.filtered) != 1 || d.filtered[0].ID != "1" {
		t.Errorf("expected only item 1, got %+v", d.filtered)
	}
}

func TestApplyFilters_LabelFilter(t *testing.T) {
	d := newTestDashboard(items(
		model.Issue{ID: "1", Title: "A", State: "open", Labels: []model.Label{{Name: "bug"}}},
		model.Issue{ID: "2", Title: "B", State: "open", Labels: []model.Label{{Name: "feature"}}},
	))
	d.labelInput.SetValue("BUG")
	d.applyFilters()

	if len(d.filtered) != 1 || d.filtered[0].ID != "1" {
		t.Errorf("expected only item 1, got %+v", d.filtered)
	}
}

func TestApplyFilters_StatusFilter(t *testing.T) {
	it1 := model.ProjectItem{ID: "1", Issue: model.Issue{ID: "1", Title: "A", State: "open"}, Status: "Todo"}
	it2 := model.ProjectItem{ID: "2", Issue: model.Issue{ID: "2", Title: "B", State: "open"}, Status: "Done"}
	it3 := model.ProjectItem{ID: "3", Issue: model.Issue{ID: "3", Title: "C", State: "open"}, Status: "In Progress"}

	d := newTestDashboard(nil)
	d.allItems = []model.ProjectItem{it1, it2, it3}
	d.statusIdx = 3 // index 3 → statusOptions[2] = "Done" (statusIdx-1 = 2)
	d.assigneeInput = textinput.New()
	d.labelInput = textinput.New()
	d.titleInput = textinput.New()
	d.descInput = textinput.New()

	d.applyFilters()

	if len(d.filtered) != 1 || d.filtered[0].ID != "2" {
		t.Errorf("expected only 'Done' item, got %+v", d.filtered)
	}
}

func TestApplyFilters_IterationBacklog(t *testing.T) {
	it1 := model.ProjectItem{ID: "1", Issue: model.Issue{ID: "1", Title: "A", State: "open"}, IterationID: ""}
	it2 := model.ProjectItem{ID: "2", Issue: model.Issue{ID: "2", Title: "B", State: "open"}, IterationID: "iter-1"}

	d := newTestDashboard(nil)
	d.allItems = []model.ProjectItem{it1, it2}
	d.iterations = []model.Iteration{
		{ID: "", Title: "All"},
		{ID: gh.BacklogIterationID, Title: "Backlog"},
		{ID: "iter-1", Title: "Sprint 1"},
	}
	d.iterationIdx = 1 // Backlog
	d.assigneeInput = textinput.New()
	d.labelInput = textinput.New()
	d.titleInput = textinput.New()
	d.descInput = textinput.New()

	d.applyFilters()

	if len(d.filtered) != 1 || d.filtered[0].ID != "1" {
		t.Errorf("expected only backlog item (no iteration), got %+v", d.filtered)
	}
}

func TestApplyFilters_IterationSpecific(t *testing.T) {
	it1 := model.ProjectItem{ID: "1", Issue: model.Issue{ID: "1", Title: "A", State: "open"}, IterationID: ""}
	it2 := model.ProjectItem{ID: "2", Issue: model.Issue{ID: "2", Title: "B", State: "open"}, IterationID: "iter-1"}
	it3 := model.ProjectItem{ID: "3", Issue: model.Issue{ID: "3", Title: "C", State: "open"}, IterationID: "iter-2"}

	d := newTestDashboard(nil)
	d.allItems = []model.ProjectItem{it1, it2, it3}
	d.iterations = []model.Iteration{
		{ID: "", Title: "All"},
		{ID: gh.BacklogIterationID, Title: "Backlog"},
		{ID: "iter-1", Title: "Sprint 1"},
		{ID: "iter-2", Title: "Sprint 2"},
	}
	d.iterationIdx = 2 // Sprint 1
	d.assigneeInput = textinput.New()
	d.labelInput = textinput.New()
	d.titleInput = textinput.New()
	d.descInput = textinput.New()

	d.applyFilters()

	if len(d.filtered) != 1 || d.filtered[0].ID != "2" {
		t.Errorf("expected only Sprint 1 item, got %+v", d.filtered)
	}
}

func TestApplyFilters_TypeIssueOnly(t *testing.T) {
	// In project items context, issues have a non-empty State; items with empty
	// State are treated as non-issues (e.g. draft/PR-like entries).
	d := newTestDashboard(items(
		model.Issue{ID: "1", Title: "A", State: "open"},
		model.Issue{ID: "2", Title: "B", State: ""},  // no State → filtered out for "issue" type
	))
	d.typeIdx = 1 // "issue"
	d.applyFilters()

	if len(d.filtered) != 1 || d.filtered[0].ID != "1" {
		t.Errorf("expected only state=open item, got %+v", d.filtered)
	}
}

// ─── filterProjects ───────────────────────────────────────────────────────────

func TestFilterProjects_MeShowsAll(t *testing.T) {
	d := Dashboard{
		orgOptions: []string{"me", "orgA", "orgB"},
		orgIdx:     0, // "me"
		allProjects: []model.Project{
			{ID: "1", Owner: "orgA"},
			{ID: "2", Owner: "orgB"},
			{ID: "3", Owner: "orgA"},
		},
	}
	d.filterProjects()
	if len(d.projects) != 3 {
		t.Errorf("expected all 3 projects for 'me', got %d", len(d.projects))
	}
}

func TestFilterProjects_OrgFilters(t *testing.T) {
	d := Dashboard{
		orgOptions: []string{"me", "orgA", "orgB"},
		orgIdx:     1, // "orgA"
		allProjects: []model.Project{
			{ID: "1", Owner: "orgA"},
			{ID: "2", Owner: "orgB"},
			{ID: "3", Owner: "orgA"},
		},
	}
	d.filterProjects()
	if len(d.projects) != 2 {
		t.Errorf("expected 2 projects for orgA, got %d", len(d.projects))
	}
	for _, p := range d.projects {
		if p.Owner != "orgA" {
			t.Errorf("unexpected owner %q in filtered projects", p.Owner)
		}
	}
}

func TestFilterProjects_OrgNoMatch(t *testing.T) {
	d := Dashboard{
		orgOptions: []string{"me", "orgC"},
		orgIdx:     1, // "orgC"
		allProjects: []model.Project{
			{ID: "1", Owner: "orgA"},
		},
	}
	d.filterProjects()
	if len(d.projects) != 0 {
		t.Errorf("expected 0 projects for unmatched org, got %d", len(d.projects))
	}
}
