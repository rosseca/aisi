package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultAPIBase = "https://skills.sh"
	defaultTimeout = 10 * time.Second
	defaultLimit   = 10
)

// Client is an HTTP client for the skills.sh registry API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new registry client with default settings
func NewClient() *Client {
	return NewClientWithURL(defaultAPIBase)
}

// NewClientWithURL creates a client with a custom API base URL
func NewClientWithURL(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Search queries the skills.sh registry for skills matching the query
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Skill, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if limit <= 0 {
		limit = defaultLimit
	}

	apiURL, err := url.Parse(c.baseURL + "/api/search")
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %w", err)
	}

	q := apiURL.Query()
	q.Set("q", query)
	q.Set("limit", fmt.Sprintf("%d", limit))
	apiURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry API returned status %d", resp.StatusCode)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return searchResp.Skills, nil
}

// GetSkillURL returns the full URL for viewing a skill on skills.sh
func (c *Client) GetSkillURL(slug string) string {
	return fmt.Sprintf("%s/%s", c.baseURL, slug)
}

// FormatInstalls formats the install count for display (e.g., "1.2K", "3M")
func FormatInstalls(count int) string {
	if count <= 0 {
		return ""
	}
	if count >= 1_000_000 {
		return fmt.Sprintf("%.1fM installs", float64(count)/1_000_000)
	}
	if count >= 1_000 {
		return fmt.Sprintf("%.1fK installs", float64(count)/1_000)
	}
	if count == 1 {
		return "1 install"
	}
	return fmt.Sprintf("%d installs", count)
}
