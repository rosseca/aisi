package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Search(t *testing.T) {
	// Create mock server that returns SkillsMP format
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error": map[string]string{
					"code":    "MISSING_API_KEY",
					"message": "API key not provided",
				},
			})
			return
		}

		// Check query parameters
		q := r.URL.Query().Get("q")
		if q == "" {
			t.Error("expected query parameter 'q'")
		}

		limit := r.URL.Query().Get("limit")
		if limit != "10" {
			t.Errorf("expected limit=10, got %s", limit)
		}

		// Return SkillsMP format response
		response := SkillsMPResponse{
			Success: true,
			Data: SkillsMPData{
				Skills: []SkillsMPSkill{
					{ID: "owner/repo/skill1", Name: "skill1", Installs: 1000, Repository: "owner/repo"},
					{ID: "owner/repo/skill2", Name: "skill2", Installs: 500, Repository: "owner/repo"},
				},
				Total: 2,
				Page:  1,
				Limit: 10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer server.Close()

	client := NewClientWithURLAndKey(server.URL, "test-api-key")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skills, err := client.Search(ctx, "typescript", 10, "stars")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}

	if skills[0].Name != "skill1" {
		t.Errorf("expected skill1, got %s", skills[0].Name)
	}

	if skills[0].Source != "owner/repo" {
		t.Errorf("expected source 'owner/repo', got %s", skills[0].Source)
	}
}

func TestClient_Search_NoAPIKey(t *testing.T) {
	client := NewClientWithURLAndKey("https://skillsmp.com", "")
	ctx := context.Background()

	_, err := client.Search(ctx, "test", 10, "")
	if err == nil {
		t.Error("expected error for missing API key")
	}

	if err.Error() != "SkillsMP API key not configured. Set it with: aisi config set-skillsmp-key <your-api-key>" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_Search_EmptyQuery(t *testing.T) {
	client := NewClientWithURLAndKey("https://skillsmp.com", "test-key")
	ctx := context.Background()

	_, err := client.Search(ctx, "", 10, "")
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestClient_Search_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClientWithURLAndKey(server.URL, "test-api-key")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Search(ctx, "test", 10, "")
	if err == nil {
		t.Error("expected error for non-200 status")
	}
}

func TestClient_Search_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error": map[string]string{
				"code":    "INVALID_API_KEY",
				"message": "Invalid API key",
			},
		})
	}))
	defer server.Close()

	client := NewClientWithURLAndKey(server.URL, "invalid-key")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Search(ctx, "test", 10, "")
	if err == nil {
		t.Error("expected error for 401 status")
	}

	if err.Error() != "invalid API key - please check your SkillsMP API key" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_Search_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error": map[string]string{
				"code":    "DAILY_QUOTA_EXCEEDED",
				"message": "Daily quota exceeded",
			},
		})
	}))
	defer server.Close()

	client := NewClientWithURLAndKey(server.URL, "test-key")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Search(ctx, "test", 10, "")
	if err == nil {
		t.Error("expected error for 429 status")
	}

	if err.Error() != "daily API quota exceeded (500 requests/day) - resets at midnight UTC" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFormatInstalls(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, ""},
		{1, "1 install"},
		{5, "5 installs"},
		{999, "999 installs"},
		{1000, "1.0K installs"},
		{1500, "1.5K installs"},
		{1000000, "1.0M installs"},
		{2500000, "2.5M installs"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatInstalls(tt.count)
			if result != tt.expected {
				t.Errorf("FormatInstalls(%d) = %s, want %s", tt.count, result, tt.expected)
			}
		})
	}
}

func TestClient_GetSkillURL(t *testing.T) {
	client := NewClientWithURLAndKey("https://skillsmp.com", "")
	url := client.GetSkillURL("owner/repo/skill-name")
	expected := "https://skillsmp.com/s/owner/repo/skill-name"
	if url != expected {
		t.Errorf("GetSkillURL() = %s, want %s", url, expected)
	}
}

func TestClient_AISearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check it's hitting the AI search endpoint
		if r.URL.Path != "/api/v1/skills/ai-search" {
			t.Errorf("expected path /api/v1/skills/ai-search, got %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		response := SkillsMPResponse{
			Success: true,
			Data: SkillsMPData{
				Skills: []SkillsMPSkill{
					{ID: "owner/repo/ai-skill", Name: "ai-skill", Repository: "owner/repo"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClientWithURLAndKey(server.URL, "test-api-key")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skills, err := client.AISearch(ctx, "how to create a web scraper")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}

	if skills[0].Name != "ai-skill" {
		t.Errorf("expected 'ai-skill', got %s", skills[0].Name)
	}
}

func TestClient_Search_SortBy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check sortBy parameter
		sortBy := r.URL.Query().Get("sortBy")
		if sortBy != "recent" {
			t.Errorf("expected sortBy=recent, got %s", sortBy)
		}

		response := SkillsMPResponse{
			Success: true,
			Data: SkillsMPData{
				Skills: []SkillsMPSkill{
					{ID: "owner/repo/skill1", Name: "skill1", Repository: "owner/repo"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClientWithURLAndKey(server.URL, "test-api-key")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skills, err := client.Search(ctx, "test", 10, "recent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}
}

func TestExtractOwnerRepoFromGithubURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/NeverSight/learn-skills.dev/tree/main/data/skills-md/ios-swift", "NeverSight/learn-skills.dev"},
		{"https://github.com/CUBETIQ/cubis-foundry/tree/main/workflows/skills/swift-best-practices", "CUBETIQ/cubis-foundry"},
		{"https://github.com/vibecode/skills/tree/main/skills/mobile-app-designer", "vibecode/skills"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"http://github.com/owner/repo", "owner/repo"},
		{"github.com/owner/repo", "owner/repo"},
		{"", ""},
		{"not-a-url", ""}, // invalid URLs return empty
	}

	for _, tt := range tests {
		result := extractOwnerRepoFromGithubURL(tt.url)
		if result != tt.expected {
			t.Errorf("extractOwnerRepoFromGithubURL(%q) = %q, want %q", tt.url, result, tt.expected)
		}
	}
}

func TestClient_Search_RealSkillsMPFormat(t *testing.T) {
	// Test with actual SkillsMP API response format
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return SkillsMP format with githubUrl (actual API format)
		response := SkillsMPResponse{
			Success: true,
			Data: SkillsMPData{
				Skills: []SkillsMPSkill{
					{
						ID:          "neversight-learn-skills-dev-data-skills-md-ios-swift-skill-md",
						Name:        "ios-swift",
						Author:      "NeverSight",
						Description: "Expert iOS development skill",
						GithubURL:   "https://github.com/NeverSight/learn-skills.dev/tree/main/data/skills-md/ios-swift",
						SkillURL:    "https://skillsmp.com/skills/ios-swift",
						Stars:       84,
					},
					{
						ID:          "cubetiq-cubis-foundry-swift-best-practices-skill-md",
						Name:        "swift-best-practices",
						Author:      "CUBETIQ",
						Description: "Swift best practices",
						GithubURL:   "https://github.com/CUBETIQ/cubis-foundry/tree/main/workflows/skills/swift-best-practices",
						SkillURL:    "https://skillsmp.com/skills/swift-best-practices",
						Stars:       5,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClientWithURLAndKey(server.URL, "test-api-key")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skills, err := client.Search(ctx, "swift", 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}

	// Check first skill
	if skills[0].Name != "ios-swift" {
		t.Errorf("expected name 'ios-swift', got %s", skills[0].Name)
	}
	if skills[0].Source != "NeverSight/learn-skills.dev" {
		t.Errorf("expected source 'NeverSight/learn-skills.dev', got %s", skills[0].Source)
	}

	// Check second skill
	if skills[1].Name != "swift-best-practices" {
		t.Errorf("expected name 'swift-best-practices', got %s", skills[1].Name)
	}
	if skills[1].Source != "CUBETIQ/cubis-foundry" {
		t.Errorf("expected source 'CUBETIQ/cubis-foundry', got %s", skills[1].Source)
	}
}
