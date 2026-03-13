package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Empty(t, cfg.Repo.URL)
	assert.Equal(t, "main", cfg.Repo.Branch)
	assert.Equal(t, "cursor", cfg.ActiveTarget)
	assert.NotNil(t, cfg.CustomTargets)
	assert.Empty(t, cfg.CustomTargets)
}

func TestConfig_SetRepo(t *testing.T) {
	cfg := DefaultConfig()

	cfg.SetRepo("git@github.com:company/repo.git", "develop")

	assert.Equal(t, "git@github.com:company/repo.git", cfg.Repo.URL)
	assert.Equal(t, "develop", cfg.Repo.Branch)
}

func TestConfig_SetRepo_EmptyBranch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Repo.Branch = "main"

	cfg.SetRepo("git@github.com:company/repo.git", "")

	assert.Equal(t, "main", cfg.Repo.Branch)
}

func TestConfig_SetActiveTarget(t *testing.T) {
	cfg := DefaultConfig()

	cfg.SetActiveTarget("kilo")

	assert.Equal(t, "kilo", cfg.ActiveTarget)
}

func TestConfig_IsConfigured(t *testing.T) {
	cfg := DefaultConfig()
	// Default config has empty repo URL, so it's not configured
	assert.False(t, cfg.IsConfigured())

	// Setting a repo URL makes it configured
	cfg.SetRepo("git@github.com:company/repo.git", "main")
	assert.True(t, cfg.IsConfigured())

	// Clearing the repo URL makes it unconfigured again
	cfg.SetRepo("", "")
	assert.False(t, cfg.IsConfigured())
}

func TestConfig_GetToken(t *testing.T) {
	cfg := DefaultConfig()

	os.Setenv("GITHUB_TOKEN", "env-token")
	defer os.Unsetenv("GITHUB_TOKEN")

	assert.Equal(t, "env-token", cfg.GetToken())

	cfg.SetHTTPSToken("config-token")
	assert.Equal(t, "config-token", cfg.GetToken())
}

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := DefaultConfig()
	cfg.SetRepo("git@github.com:test/repo.git", "develop")
	cfg.SetActiveTarget("kilo")
	cfg.CustomTargets["custom"] = CustomTarget{
		DisplayName: "Custom",
		ConfigDir:   ".custom",
		RulesDir:    "rules",
	}

	err := cfg.Save()
	require.NoError(t, err)

	loaded, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "git@github.com:test/repo.git", loaded.Repo.URL)
	assert.Equal(t, "develop", loaded.Repo.Branch)
	assert.Equal(t, "kilo", loaded.ActiveTarget)
	assert.Equal(t, "Custom", loaded.CustomTargets["custom"].DisplayName)
}

func TestLoad_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg, err := Load()
	require.NoError(t, err)

	// Default config has empty repo URL
	assert.Empty(t, cfg.Repo.URL)
	assert.Equal(t, "main", cfg.Repo.Branch)
	assert.Equal(t, "cursor", cfg.ActiveTarget)
}

func TestLoadWithExists_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg, exists, err := LoadWithExists()
	require.NoError(t, err)
	assert.False(t, exists, "should return false when config file doesn't exist")

	// Default config is still returned with empty repo URL
	assert.Empty(t, cfg.Repo.URL)
	assert.Equal(t, "cursor", cfg.ActiveTarget)
}

func TestLoadWithExists_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create a config file
	cfg := DefaultConfig()
	cfg.SetActiveTarget("kilo")
	err := cfg.Save()
	require.NoError(t, err)

	// Load and verify exists is true
	loadedCfg, exists, err := LoadWithExists()
	require.NoError(t, err)
	assert.True(t, exists, "should return true when config file exists")
	assert.Equal(t, "kilo", loadedCfg.ActiveTarget)
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	require.NoError(t, err)
	assert.Contains(t, dir, ConfigDirName)
}

func TestCacheDir(t *testing.T) {
	dir, err := CacheDir()
	require.NoError(t, err)
	assert.Contains(t, dir, CacheDirName)
}

func TestExternalCacheDir(t *testing.T) {
	dir, err := ExternalCacheDir()
	require.NoError(t, err)
	assert.Contains(t, dir, "external")
}

func TestEnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	err := EnsureConfigDir()
	require.NoError(t, err)

	expectedDir := filepath.Join(tmpDir, ConfigDirName)
	_, err = os.Stat(expectedDir)
	assert.NoError(t, err)
}

func TestEnsureCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	err := EnsureCacheDir()
	require.NoError(t, err)

	expectedDir := filepath.Join(tmpDir, ConfigDirName, CacheDirName)
	_, err = os.Stat(expectedDir)
	assert.NoError(t, err)
}
