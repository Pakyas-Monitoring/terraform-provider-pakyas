package client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Project represents a Pakyas project.
type Project struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"org_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

// CreateProjectRequest is the request body for creating a project.
type CreateProjectRequest struct {
	OrgID       string  `json:"org_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// UpdateProjectRequest is the request body for updating a project (PATCH-style).
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CreateProject creates a new project.
func (c *Client) CreateProject(ctx context.Context, name string, description *string) (*Project, error) {
	req := CreateProjectRequest{
		OrgID:       c.orgID,
		Name:        name,
		Description: normalizeDescription(description),
	}

	var project Project
	if err := c.doRequest(ctx, http.MethodPost, "/api/v1/projects", req, &project); err != nil {
		if IsConflict(err) {
			return nil, ConflictError("project")
		}
		return nil, err
	}

	// Read after create to ensure we have all server-populated fields
	return c.GetProject(ctx, project.ID)
}

// GetProject retrieves a project by ID.
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	var project Project
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s", id), nil, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// UpdateProject updates a project (PATCH-style, only changed fields).
func (c *Client) UpdateProject(ctx context.Context, id string, name *string, description *string) (*Project, error) {
	req := UpdateProjectRequest{
		Name:        name,
		Description: normalizeDescription(description),
	}

	if err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/projects/%s", id), req, nil); err != nil {
		return nil, err
	}

	// Read after update to get the updated state
	return c.GetProject(ctx, id)
}

// DeleteProject archives a project.
func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/projects/%s", id), nil, nil)
}

// normalizeDescription normalizes description field.
// Empty string is treated as null to prevent diffs.
func normalizeDescription(desc *string) *string {
	if desc != nil && *desc == "" {
		return nil
	}
	return desc
}
