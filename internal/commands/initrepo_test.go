package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDirectory(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "creates non-existent directory",
			setup: func() string {
				return filepath.Join(t.TempDir(), "new-repo")
			},
			wantErr: false,
		},
		{
			name: "accepts empty directory",
			setup: func() string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "accepts directory with only .git",
			setup: func() string {
				dir := t.TempDir()
				_ = os.Mkdir(filepath.Join(dir, ".git"), 0755)
				return dir
			},
			wantErr: false,
		},
		{
			name: "rejects non-empty directory",
			setup: func() string {
				dir := t.TempDir()
				_ = os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("content"), 0644)
				return dir
			},
			wantErr:     true,
			errContains: "not empty",
		},
		{
			name: "rejects file path",
			setup: func() string {
				dir := t.TempDir()
				path := filepath.Join(dir, "file.txt")
				_ = os.WriteFile(path, []byte("content"), 0644)
				return path
			},
			wantErr:     true,
			errContains: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			err := validateDirectory(path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateDirectory() expected error but got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("validateDirectory() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateDirectory() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestCreateDirectoryStructure(t *testing.T) {
	dir := t.TempDir()

	err := createDirectoryStructure(dir)
	if err != nil {
		t.Fatalf("createDirectoryStructure() error = %v", err)
	}

	// Check all directories exist
	for _, d := range repoDirs {
		path := filepath.Join(dir, d)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Directory %s not created: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}
}

func TestCreateGitkeepFiles(t *testing.T) {
	dir := t.TempDir()

	// Create directory structure first
	if err := createDirectoryStructure(dir); err != nil {
		t.Fatalf("createDirectoryStructure() error = %v", err)
	}

	// Then create gitkeep files
	if err := createGitkeepFiles(dir); err != nil {
		t.Fatalf("createGitkeepFiles() error = %v", err)
	}

	// Check gitkeep files exist
	keepDirs := []string{"rules", "skills", "agents", "hooks/scripts", "mcp", "agents-md"}
	for _, d := range keepDirs {
		gitkeepPath := filepath.Join(dir, d, ".gitkeep")
		if _, err := os.Stat(gitkeepPath); err != nil {
			t.Errorf(".gitkeep not found in %s: %v", d, err)
		}
	}
}

func TestCreateManifestRule(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".cursor/rules"), 0755)

	err := createManifestRule(dir)
	if err != nil {
		t.Fatalf("createManifestRule() error = %v", err)
	}

	rulePath := filepath.Join(dir, ".cursor/rules/manifest-generation.mdc")
	content, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("Failed to read manifest rule: %v", err)
	}

	// Check it contains expected content
	if !contains(string(content), "AISI Manifest Generation Rule") {
		t.Error("Manifest rule missing expected content")
	}
}

func TestCreateExampleManifest(t *testing.T) {
	dir := t.TempDir()

	err := createExampleManifest(dir)
	if err != nil {
		t.Fatalf("createExampleManifest() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "manifest.yaml")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest.yaml: %v", err)
	}

	// Check it contains expected content
	if !contains(string(content), "version") {
		t.Error("Manifest missing version field")
	}
	if !contains(string(content), "rules:") {
		t.Error("Manifest missing rules section")
	}
}

func TestCreateReadme(t *testing.T) {
	dir := t.TempDir()

	err := createReadme(dir)
	if err != nil {
		t.Fatalf("createReadme() error = %v", err)
	}

	readmePath := filepath.Join(dir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	// Check it contains expected sections
	contentStr := string(content)
	if !contains(contentStr, "AISI Shared Intelligence Repository") {
		t.Error("README missing AISI reference")
	}
	if !contains(contentStr, "## Repository Structure") {
		t.Error("README missing structure section")
	}
	if !contains(contentStr, "## Quick Start") {
		t.Error("README missing quick start section")
	}
}

func TestCreateReadmeSkipsExisting(t *testing.T) {
	dir := t.TempDir()

	// Create existing README
	existingContent := "# My Existing README"
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte(existingContent), 0644)

	// Try to create again
	err := createReadme(dir)
	if err != nil {
		t.Fatalf("createReadme() should not error on existing file: %v", err)
	}

	// Verify original content preserved
	content, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if string(content) != existingContent {
		t.Error("Existing README was overwritten")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
