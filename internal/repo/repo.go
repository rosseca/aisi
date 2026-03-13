package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rosseca/aisi/internal/config"
)

type Manager struct {
	git      GitRunner
	cacheDir string
	extDir   string
	repoURL  string
	branch   string
	token    string
}

func NewManager(cfg *config.Config) (*Manager, error) {
	cacheDir, err := config.CacheDir()
	if err != nil {
		return nil, err
	}

	extDir, err := config.ExternalCacheDir()
	if err != nil {
		return nil, err
	}

	return &Manager{
		git:      NewGitRunner(),
		cacheDir: cacheDir,
		extDir:   extDir,
		repoURL:  cfg.Repo.URL,
		branch:   cfg.Repo.Branch,
		token:    cfg.GetToken(),
	}, nil
}

func NewManagerWithGit(cfg *config.Config, git GitRunner) (*Manager, error) {
	m, err := NewManager(cfg)
	if err != nil {
		return nil, err
	}
	m.git = git
	return m, nil
}

func (m *Manager) repoNameFromURL(url string) string {
	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
	}

	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "-" + parts[len(parts)-1]
	}
	return parts[len(parts)-1]
}

func (m *Manager) MainRepoPath() string {
	// If it's a local path, use it directly
	if m.isLocalPath(m.repoURL) {
		return m.repoURL
	}
	return filepath.Join(m.cacheDir, m.repoNameFromURL(m.repoURL))
}

func (m *Manager) isLocalPath(url string) bool {
	// Check if it's an absolute path or starts with ./ or ../
	if filepath.IsAbs(url) {
		return true
	}
	if strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../") {
		return true
	}
	// Check if it exists as a directory
	if info, err := os.Stat(url); err == nil && info.IsDir() {
		return true
	}
	return false
}

func (m *Manager) ExternalRepoPath(repoURL string) string {
	return filepath.Join(m.extDir, m.repoNameFromURL(repoURL))
}

func (m *Manager) EnsureMainRepo() error {
	if m.repoURL == "" {
		return fmt.Errorf("repository URL not configured")
	}

	// For local paths, just verify it exists
	if m.isLocalPath(m.repoURL) {
		if _, err := os.Stat(m.repoURL); err != nil {
			return fmt.Errorf("local repository path not found: %w", err)
		}
		return nil
	}

	repoPath := m.MainRepoPath()

	if m.repoExists(repoPath) {
		return nil
	}

	if err := os.MkdirAll(m.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	url := m.getCloneURL(m.repoURL)
	if err := m.git.Clone(url, repoPath, 1); err != nil {
		return fmt.Errorf("failed to clone main repository: %w", err)
	}

	return nil
}

func (m *Manager) EnsureExternalRepo(repoURL, ref string) (string, error) {
	repoPath := m.ExternalRepoPath(repoURL)

	if m.repoExists(repoPath) {
		if ref != "" && ref != "main" && ref != "master" {
			if err := m.git.Checkout(repoPath, ref); err != nil {
				return "", fmt.Errorf("failed to checkout ref %s: %w", ref, err)
			}
		}
		return repoPath, nil
	}

	if err := os.MkdirAll(m.extDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create external cache directory: %w", err)
	}

	url := m.getCloneURL(repoURL)

	// Verify repository access before attempting to clone
	if err := m.git.VerifyRepoAccess(url); err != nil {
		return "", fmt.Errorf("repository not accessible: %w", err)
	}

	if err := m.git.Clone(url, repoPath, 1); err != nil {
		return "", fmt.Errorf("failed to clone external repository: %w", err)
	}

	if ref != "" && ref != "main" && ref != "master" {
		if err := m.git.Checkout(repoPath, ref); err != nil {
			return "", fmt.Errorf("failed to checkout ref %s: %w", ref, err)
		}
	}

	return repoPath, nil
}

func (m *Manager) UpdateMainRepo() error {
	repoPath := m.MainRepoPath()

	if !m.repoExists(repoPath) {
		return m.EnsureMainRepo()
	}

	if err := m.git.Pull(repoPath); err != nil {
		return fmt.Errorf("failed to update main repository: %w", err)
	}

	return nil
}

func (m *Manager) UpdateExternalRepo(repoURL string) error {
	repoPath := m.ExternalRepoPath(repoURL)

	if !m.repoExists(repoPath) {
		_, err := m.EnsureExternalRepo(repoURL, "")
		return err
	}

	if err := m.git.Pull(repoPath); err != nil {
		return fmt.Errorf("failed to update external repository: %w", err)
	}

	return nil
}

func (m *Manager) GetCurrentCommit() (string, error) {
	repoPath := m.MainRepoPath()

	// For local paths, try to get the git commit if it's a git repo
	if m.isLocalPath(m.repoURL) {
		if m.repoExists(repoPath) {
			return m.git.GetCurrentCommit(repoPath)
		}
		return "local", nil
	}

	return m.git.GetCurrentCommit(repoPath)
}

func (m *Manager) repoExists(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

func (m *Manager) getCloneURL(repoURL string) string {
	if strings.HasPrefix(repoURL, "git@") {
		return repoURL
	}

	if strings.HasPrefix(repoURL, "https://") {
		if m.token != "" {
			return strings.Replace(repoURL, "https://", "https://"+m.token+"@", 1)
		}
		return repoURL
	}

	if strings.HasPrefix(repoURL, "github.com/") {
		return "git@github.com:" + strings.TrimPrefix(repoURL, "github.com/") + ".git"
	}

	return repoURL
}

func (m *Manager) GetFilePath(relativePath string) string {
	return filepath.Join(m.MainRepoPath(), relativePath)
}

func (m *Manager) GetExternalFilePath(repoURL, relativePath string) string {
	return filepath.Join(m.ExternalRepoPath(repoURL), relativePath)
}

func (m *Manager) ReadFile(relativePath string) ([]byte, error) {
	fullPath := m.GetFilePath(relativePath)
	return os.ReadFile(fullPath)
}

func (m *Manager) ReadExternalFile(repoURL, relativePath string) ([]byte, error) {
	fullPath := m.GetExternalFilePath(repoURL, relativePath)
	return os.ReadFile(fullPath)
}

// GetManifestPath looks for manifest files in order of preference:
// 1. manifest.yaml
// 2. manifest.yml
// 3. manifest.json
// Returns the path to the first found manifest file, or the default (yaml) if none exist
func (m *Manager) GetManifestPath() string {
	basePath := m.MainRepoPath()
	candidates := []string{
		filepath.Join(basePath, "manifest.yaml"),
		filepath.Join(basePath, "manifest.yml"),
		filepath.Join(basePath, "manifest.json"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to yaml if none found (will fail later with proper error)
	return candidates[0]
}

// GetExternalManifestPath looks for manifest files in external repos
func (m *Manager) GetExternalManifestPath(repoURL string) string {
	basePath := m.ExternalRepoPath(repoURL)
	candidates := []string{
		filepath.Join(basePath, "manifest.yaml"),
		filepath.Join(basePath, "manifest.yml"),
		filepath.Join(basePath, "manifest.json"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return candidates[0]
}
