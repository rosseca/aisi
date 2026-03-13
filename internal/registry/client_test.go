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
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query parameters
		q := r.URL.Query().Get("q")
		if q == "" {
			t.Error("expected query parameter 'q'")
		}

		limit := r.URL.Query().Get("limit")
		if limit != "10" {
			t.Errorf("expected limit=10, got %s", limit)
		}

		response := SearchResponse{
			Skills: []Skill{
				{ID: "owner/repo/skill1", Name: "skill1", Installs: 1000, Source: "owner/repo"},
				{ID: "owner/repo/skill2", Name: "skill2", Installs: 500, Source: "owner/repo"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skills, err := client.Search(ctx, "typescript", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}

	if skills[0].Name != "skill1" {
		t.Errorf("expected skill1, got %s", skills[0].Name)
	}
}

func TestClient_Search_EmptyQuery(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	_, err := client.Search(ctx, "", 10)
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestClient_Search_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Search(ctx, "test", 10)
	if err == nil {
		t.Error("expected error for non-200 status")
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
	client := NewClientWithURL("https://skills.sh")
	url := client.GetSkillURL("owner/repo/skill-name")
	expected := "https://skills.sh/owner/repo/skill-name"
	if url != expected {
		t.Errorf("GetSkillURL() = %s, want %s", url, expected)
	}
}
