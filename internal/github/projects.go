package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jhermoso/ghtui/internal/model"
	"github.com/shurcooL/githubv4"
)

// ListUserProjects returns the authenticated user's Projects v2.
func (c *Client) ListUserProjects(ctx context.Context) ([]model.Project, error) {
	var q struct {
		Viewer struct {
			Login      string
			ProjectsV2 struct {
				Nodes []struct {
					ID     githubv4.ID
					Number int
					Title  string
					URL    string
				}
			} `graphql:"projectsV2(first: 50)"`
		}
	}
	if err := c.GraphQL.Query(ctx, &q, nil); err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	projects := make([]model.Project, 0, len(q.Viewer.ProjectsV2.Nodes))
	for _, n := range q.Viewer.ProjectsV2.Nodes {
		projects = append(projects, model.Project{
			ID:     fmt.Sprintf("%v", n.ID),
			Number: n.Number,
			Title:  n.Title,
			URL:    n.URL,
			Owner:  q.Viewer.Login,
		})
	}
	return projects, nil
}

// ListAllProjects returns the viewer's own Projects v2 plus every accessible
// project from each organization the viewer belongs to, in a single query.
func (c *Client) ListAllProjects(ctx context.Context) ([]model.Project, error) {
	var q struct {
		Viewer struct {
			Login      string
			ProjectsV2 struct {
				Nodes []struct {
					ID     githubv4.ID
					Number int
					Title  string
					URL    string
				}
			} `graphql:"projectsV2(first: 50)"`
			Organizations struct {
				Nodes []struct {
					Login      string
					ProjectsV2 struct {
						Nodes []struct {
							ID     githubv4.ID
							Number int
							Title  string
							URL    string
						}
					} `graphql:"projectsV2(first: 50)"`
				}
			} `graphql:"organizations(first: 100)"`
		}
	}
	if err := c.GraphQL.Query(ctx, &q, nil); err != nil {
		return nil, fmt.Errorf("listing all projects: %w", err)
	}

	var projects []model.Project

	for _, n := range q.Viewer.ProjectsV2.Nodes {
		projects = append(projects, model.Project{
			ID:     fmt.Sprintf("%v", n.ID),
			Number: n.Number,
			Title:  n.Title,
			URL:    n.URL,
			Owner:  q.Viewer.Login,
		})
	}

	for _, org := range q.Viewer.Organizations.Nodes {
		for _, n := range org.ProjectsV2.Nodes {
			projects = append(projects, model.Project{
				ID:     fmt.Sprintf("%v", n.ID),
				Number: n.Number,
				Title:  fmt.Sprintf("[%s] %s", org.Login, n.Title),
				URL:    n.URL,
				Owner:  org.Login,
			})
		}
	}

	return projects, nil
}

// ListOrgProjects returns a GitHub org's Projects v2.
func (c *Client) ListOrgProjects(ctx context.Context, org string) ([]model.Project, error) {
	var q struct {
		Organization struct {
			Login      string
			ProjectsV2 struct {
				Nodes []struct {
					ID     githubv4.ID
					Number int
					Title  string
					URL    string
				}
			} `graphql:"projectsV2(first: 50)"`
		} `graphql:"organization(login: $org)"`
	}
	vars := map[string]interface{}{"org": githubv4.String(org)}
	if err := c.GraphQL.Query(ctx, &q, vars); err != nil {
		return nil, fmt.Errorf("listing org projects: %w", err)
	}
	projects := make([]model.Project, 0, len(q.Organization.ProjectsV2.Nodes))
	for _, n := range q.Organization.ProjectsV2.Nodes {
		projects = append(projects, model.Project{
			ID:     fmt.Sprintf("%v", n.ID),
			Number: n.Number,
			Title:  n.Title,
			URL:    n.URL,
			Owner:  org,
		})
	}
	return projects, nil
}

type iterationFieldFragment struct {
	ID   githubv4.ID
	Name string
	Configuration struct {
		Iterations []struct {
			ID        string
			Title     string
			StartDate string
			Duration  int
		}
	}
}

// GetProjectIterations returns all iterations defined on a project.
func (c *Client) GetProjectIterations(ctx context.Context, projectID string) ([]model.Iteration, error) {
	var q struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						TypeName              string                 `graphql:"__typename"`
						ProjectV2IterationField iterationFieldFragment `graphql:"... on ProjectV2IterationField"`
					}
				} `graphql:"fields(first: 30)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}
	vars := map[string]interface{}{"id": githubv4.ID(projectID)}
	if err := c.GraphQL.Query(ctx, &q, vars); err != nil {
		return nil, fmt.Errorf("getting iterations: %w", err)
	}

	var iterations []model.Iteration
	for _, field := range q.Node.ProjectV2.Fields.Nodes {
		if field.TypeName == "ProjectV2IterationField" {
			for _, it := range field.ProjectV2IterationField.Configuration.Iterations {
				iterations = append(iterations, model.Iteration{
					ID:        it.ID,
					Title:     it.Title,
					StartDate: it.StartDate,
					Duration:  it.Duration,
				})
			}
		}
	}
	return iterations, nil
}

type projectItemNode struct {
	ID      githubv4.ID
	Content struct {
		TypeName string `graphql:"__typename"`
		Issue    struct {
			ID         githubv4.ID
			DatabaseID int
			Number     int
			Title      string
			Body       string
			State      string
			URL        string
			Repository struct {
				Name  string
				Owner struct{ Login string }
			}
			Assignees struct {
				Nodes []struct {
					Login     string
					AvatarURL string
				}
			} `graphql:"assignees(first: 5)"`
			Labels struct {
				Nodes []struct {
					Name  string
					Color string
				}
			} `graphql:"labels(first: 10)"`
		} `graphql:"... on Issue"`
	}
	FieldValues struct {
		Nodes []struct {
			TypeName       string `graphql:"__typename"`
			IterationValue struct {
				IterationID string `graphql:"iterationId"`
				Title       string
			} `graphql:"... on ProjectV2ItemFieldIterationValue"`
			SingleSelectValue struct {
				Name  string
				Field struct {
					SingleSelectField struct {
						Name string
					} `graphql:"... on ProjectV2SingleSelectField"`
				}
			} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
		}
	} `graphql:"fieldValues(first: 20)"`
}

// BacklogIterationID is the sentinel passed to GetProjectItems to request only
// items that have no iteration field value assigned (i.e. the backlog).
const BacklogIterationID = "__backlog__"

// GetProjectItems returns all items in a project, optionally filtered by iteration.
func (c *Client) GetProjectItems(ctx context.Context, projectID string, iterationID string) ([]model.ProjectItem, error) {
	var q struct {
		Node struct {
			ProjectV2 struct {
				Items struct {
					PageInfo struct {
						HasNextPage bool
						EndCursor   githubv4.String
					}
					Nodes []projectItemNode
				} `graphql:"items(first: 100)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}
	vars := map[string]interface{}{"id": githubv4.ID(projectID)}
	if err := c.GraphQL.Query(ctx, &q, vars); err != nil {
		return nil, fmt.Errorf("getting project items: %w", err)
	}

	var items []model.ProjectItem
	for _, n := range q.Node.ProjectV2.Items.Nodes {
		if n.Content.TypeName != "Issue" {
			continue
		}
		issue := n.Content.Issue

		var assignees []model.Assignee
		for _, a := range issue.Assignees.Nodes {
			assignees = append(assignees, model.Assignee{Login: a.Login, AvatarURL: a.AvatarURL})
		}
		var labels []model.Label
		for _, l := range issue.Labels.Nodes {
			labels = append(labels, model.Label{Name: l.Name, Color: l.Color})
		}

		var itemIterationID, status string
		for _, fv := range n.FieldValues.Nodes {
			switch fv.TypeName {
			case "ProjectV2ItemFieldIterationValue":
				itemIterationID = fv.IterationValue.IterationID
			case "ProjectV2ItemFieldSingleSelectValue":
				if fv.SingleSelectValue.Field.SingleSelectField.Name == "Status" {
					status = fv.SingleSelectValue.Name
				}
			}
		}

		switch iterationID {
		case "":
			// no filter — include everything
		case BacklogIterationID:
			if itemIterationID != "" {
				continue
			}
		default:
			if itemIterationID != iterationID {
				continue
			}
		}

		items = append(items, model.ProjectItem{
			ID: fmt.Sprintf("%v", n.ID),
			Issue: model.Issue{
				ID:         fmt.Sprintf("%v", issue.ID),
				Number:     issue.Number,
				Title:      issue.Title,
				Body:       issue.Body,
				State:      issue.State,
				URL:        issue.URL,
				Repository: issue.Repository.Name,
				Owner:      issue.Repository.Owner.Login,
				Assignees:  assignees,
				Labels:     labels,
			},
			IterationID: itemIterationID,
			Status:      status,
		})
	}
	return items, nil
}

// GetProjectMeta returns iteration field and status field options for a project.
func (c *Client) GetProjectMeta(ctx context.Context, projectID string) (*model.ProjectMeta, error) {
	var q struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						TypeName string `graphql:"__typename"`
						IterField struct {
							ID   githubv4.ID
							Name string
							Configuration struct {
								Iterations []struct {
									ID        string
									Title     string
									StartDate string
									Duration  int
								}
							}
						} `graphql:"... on ProjectV2IterationField"`
						SelectField struct {
							ID      githubv4.ID
							Name    string
							Options []struct {
								ID   string
								Name string
							}
						} `graphql:"... on ProjectV2SingleSelectField"`
					}
				} `graphql:"fields(first: 30)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}

	vars := map[string]interface{}{"id": githubv4.ID(projectID)}
	if err := c.GraphQL.Query(ctx, &q, vars); err != nil {
		return nil, fmt.Errorf("getting project meta: %w", err)
	}

	meta := &model.ProjectMeta{}
	for _, f := range q.Node.ProjectV2.Fields.Nodes {
		switch f.TypeName {
		case "ProjectV2IterationField":
			for _, it := range f.IterField.Configuration.Iterations {
				meta.Iterations = append(meta.Iterations, model.Iteration{
					ID:        it.ID,
					Title:     it.Title,
					StartDate: it.StartDate,
					Duration:  it.Duration,
				})
			}
		case "ProjectV2SingleSelectField":
			if f.SelectField.Name == "Status" {
				meta.StatusFieldID = fmt.Sprintf("%v", f.SelectField.ID)
				for _, o := range f.SelectField.Options {
					meta.StatusOptions = append(meta.StatusOptions, model.StatusOption{
						ID:   o.ID,
						Name: o.Name,
					})
				}
			}
		}
	}
	return meta, nil
}

// UpdateItemStatus sets the Status single-select field on a project item.
func (c *Client) UpdateItemStatus(ctx context.Context, projectID, itemID, fieldID, optionID string) error {
	query := `mutation($projectId:ID!,$itemId:ID!,$fieldId:ID!,$optId:String!){
		updateProjectV2ItemFieldValue(input:{
			projectId:$projectId itemId:$itemId fieldId:$fieldId
			value:{singleSelectOptionId:$optId}
		}){projectV2Item{id}}
	}`
	body, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"variables": map[string]string{
			"projectId": projectID,
			"itemId":    itemID,
			"fieldId":   fieldID,
			"optId":     optionID,
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.RawHTTP().Do(req)
	if err != nil {
		return fmt.Errorf("updating status: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", result.Errors[0].Message)
	}
	return nil
}
