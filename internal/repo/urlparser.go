package repo

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// SkillURL represents a parsed skill URL with all components needed for installation
type SkillURL struct {
	RepoURL string // Cloneable URL (e.g., https://github.com/owner/repo or git@github.com:owner/repo.git)
	Ref     string // Branch/tag (default: "main")
	Path    string // Path within repo (default: ".")
	IsLocal bool   // True if local path (no git clone needed)
}

// ParseSkillURL parses various URL formats into a SkillURL struct.
// Supports:
//   - GitHub shorthand: "owner/repo" → https://github.com/owner/repo
//   - Full URLs: "https://github.com/owner/repo"
//   - Tree/blob paths: "https://github.com/owner/repo/tree/main/skills/foo"
//   - GitLab URLs: "https://gitlab.com/org/repo"
//   - SSH URLs: "git@github.com:owner/repo.git"
//   - Local paths: "./my-skills" or "/absolute/path"
func ParseSkillURL(input string) (*SkillURL, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Check for local path first
	if isLocalPath(input) {
		absPath, err := filepath.Abs(input)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve local path: %w", err)
		}
		return &SkillURL{
			RepoURL: absPath,
			Ref:     "",
			Path:    absPath,
			IsLocal: true,
		}, nil
	}

	// Parse GitHub shorthand (owner/repo or owner/repo@skill)
	if !strings.Contains(input, "://") && !strings.HasPrefix(input, "git@") {
		// First check for @skill syntax: owner/repo@skill-name
		atSkillMatch := regexp.MustCompile(`^([^/]+)/([^/@]+)@(.+)$`).FindStringSubmatch(input)
		if atSkillMatch != nil {
			owner := atSkillMatch[1]
			repo := atSkillMatch[2]
			skillPath := atSkillMatch[3]
			return &SkillURL{
				RepoURL: fmt.Sprintf("https://github.com/%s/%s", owner, repo),
				Ref:     "main",
				Path:    skillPath,
				IsLocal: false,
			}, nil
		}

		// Simple owner/repo format
		parts := strings.Split(input, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return &SkillURL{
				RepoURL: fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1]),
				Ref:     "main",
				Path:    ".",
				IsLocal: false,
			}, nil
		}
		return nil, fmt.Errorf("invalid shorthand format: %s (expected owner/repo or owner/repo@skill)", input)
	}

	// Parse HTTPS/HTTP URLs (including tree/blob paths)
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return parseHTTPURL(input)
	}

	// Parse SSH URLs (git@host:owner/repo.git)
	if strings.HasPrefix(input, "git@") {
		return parseSSHURL(input)
	}

	return nil, fmt.Errorf("unsupported URL format: %s", input)
}

// isLocalPath checks if the input is a local filesystem path
func isLocalPath(input string) bool {
	// Check for relative path indicators
	if strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") {
		return true
	}

	// Check for absolute path
	if filepath.IsAbs(input) {
		return true
	}

	// Check if it exists as a directory (for paths without ./)
	// We only check for single-segment paths that don't look like URLs
	if !strings.Contains(input, "/") && !strings.Contains(input, "\\") {
		// Single word like "skills" - could be a local dir
		// For single words, let's be conservative and require explicit ./ or check if exists
		if _, err := filepath.Abs(input); err == nil {
			// Actually, single words without ./ are more likely to be errors
			// unless they exist. Let's be conservative and only treat obvious
			// local paths as local.
			return false
		}
	}

	return false
}

// parseHTTPURL parses HTTP(S) URLs, including tree/blob paths
func parseHTTPURL(input string) (*SkillURL, error) {
	// Handle GitHub tree/blob URLs
	// Pattern: https://github.com/owner/repo/tree/branch/path/to/skill
	// Pattern: https://github.com/owner/repo/blob/branch/path/to/skill
	if strings.Contains(input, "github.com") {
		return parseGitHubURL(input)
	}

	// Handle GitLab and other Git hosting platforms
	// For these, we don't extract tree/blob paths (more complex patterns)
	// Just use the URL as-is for cloning
	return &SkillURL{
		RepoURL: input,
		Ref:     "main",
		Path:    ".",
		IsLocal: false,
	}, nil
}

// parseGitHubURL parses GitHub URLs including tree/blob paths
func parseGitHubURL(input string) (*SkillURL, error) {
	// Remove .git suffix if present for tree/blob URLs
	input = strings.TrimSuffix(input, ".git")

	// Regex for GitHub tree/blob URLs
	// Matches: github.com/owner/repo/tree/branch/path
	// Matches: github.com/owner/repo/blob/branch/path
	treeBlobPattern := regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/(tree|blob)/([^/]+)(?:/(.+))?$`)

	matches := treeBlobPattern.FindStringSubmatch(input)
	if matches != nil {
		owner := matches[1]
		repo := matches[2]
		ref := matches[4]     // branch/tag
		path := matches[5]    // path within repo (may be empty)

		if path == "" {
			path = "."
		}

		return &SkillURL{
			RepoURL: fmt.Sprintf("https://github.com/%s/%s", owner, repo),
			Ref:     ref,
			Path:    path,
			IsLocal: false,
		}, nil
	}

	// Regular GitHub repo URL (no tree/blob)
	// Pattern: github.com/owner/repo or github.com/owner/repo.git
	repoPattern := regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)

	matches = repoPattern.FindStringSubmatch(input)
	if matches != nil {
		owner := matches[1]
		repo := matches[2]

		return &SkillURL{
			RepoURL: fmt.Sprintf("https://github.com/%s/%s", owner, repo),
			Ref:     "main",
			Path:    ".",
			IsLocal: false,
		}, nil
	}

	return nil, fmt.Errorf("unable to parse GitHub URL: %s", input)
}

// parseSSHURL parses SSH git URLs
// Format: git@github.com:owner/repo.git
func parseSSHURL(input string) (*SkillURL, error) {
	// SSH URL pattern: git@host:path/to/repo.git
	// The colon after host is the key differentiator
	if !strings.Contains(input, ":") {
		return nil, fmt.Errorf("invalid SSH URL format: %s", input)
	}

	// Remove .git suffix for consistency
	input = strings.TrimSuffix(input, ".git")

	return &SkillURL{
		RepoURL: input + ".git", // Add .git back for clone
		Ref:     "main",
		Path:    ".",
		IsLocal: false,
	}, nil
}

// GetSkillName returns a suggested skill name based on the URL
// Uses the last path component, optionally sanitized
func (s *SkillURL) GetSkillName() string {
	if s.IsLocal {
		return filepath.Base(s.Path)
	}

	if s.Path != "." && s.Path != "" {
		// Use the last component of the path within repo
		parts := strings.Split(s.Path, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" {
				return sanitizeSkillName(parts[i])
			}
		}
	}

	// Use repo name
	repoName := filepath.Base(s.RepoURL)
	repoName = strings.TrimSuffix(repoName, ".git")
	return sanitizeSkillName(repoName)
}

// sanitizeSkillName cleans up a name to be a valid skill directory name
func sanitizeSkillName(name string) string {
	// Replace spaces and special characters with hyphens
	name = strings.ToLower(name)
	name = regexp.MustCompile(`[^a-z0-9_-]+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}
