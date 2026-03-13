package tracker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestTracker(t *testing.T) (*Tracker, string) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	tracker := New(projectDir, targets.CursorTarget)
	return tracker, projectDir
}

func TestTracker_LoadEmpty(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	lock, err := tracker.Load()
	require.NoError(t, err)
	assert.NotNil(t, lock)
	assert.Empty(t, lock.Assets.Rules)
	assert.Empty(t, lock.Assets.Skills)
}

func TestTracker_SaveAndLoad(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	lock := &LockFile{
		InstalledAt: "2026-03-12T10:00:00Z",
		RepoCommit:  "abc123",
		Target:      "cursor",
		Assets: InstalledAssets{
			Rules:  []string{"rule1", "rule2"},
			Skills: []SkillEntry{{Name: "skill1"}},
		},
	}

	err := tracker.Save(lock)
	require.NoError(t, err)

	loaded, err := tracker.Load()
	require.NoError(t, err)
	assert.Equal(t, "abc123", loaded.RepoCommit)
	assert.Equal(t, "cursor", loaded.Target)
	assert.Equal(t, []string{"rule1", "rule2"}, loaded.Assets.Rules)
	assert.Len(t, loaded.Assets.Skills, 1)
	assert.Equal(t, "skill1", loaded.Assets.Skills[0].Name)
}

func TestTracker_RecordInstall(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	err := tracker.RecordInstall(manifest.AssetTypeRule, "my-rule", "https://github.com/test/repo", "commit1")
	require.NoError(t, err)

	err = tracker.RecordInstall(manifest.AssetTypeSkill, "my-skill", "https://github.com/test/repo", "commit2")
	require.NoError(t, err)

	lock, err := tracker.Load()
	require.NoError(t, err)

	assert.Contains(t, lock.Assets.Rules, "my-rule")
	// Skills are stored as SkillEntry structs
	skillNames := make([]string, len(lock.Assets.Skills))
	for i, s := range lock.Assets.Skills {
		skillNames[i] = s.Name
	}
	assert.Contains(t, skillNames, "my-skill")
	assert.Equal(t, "commit2", lock.RepoCommit)
	assert.Equal(t, "https://github.com/test/repo", lock.RepoURL)
}

func TestTracker_RecordSkillInstall(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	// Install skill with full source information
	skillEntry := SkillEntry{
		Name:   "online-skill",
		Source: "owner/repo",
		Path:   "skills/online",
		Commit: "abc123",
	}
	err := tracker.RecordSkillInstall(skillEntry, "https://github.com/main/repo", "commit1")
	require.NoError(t, err)

	lock, err := tracker.Load()
	require.NoError(t, err)

	require.Len(t, lock.Assets.Skills, 1)
	assert.Equal(t, "online-skill", lock.Assets.Skills[0].Name)
	assert.Equal(t, "owner/repo", lock.Assets.Skills[0].Source)
	assert.Equal(t, "skills/online", lock.Assets.Skills[0].Path)
	assert.Equal(t, "abc123", lock.Assets.Skills[0].Commit)
}

func TestTracker_RecordInstall_NoDuplicates(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	err := tracker.RecordInstall(manifest.AssetTypeRule, "my-rule", "https://github.com/test/repo", "commit1")
	require.NoError(t, err)

	err = tracker.RecordInstall(manifest.AssetTypeRule, "my-rule", "https://github.com/test/repo", "commit2")
	require.NoError(t, err)

	lock, err := tracker.Load()
	require.NoError(t, err)

	count := 0
	for _, r := range lock.Assets.Rules {
		if r == "my-rule" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestTracker_RecordInstalls(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	records := []InstallRecord{
		{Name: "rule1", Type: manifest.AssetTypeRule},
		{Name: "rule2", Type: manifest.AssetTypeRule},
		{Name: "skill1", Type: manifest.AssetTypeSkill},
		{Name: "mcp1", Type: manifest.AssetTypeMCP},
	}

	err := tracker.RecordInstalls(records, "https://github.com/test/repo", "commit123")
	require.NoError(t, err)

	lock, err := tracker.Load()
	require.NoError(t, err)

	assert.Len(t, lock.Assets.Rules, 2)
	assert.Len(t, lock.Assets.Skills, 1)
	assert.Len(t, lock.Assets.MCP, 1)
}

func TestTracker_IsInstalled(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	err := tracker.RecordInstall(manifest.AssetTypeRule, "my-rule", "https://github.com/test/repo", "commit1")
	require.NoError(t, err)

	installed, err := tracker.IsInstalled(manifest.AssetTypeRule, "my-rule")
	require.NoError(t, err)
	assert.True(t, installed)

	installed, err = tracker.IsInstalled(manifest.AssetTypeRule, "other-rule")
	require.NoError(t, err)
	assert.False(t, installed)

	installed, err = tracker.IsInstalled(manifest.AssetTypeSkill, "my-rule")
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestTracker_GetInstalled(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	err := tracker.RecordInstall(manifest.AssetTypeRule, "rule1", "https://github.com/test/repo", "commit1")
	require.NoError(t, err)
	err = tracker.RecordInstall(manifest.AssetTypeSkill, "skill1", "https://github.com/test/repo", "commit2")
	require.NoError(t, err)

	assets, err := tracker.GetInstalled()
	require.NoError(t, err)

	assert.Contains(t, assets.Rules, "rule1")
	// Skills are SkillEntry structs
	skillNames := make([]string, len(assets.Skills))
	for i, s := range assets.Skills {
		skillNames[i] = s.Name
	}
	assert.Contains(t, skillNames, "skill1")
}

func TestTracker_GetRepoCommit(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	commit, err := tracker.GetRepoCommit()
	require.NoError(t, err)
	assert.Empty(t, commit)

	err = tracker.RecordInstall(manifest.AssetTypeRule, "rule1", "https://github.com/test/repo", "abc123")
	require.NoError(t, err)

	commit, err = tracker.GetRepoCommit()
	require.NoError(t, err)
	assert.Equal(t, "abc123", commit)
}

func TestTracker_Remove(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	err := tracker.RecordInstall(manifest.AssetTypeRule, "rule1", "https://github.com/test/repo", "commit1")
	require.NoError(t, err)
	err = tracker.RecordInstall(manifest.AssetTypeRule, "rule2", "https://github.com/test/repo", "commit1")
	require.NoError(t, err)

	err = tracker.Remove(manifest.AssetTypeRule, "rule1")
	require.NoError(t, err)

	lock, err := tracker.Load()
	require.NoError(t, err)

	assert.NotContains(t, lock.Assets.Rules, "rule1")
	assert.Contains(t, lock.Assets.Rules, "rule2")
}

func TestTracker_AllAssetTypes(t *testing.T) {
	tracker, _ := setupTestTracker(t)

	tests := []struct {
		assetType manifest.AssetType
		name      string
	}{
		{manifest.AssetTypeRule, "rule1"},
		{manifest.AssetTypeSkill, "skill1"},
		{manifest.AssetTypeAgent, "agent1"},
		{manifest.AssetTypeHook, "hook1"},
		{manifest.AssetTypeMCP, "mcp1"},
		{manifest.AssetTypeAgentsMD, "agents-md1"},
		{manifest.AssetType("external"), "external1"},
	}

	for _, tt := range tests {
		err := tracker.RecordInstall(tt.assetType, tt.name, "https://github.com/test/repo", "commit")
		require.NoError(t, err)

		installed, err := tracker.IsInstalled(tt.assetType, tt.name)
		require.NoError(t, err)
		assert.True(t, installed, "asset %s of type %s should be installed", tt.name, tt.assetType)
	}

	lock, err := tracker.Load()
	require.NoError(t, err)

	assert.Len(t, lock.Assets.Rules, 1)
	assert.Len(t, lock.Assets.Skills, 1)
	assert.Len(t, lock.Assets.Agents, 1)
	assert.Len(t, lock.Assets.Hooks, 1)
	assert.Len(t, lock.Assets.MCP, 1)
	assert.Len(t, lock.Assets.AgentsMD, 1)
	assert.Len(t, lock.Assets.External, 1)
}

// TestTracker_BackwardCompatibility verifies that old lock files with skills as strings
// can still be loaded correctly.
func TestTracker_BackwardCompatibility(t *testing.T) {
	tracker, projectDir := setupTestTracker(t)

	// Create an old-style lock file with skills as strings
	oldLockContent := `{
  "installedAt": "2026-03-12T10:00:00Z",
  "repoURL": "https://github.com/test/repo",
  "repoCommit": "abc123",
  "target": "cursor",
  "assets": {
    "rules": ["rule1", "rule2"],
    "skills": ["legacy-skill-1", "legacy-skill-2"]
  }
}`

	lockPath := filepath.Join(projectDir, targets.CursorTarget.ConfigDir, LockFileName)
	err := os.MkdirAll(filepath.Dir(lockPath), 0755)
	require.NoError(t, err)

	err = os.WriteFile(lockPath, []byte(oldLockContent), 0644)
	require.NoError(t, err)

	// Load the old lock file
	lock, err := tracker.Load()
	require.NoError(t, err)

	// Verify skills were loaded correctly (converted from strings to SkillEntry)
	assert.Len(t, lock.Assets.Skills, 2)
	assert.Equal(t, "legacy-skill-1", lock.Assets.Skills[0].Name)
	assert.Equal(t, "legacy-skill-2", lock.Assets.Skills[1].Name)
	// Source should be empty for legacy skills
	assert.Empty(t, lock.Assets.Skills[0].Source)
	assert.Empty(t, lock.Assets.Skills[1].Source)

	// Verify other assets still work
	assert.Equal(t, []string{"rule1", "rule2"}, lock.Assets.Rules)
}
