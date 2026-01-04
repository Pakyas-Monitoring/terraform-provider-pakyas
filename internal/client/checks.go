package client

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// Check represents a Pakyas check.
type Check struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	PeriodSeconds int64      `json:"period_seconds"`
	GraceSeconds  int64      `json:"grace_seconds"`
	Description   *string    `json:"description"`
	Tags          []string   `json:"tags"`
	Paused        bool       `json:"paused"`
	PublicID      string     `json:"public_id"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// CreateCheckRequest is the request body for creating a check.
type CreateCheckRequest struct {
	ProjectID     string   `json:"project_id"`
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	PeriodSeconds int64    `json:"period_seconds"`
	GraceSeconds  int64    `json:"grace_seconds,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Paused        bool     `json:"paused,omitempty"`
}

// UpdateCheckRequest is the request body for updating a check (PATCH-style).
type UpdateCheckRequest struct {
	Name          *string  `json:"name,omitempty"`
	PeriodSeconds *int64   `json:"period_seconds,omitempty"`
	GraceSeconds  *int64   `json:"grace_seconds,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Paused        *bool    `json:"paused,omitempty"`
}

// CreateCheck creates a new check.
func (c *Client) CreateCheck(ctx context.Context, req CreateCheckRequest) (*Check, error) {
	// Normalize description
	req.Description = normalizeDescription(req.Description)
	// Sort tags for deterministic API logs
	req.Tags = normalizeTags(req.Tags)

	var check Check
	if err := c.doRequest(ctx, http.MethodPost, "/api/v1/checks", req, &check); err != nil {
		if IsConflict(err) {
			return nil, ConflictError("check")
		}
		return nil, err
	}

	// Read after create to ensure we have all server-populated fields
	return c.GetCheck(ctx, check.ID)
}

// GetCheck retrieves a check by ID.
func (c *Client) GetCheck(ctx context.Context, id string) (*Check, error) {
	var check Check
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/checks/%s", id), nil, &check); err != nil {
		return nil, err
	}
	// Normalize tags for consistent state
	check.Tags = normalizeTags(check.Tags)
	return &check, nil
}

// UpdateCheck updates a check (PATCH-style, only changed fields).
func (c *Client) UpdateCheck(ctx context.Context, id string, req UpdateCheckRequest) (*Check, error) {
	// Normalize description
	req.Description = normalizeDescription(req.Description)
	// Sort tags for deterministic API logs
	req.Tags = normalizeTags(req.Tags)

	if err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/checks/%s", id), req, nil); err != nil {
		return nil, err
	}

	// Read after update to get the updated state
	return c.GetCheck(ctx, id)
}

// DeleteCheck soft-deletes a check.
func (c *Client) DeleteCheck(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/checks/%s", id), nil, nil)
}

// normalizeTags normalizes tags: nil/empty → empty slice, and sorts for determinism.
func normalizeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	// Create a copy and sort
	sorted := make([]string, len(tags))
	copy(sorted, tags)
	sort.Strings(sorted)
	return sorted
}
