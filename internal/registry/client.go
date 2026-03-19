package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rosseca/aisi/internal/config"
)

const (
	defaultAPIBase = "https://skillsmp.com"
	defaultTimeout = 10 * time.Second
	defaultLimit   = 20
)

// Client is an HTTP client for the SkillsMP registry API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new registry client with default settings
// It reads the API key from config or SKILLSMP_API_KEY env var
func NewClient() *Client {
	cfg, _ := config.Load()
	apiKey := ""
	if cfg != nil {
		apiKey = cfg.GetSkillsMPAPIKey()
	}
	return NewClientWithKey(apiKey)
}

// NewClientWithKey creates a client with a specific API key
func NewClientWithKey(apiKey string) *Client {
	return NewClientWithURLAndKey(defaultAPIBase, apiKey)
}

// NewClientWithURLAndKey creates a client with custom URL and API key
func NewClientWithURLAndKey(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// SetAPIKey sets the API key for subsequent requests
func (c *Client) SetAPIKey(key string) {
	c.apiKey = key
}

// Search queries the SkillsMP registry for skills matching the query
// sortBy can be "stars" or "recent" (empty defaults to API default)
func (c *Client) Search(ctx context.Context, query string, limit int, sortBy string) ([]Skill, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if limit <= 0 {
		limit = defaultLimit
	}

	if c.apiKey == "" {
		return nil, fmt.Errorf("SkillsMP API key not configured. Set it with: aisi config set-skillsmp-key <your-api-key>")
	}

	apiURL, err := url.Parse(c.baseURL + "/api/v1/skills/search")
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %w", err)
	}

	q := apiURL.Query()
	q.Set("q", query)
	q.Set("limit", fmt.Sprintf("%d", limit))
	if sortBy != "" {
		q.Set("sortBy", sortBy)
	}
	apiURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Bearer authorization header
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key - please check your SkillsMP API key")
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("daily API quota exceeded (500 requests/day) - resets at midnight UTC")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry API returned status %d", resp.StatusCode)
	}

	var searchResp SkillsMPResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Convert SkillsMP response to our internal Skill format
	return searchResp.toSkills(), nil
}

// AISearch performs AI semantic search on SkillsMP
func (c *Client) AISearch(ctx context.Context, query string) ([]Skill, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if c.apiKey == "" {
		return nil, fmt.Errorf("SkillsMP API key not configured. Set it with: aisi config set-skillsmp-key <your-api-key>")
	}

	apiURL, err := url.Parse(c.baseURL + "/api/v1/skills/ai-search")
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %w", err)
	}

	q := apiURL.Query()
	q.Set("q", query)
	apiURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key - please check your SkillsMP API key")
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("daily API quota exceeded (500 requests/day) - resets at midnight UTC")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry API returned status %d", resp.StatusCode)
	}

	var searchResp SkillsMPResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return searchResp.toSkills(), nil
}

// GetSkillURL returns the full URL for viewing a skill on skillsmp.com
func (c *Client) GetSkillURL(slug string) string {
	return fmt.Sprintf("%s/s/%s", c.baseURL, slug)
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
