package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rosseca/aisi/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockGitRunner struct {
	CloneFunc            func(url, dest string, depth int) error
	PullFunc             func(repoPath string) error
	GetRemoteURLFunc     func(repoPath string) (string, error)
	GetCurrentCommitFunc func(repoPath string) (string, error)
	CheckoutFunc         func(repoPath, ref string) error
	VerifyRepoAccessFunc func(url string) error

	CloneCalls    []CloneCall
	PullCalls     []string
	CheckoutCalls []CheckoutCall
}

type CloneCall struct {
	URL   string
	Dest  string
	Depth int
}

type CheckoutCall struct {
	RepoPath string
	Ref      string
}

func (m *MockGitRunner) Clone(url, dest string, depth int) error {
	m.CloneCalls = append(m.CloneCalls, CloneCall{url, dest, depth})
	if m.CloneFunc != nil {
		return m.CloneFunc(url, dest, depth)
	}
	if err := os.MkdirAll(filepath.Join(dest, ".git"), 0755); err != nil {
		return err
	}
	return nil
}

func (m *MockGitRunner) Pull(repoPath string) error {
	m.PullCalls = append(m.PullCalls, repoPath)
	if m.PullFunc != nil {
		return m.PullFunc(repoPath)
	}
	return nil
}

func (m *MockGitRunner) GetRemoteURL(repoPath string) (string, error) {
	if m.GetRemoteURLFunc != nil {
		return m.GetRemoteURLFunc(repoPath)
	}
	return "git@github.com:test/repo.git", nil
}

func (m *MockGitRunner) GetCurrentCommit(repoPath string) (string, error) {
	if m.GetCurrentCommitFunc != nil {
		return m.GetCurrentCommitFunc(repoPath)
	}
	return "abc123def456", nil
}

func (m *MockGitRunner) Checkout(repoPath, ref string) error {
	m.CheckoutCalls = append(m.CheckoutCalls, CheckoutCall{repoPath, ref})
	if m.CheckoutFunc != nil {
		return m.CheckoutFunc(repoPath, ref)
	}
	return nil
}

func (m *MockGitRunner) VerifyRepoAccess(url string) error {
	if m.VerifyRepoAccessFunc != nil {
		return m.VerifyRepoAccessFunc(url)
	}
	return nil
}

func setupTestManager(t *testing.T) (*Manager, *MockGitRunner, string) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})

	cfg := config.DefaultConfig()
	cfg.SetRepo("git@github.com:company/mau-shared-agent-intelligence.git", "main")

	mockGit := &MockGitRunner{}
	mgr, err := NewManagerWithGit(cfg, mockGit)
	require.NoError(t, err)

	return mgr, mockGit, tmpDir
}

func TestManager_EnsureMainRepo(t *testing.T) {
	mgr, mockGit, _ := setupTestManager(t)

	err := mgr.EnsureMainRepo()
	require.NoError(t, err)

	assert.Len(t, mockGit.CloneCalls, 1)
	assert.Equal(t, "git@github.com:company/mau-shared-agent-intelligence.git", mockGit.CloneCalls[0].URL)
}

func TestManager_EnsureMainRepo_AlreadyExists(t *testing.T) {
	mgr, mockGit, tmpDir := setupTestManager(t)

	repoPath := filepath.Join(tmpDir, ".aisi", "cache", "company-mau-shared-agent-intelligence")
	err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)
	require.NoError(t, err)

	err = mgr.EnsureMainRepo()
	require.NoError(t, err)

	assert.Len(t, mockGit.CloneCalls, 0)
}

func TestManager_EnsureMainRepo_NoURL(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	cfg := config.DefaultConfig()
	cfg.SetRepo("", "") // Explicitly clear the default URL
	mockGit := &MockGitRunner{}
	mgr, err := NewManagerWithGit(cfg, mockGit)
	require.NoError(t, err)

	err = mgr.EnsureMainRepo()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestManager_UpdateMainRepo(t *testing.T) {
	mgr, mockGit, tmpDir := setupTestManager(t)

	repoPath := filepath.Join(tmpDir, ".aisi", "cache", "company-mau-shared-agent-intelligence")
	err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)
	require.NoError(t, err)

	err = mgr.UpdateMainRepo()
	require.NoError(t, err)

	assert.Len(t, mockGit.PullCalls, 1)
}

func TestManager_UpdateMainRepo_NotExists(t *testing.T) {
	mgr, mockGit, _ := setupTestManager(t)

	err := mgr.UpdateMainRepo()
	require.NoError(t, err)

	assert.Len(t, mockGit.CloneCalls, 1)
}

func TestManager_EnsureExternalRepo(t *testing.T) {
	mgr, mockGit, _ := setupTestManager(t)

	path, err := mgr.EnsureExternalRepo("github.com/twostraws/SwiftUI-Agent-Skill", "main")
	require.NoError(t, err)
	assert.Contains(t, path, "twostraws-SwiftUI-Agent-Skill")

	assert.Len(t, mockGit.CloneCalls, 1)
}

func TestManager_EnsureExternalRepo_WithRef(t *testing.T) {
	mgr, mockGit, _ := setupTestManager(t)

	path, err := mgr.EnsureExternalRepo("github.com/example/repo", "v1.2.0")
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	assert.Len(t, mockGit.CloneCalls, 1)
	assert.Len(t, mockGit.CheckoutCalls, 1)
	assert.Equal(t, "v1.2.0", mockGit.CheckoutCalls[0].Ref)
}

func TestManager_GetFilePath(t *testing.T) {
	mgr, _, _ := setupTestManager(t)

	path := mgr.GetFilePath("rules/soul.mdc")
	assert.Contains(t, path, "rules/soul.mdc")
	assert.Contains(t, path, "company-mau-shared-agent-intelligence")
}

func TestManager_repoNameFromURL(t *testing.T) {
	mgr, _, _ := setupTestManager(t)

	tests := []struct {
		url      string
		expected string
	}{
		{"git@github.com:company/repo.git", "company-repo"},
		{"git@github.com:company/repo", "company-repo"},
		{"https://github.com/company/repo.git", "company-repo"},
		{"github.com/company/repo", "company-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, mgr.repoNameFromURL(tt.url))
		})
	}
}

func TestManager_getCloneURL(t *testing.T) {
	mgr, _, _ := setupTestManager(t)

	tests := []struct {
		input    string
		expected string
	}{
		{"git@github.com:company/repo.git", "git@github.com:company/repo.git"},
		{"https://github.com/company/repo.git", "https://github.com/company/repo.git"},
		{"github.com/company/repo", "git@github.com:company/repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, mgr.getCloneURL(tt.input))
		})
	}
}

func TestManager_getCloneURL_WithToken(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	cfg := config.DefaultConfig()
	cfg.SetRepo("https://github.com/company/repo.git", "main")
	cfg.SetHTTPSToken("my-token")

	mockGit := &MockGitRunner{}
	mgr, err := NewManagerWithGit(cfg, mockGit)
	require.NoError(t, err)

	url := mgr.getCloneURL("https://github.com/company/repo.git")
	assert.Equal(t, "https://my-token@github.com/company/repo.git", url)
}
