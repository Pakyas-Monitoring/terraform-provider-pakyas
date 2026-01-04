package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// DefaultBaseURL is the default Pakyas API URL.
	DefaultBaseURL = "https://api.pakyas.com"
	// DefaultPingURLBase is the fallback ping URL base if not returned by /me.
	DefaultPingURLBase = "https://ping.pakyas.com"
	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 15 * time.Second
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries = 5
	// BaseRetryDelay is the base delay between retries.
	BaseRetryDelay = 1 * time.Second
)

// Client is the Pakyas API client.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	apiKey      string
	userAgent   string
	orgID       string // Cached from /me
	pingURLBase string // Cached from /me
}

// MeResponse represents the response from GET /api/v1/me.
type MeResponse struct {
	OrganizationID   string   `json:"organization_id"`
	OrganizationName string   `json:"organization_name"`
	Scopes           []string `json:"scopes"`
	PingURLBase      string   `json:"ping_url_base"`
}

// ClientConfig holds configuration for creating a new client.
type ClientConfig struct {
	APIKey    string
	BaseURL   string
	UserAgent string
}

// New creates a new Pakyas API client.
// It calls /me to cache organization context and ping URL base.
func New(ctx context.Context, cfg ClientConfig) (*Client, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	// Normalize: strip trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "terraform-provider-pakyas"
	}

	c := &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL:   baseURL,
		apiKey:    cfg.APIKey,
		userAgent: userAgent,
	}

	// Call /me to get org context
	if err := c.fetchOrgContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch organization context: %w", err)
	}

	return c, nil
}

// OrgID returns the cached organization ID.
func (c *Client) OrgID() string {
	return c.orgID
}

// PingURLBase returns the cached ping URL base.
func (c *Client) PingURLBase() string {
	return c.pingURLBase
}

// fetchOrgContext calls GET /me to retrieve and cache org context.
func (c *Client) fetchOrgContext(ctx context.Context) error {
	var meResp MeResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/v1/me", nil, &meResp); err != nil {
		return err
	}

	c.orgID = meResp.OrganizationID
	c.pingURLBase = meResp.PingURLBase

	// Fallback if ping_url_base is empty
	if c.pingURLBase == "" {
		tflog.Warn(ctx, "ping_url_base not returned by /me, using default", map[string]interface{}{
			"default": DefaultPingURLBase,
		})
		c.pingURLBase = DefaultPingURLBase
	}

	// Normalize: strip trailing slash
	c.pingURLBase = strings.TrimSuffix(c.pingURLBase, "/")

	tflog.Debug(ctx, "fetched organization context", map[string]interface{}{
		"org_id":        c.orgID,
		"ping_url_base": c.pingURLBase,
	})

	return nil
}

// doRequest performs an HTTP request with retry logic.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff + jitter
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * BaseRetryDelay
			jitter := time.Duration(rand.Int63n(int64(delay / 2)))
			delay = delay + jitter

			tflog.Debug(ctx, "retrying request", map[string]interface{}{
				"attempt": attempt,
				"delay":   delay.String(),
				"url":     url,
			})

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Reset body reader for retry
			if body != nil {
				jsonBody, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(jsonBody)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			// Network errors are retryable
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Check for error status codes
		if resp.StatusCode >= 400 {
			apiErr := &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(respBody),
			}

			// Try to parse error message from JSON
			var errResp struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}
			if json.Unmarshal(respBody, &errResp) == nil {
				if errResp.Error != "" {
					apiErr.Message = errResp.Error
				} else if errResp.Message != "" {
					apiErr.Message = errResp.Message
				}
			}

			// Check if retryable
			if IsRetryable(apiErr) && attempt < MaxRetries {
				lastErr = apiErr
				continue
			}

			return apiErr
		}

		// Success - parse response
		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
