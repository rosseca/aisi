package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillDiscovery_FindSkillByName(t *testing.T) {
	// Create a temp directory structure:
	// /tmp/xxx/
	//   skills/
	//     skill-a/
	//       SKILL.md
	//     skill-b/
	//       SKILL.md
	//   other/
	//     not-a-skill/
	//       random.md
	//     skill-c/
	//       SKILL.md
	tmpDir := t.TempDir()

	// Create skill directories with SKILL.md files
	skillA := filepath.Join(tmpDir, "skills", "skill-a")
	skillB := filepath.Join(tmpDir, "skills", "skill-b")
	skillC := filepath.Join(tmpDir, "other", "skill-c")

	require.NoError(t, os.MkdirAll(skillA, 0755))
	require.NoError(t, os.MkdirAll(skillB, 0755))
	require.NoError(t, os.MkdirAll(skillC, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "other", "not-a-skill"), 0755))

	// Create SKILL.md files
	require.NoError(t, os.WriteFile(filepath.Join(skillA, "SKILL.md"), []byte("# Skill A"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillB, "SKILL.md"), []byte("# Skill B"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillC, "SKILL.md"), []byte("# Skill C"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "other", "not-a-skill", "random.md"), []byte("not a skill"), 0644))

	discovery := NewSkillDiscovery(tmpDir)

	t.Run("find skill in subdirectory", func(t *testing.T) {
		skill, err := discovery.FindSkillByName("skill-a")
		require.NoError(t, err)
		assert.Equal(t, "skill-a", skill.Name)
		assert.Equal(t, filepath.Join("skills", "skill-a"), skill.Path)
		assert.Equal(t, skillA, skill.FullPath)
	})

	t.Run("find another skill", func(t *testing.T) {
		skill, err := discovery.FindSkillByName("skill-b")
		require.NoError(t, err)
		assert.Equal(t, "skill-b", skill.Name)
	})

	t.Run("find skill in different location", func(t *testing.T) {
		skill, err := discovery.FindSkillByName("skill-c")
		require.NoError(t, err)
		assert.Equal(t, "skill-c", skill.Name)
		assert.Equal(t, filepath.Join("other", "skill-c"), skill.Path)
	})

	t.Run("skill not found", func(t *testing.T) {
		_, err := discovery.FindSkillByName("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("case insensitive match", func(t *testing.T) {
		skill, err := discovery.FindSkillByName("SKILL-A")
		require.NoError(t, err)
		assert.Equal(t, "skill-a", skill.Name) // Returns actual name from directory
	})
}

func TestSkillDiscovery_FindAllSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple skills
	skills := []string{
		filepath.Join(tmpDir, "skill-1", "SKILL.md"),
		filepath.Join(tmpDir, "nested", "skill-2", "SKILL.md"),
		filepath.Join(tmpDir, "deeply", "nested", "path", "skill-3", "SKILL.md"),
	}

	for _, path := range skills {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte("# Skill"), 0644))
	}

	discovery := NewSkillDiscovery(tmpDir)
	found, err := discovery.FindAllSkills()
	require.NoError(t, err)
	assert.Len(t, found, 3)

	// Check that all skills were found
	names := make([]string, len(found))
	for i, s := range found {
		names[i] = s.Name
	}
	assert.Contains(t, names, "skill-1")
	assert.Contains(t, names, "skill-2")
	assert.Contains(t, names, "skill-3")
}

func TestSkillDiscovery_EmptyRepo(t *testing.T) {
	tmpDir := t.TempDir()

	discovery := NewSkillDiscovery(tmpDir)

	_, err := discovery.FindSkillByName("any-skill")
	assert.Error(t, err)

	skills, err := discovery.FindAllSkills()
	require.NoError(t, err)
	assert.Empty(t, skills)
}
