package registry

// Skill represents a skill from the skills.sh registry
type Skill struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Installs int    `json:"installs"`
	Source   string `json:"source"` // owner/repo format
}

// SearchResponse represents the API response from skills.sh/api/search
type SearchResponse struct {
	Skills []Skill `json:"skills"`
}

// SearchResult wraps a skill with additional metadata for display
type SearchResult struct {
	Skill      Skill
	Slug       string // Full slug (owner/repo/skill-name)
	InstallsFormatted string
}
