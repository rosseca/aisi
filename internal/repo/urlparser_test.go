package repo

import (
	"strings"
	"testing"
)

func TestParseSkillURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantRepoURL string
		wantRef     string
		wantPath    string
		wantIsLocal bool
		wantErr     bool
	}{
		{
			name:        "GitHub shorthand",
			input:       "vercel-labs/agent-skills",
			wantRepoURL: "https://github.com/vercel-labs/agent-skills",
			wantRef:     "main",
			wantPath:    ".",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "GitHub shorthand with @skill",
			input:       "microsoft/github-copilot-for-azure@azure-hosted-copilot-sdk",
			wantRepoURL: "https://github.com/microsoft/github-copilot-for-azure",
			wantRef:     "main",
			wantPath:    "azure-hosted-copilot-sdk",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "GitHub shorthand with @skill - vercel",
			input:       "vercel-labs/agent-skills@vercel-react-best-practices",
			wantRepoURL: "https://github.com/vercel-labs/agent-skills",
			wantRef:     "main",
			wantPath:    "vercel-react-best-practices",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "GitHub HTTPS URL",
			input:       "https://github.com/vercel-labs/agent-skills",
			wantRepoURL: "https://github.com/vercel-labs/agent-skills",
			wantRef:     "main",
			wantPath:    ".",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "GitHub HTTPS URL with .git",
			input:       "https://github.com/vercel-labs/agent-skills.git",
			wantRepoURL: "https://github.com/vercel-labs/agent-skills",
			wantRef:     "main",
			wantPath:    ".",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "GitHub tree URL",
			input:       "https://github.com/vercel-labs/agent-skills/tree/main/skills/web-design-guidelines",
			wantRepoURL: "https://github.com/vercel-labs/agent-skills",
			wantRef:     "main",
			wantPath:    "skills/web-design-guidelines",
			wantIsLocal: false,
			wantErr:     false,
	},
		{
			name:        "GitHub blob URL",
			input:       "https://github.com/vercel-labs/agent-skills/blob/main/skills/web-design-guidelines",
			wantRepoURL: "https://github.com/vercel-labs/agent-skills",
			wantRef:     "main",
			wantPath:    "skills/web-design-guidelines",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "GitHub tree URL with specific branch",
			input:       "https://github.com/vercel-labs/agent-skills/tree/develop/skills/test",
			wantRepoURL: "https://github.com/vercel-labs/agent-skills",
			wantRef:     "develop",
			wantPath:    "skills/test",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "GitLab HTTPS URL",
			input:       "https://gitlab.com/org/repo",
			wantRepoURL: "https://gitlab.com/org/repo",
			wantRef:     "main",
			wantPath:    ".",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "SSH git URL",
			input:       "git@github.com:vercel-labs/agent-skills.git",
			wantRepoURL: "git@github.com:vercel-labs/agent-skills.git",
			wantRef:     "main",
			wantPath:    ".",
			wantIsLocal: false,
			wantErr:     false,
		},
		{
			name:        "Relative local path with ./",
			input:       "./my-local-skills",
			wantRepoURL: "", // Will be set to absolute path
			wantRef:     "",
			wantPath:    "", // Will be set to absolute path
			wantIsLocal: true,
			wantErr:     false,
		},
		{
			name:        "Relative local path with ../",
			input:       "../my-skills",
			wantRepoURL: "", // Will be set to absolute path
			wantRef:     "",
			wantPath:    "", // Will be set to absolute path
			wantIsLocal: true,
			wantErr:     false,
		},
		{
			name:        "Empty input",
			input:       "",
			wantErr:     true,
		},
		{
			name:        "Invalid shorthand - multiple slashes",
			input:       "owner/repo/extra",
			wantErr:     true,
		},
		{
			name:        "Invalid shorthand - single part",
			input:       "repo-only",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSkillURL(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseSkillURL() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseSkillURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.IsLocal != tt.wantIsLocal {
				t.Errorf("IsLocal = %v, want %v", got.IsLocal, tt.wantIsLocal)
			}

			if !tt.wantIsLocal {
				if got.RepoURL != tt.wantRepoURL {
					t.Errorf("RepoURL = %v, want %v", got.RepoURL, tt.wantRepoURL)
				}
				if got.Ref != tt.wantRef {
					t.Errorf("Ref = %v, want %v", got.Ref, tt.wantRef)
				}
				if got.Path != tt.wantPath {
					t.Errorf("Path = %v, want %v", got.Path, tt.wantPath)
				}
			}

			if tt.wantIsLocal {
				// For local paths, just verify they were resolved to absolute paths
				if !strings.HasPrefix(got.Path, "/") && !strings.Contains(got.Path, ":") {
					t.Errorf("Local path not absolute: %v", got.Path)
				}
			}
		})
	}
}

func TestSkillURL_GetSkillName(t *testing.T) {
	tests := []struct {
		name     string
		url      *SkillURL
		expected string
	}{
		{
			name: "From tree path",
			url: &SkillURL{
				RepoURL: "https://github.com/vercel-labs/agent-skills",
				Ref:     "main",
				Path:    "skills/web-design-guidelines",
				IsLocal: false,
			},
			expected: "web-design-guidelines",
		},
		{
			name: "From repo root",
			url: &SkillURL{
				RepoURL: "https://github.com/vercel-labs/agent-skills",
				Ref:     "main",
				Path:    ".",
				IsLocal: false,
			},
			expected: "agent-skills",
		},
		{
			name: "From repo with .git suffix",
			url: &SkillURL{
				RepoURL: "https://github.com/user/skill-repo.git",
				Ref:     "main",
				Path:    ".",
				IsLocal: false,
			},
			expected: "skill-repo",
		},
		{
			name: "From local path",
			url: &SkillURL{
				RepoURL: "/home/user/skills/my-cool-skill",
				Ref:     "",
				Path:    "/home/user/skills/my-cool-skill",
				IsLocal: true,
			},
			expected: "my-cool-skill",
		},
		{
			name: "Nested path with special chars",
			url: &SkillURL{
				RepoURL: "https://github.com/user/repo",
				Ref:     "main",
				Path:    "skills/Some Cool Skill!V2",
				IsLocal: false,
			},
			expected: "some-cool-skill-v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.url.GetSkillName()
			if got != tt.expected {
				t.Errorf("GetSkillName() = %v, want %v", got, tt.expected)
			}
		})
	}
}
