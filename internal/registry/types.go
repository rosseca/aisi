package registry

import "strings"

// Skill represents a skill from the SkillsMP registry
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Installs    int    `json:"installs"`
	Source      string `json:"source"` // owner/repo format
	Stars       int    `json:"stars,omitempty"`
	URL         string `json:"url,omitempty"`
	GithubURL   string `json:"githubUrl,omitempty"` // Full GitHub URL for fetching SKILL.md
}

// SkillsMPResponse represents the API response from SkillsMP
type SkillsMPResponse struct {
	Success bool           `json:"success"`
	Data    SkillsMPData   `json:"data,omitempty"`
	Error   *SkillsMPError `json:"error,omitempty"`
	// Support both wrapped and unwrapped responses
	Skills []SkillsMPSkill `json:"skills,omitempty"`
}

type SkillsMPData struct {
	Skills []SkillsMPSkill `json:"skills"`
	Total  int             `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
	// AI search returns results in a nested data array
	Results []SkillsMPResult `json:"data"`
}

// SkillsMPResult represents a single AI search result (vector search)
type SkillsMPResult struct {
	FileID   string      `json:"file_id"`
	Filename string      `json:"filename"`
	Score    float64     `json:"score"`
	Skill    SkillsMPSkill `json:"skill"`
}

type SkillsMPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SkillsMPSkill represents a skill in the SkillsMP API response
type SkillsMPSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	Repository  string `json:"repository,omitempty"` // owner/repo format
	Installs    int    `json:"installs,omitempty"`
	Stars       int    `json:"stars,omitempty"`
	URL         string `json:"url,omitempty"`
	GithubURL   string `json:"githubUrl,omitempty"` // Full GitHub URL
	SkillURL    string `json:"skillUrl,omitempty"`
	// Alternative field names that might be in the response
	Owner string `json:"owner,omitempty"`
	Repo  string `json:"repo,omitempty"`
}

// toSkills converts SkillsMP response to our internal Skill format
func (r *SkillsMPResponse) toSkills() []Skill {
	var skills []Skill

	// Try wrapped format first (data.skills)
	sourceSkills := r.Data.Skills
	// Fall back to unwrapped format
	if len(sourceSkills) == 0 && len(r.Skills) > 0 {
		sourceSkills = r.Skills
	}
	// AI search returns results in data.data[].skill
	if len(sourceSkills) == 0 && len(r.Data.Results) > 0 {
		for _, result := range r.Data.Results {
			sourceSkills = append(sourceSkills, result.Skill)
		}
	}

	for _, s := range sourceSkills {
		skill := Skill{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Installs:    s.Installs,
			Stars:       s.Stars,
			URL:         s.SkillURL,
			GithubURL:   s.GithubURL,
		}

		// Build source from various possible fields
		if s.Repository != "" {
			// Direct repository field (owner/repo format)
			skill.Source = s.Repository
		} else if s.GithubURL != "" {
			// Extract owner/repo from GitHub URL
			skill.Source = extractOwnerRepoFromGithubURL(s.GithubURL)
		} else if s.Owner != "" && s.Repo != "" {
			// Separate owner and repo fields
			skill.Source = s.Owner + "/" + s.Repo
		} else if s.Author != "" {
			// Use author as owner, try to infer repo from name or ID
			repo := s.Name
			if repo == "" {
				repo = extractRepoFromID(s.ID)
			}
			skill.Source = s.Author + "/" + repo
		}

		skills = append(skills, skill)
	}

	return skills
}

// extractOwnerRepoFromGithubURL parses a GitHub URL and returns "owner/repo"
// Supports formats like:
// - https://github.com/owner/repo/tree/main/path
// - https://github.com/owner/repo
func extractOwnerRepoFromGithubURL(url string) string {
	// Remove protocol and domain
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")

	// Split path
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		// Return owner/repo
		return parts[0] + "/" + parts[1]
	}

	return ""
}

// extractRepoFromID tries to extract a reasonable repo name from the skill ID
func extractRepoFromID(id string) string {
	// IDs are typically long strings like "owner-repo-path-skill-name-skill-md"
	// Try to find the skill name portion
	parts := strings.Split(id, "-")

	// Look for "skill" or "skills" in the ID to identify the skill name portion
	for i, part := range parts {
		if part == "skill" || part == "skills" {
			// The parts before "skill" likely contain the repo name
			if i > 0 {
				// Return the part right before "skill" (likely the actual skill name)
				return parts[i-1]
			}
		}
	}

	// Fallback: return last non-empty part
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && parts[i] != "md" {
			return parts[i]
		}
	}

	return "unknown"
}

// SearchResponse represents the API response (legacy format for compatibility)
type SearchResponse struct {
	Skills []Skill `json:"skills"`
}

// SearchResult wraps a skill with additional metadata for display
type SearchResult struct {
	Skill             Skill
	Slug              string // Full slug (owner/repo/skill-name)
	InstallsFormatted string
}
