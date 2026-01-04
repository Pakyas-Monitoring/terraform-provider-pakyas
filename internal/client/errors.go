package client

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError represents an error from the Pakyas API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("pakyas API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("pakyas API error (status %d): %s", e.StatusCode, e.Body)
}

// IsNotFound returns true if the error is a 404 Not Found error.
// Used to remove resources from state when they no longer exist.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsConflict returns true if the error is a 409 Conflict error.
// Used to detect when a resource already exists on create.
func IsConflict(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusConflict
	}
	return false
}

// IsUnauthorized returns true if the error is a 401 Unauthorized error.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsForbidden returns true if the error is a 403 Forbidden error.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsRetryable returns true if the error is transient and the request should be retried.
// Retryable errors: 429 Too Many Requests, 5xx Server Errors.
func IsRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode >= 500
	}
	return false
}

// ConflictError returns an error message for 409 conflicts.
func ConflictError(resourceType string) error {
	return fmt.Errorf("%s already exists, use `terraform import` to manage it", resourceType)
}
