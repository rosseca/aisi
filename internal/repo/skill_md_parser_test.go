package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSkillMD(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("skill with frontmatter", func(t *testing.T) {
		skillContent := `---
name: react-best-practices
description: Best practices for React applications with performance optimizations
---

# React Best Practices

This skill provides guidelines for writing high-quality React code.

## When to Use

- Building new React components
- Refactoring existing code
`
		skillPath := filepath.Join(tmpDir, "react-skill", "SKILL.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0755))
		require.NoError(t, os.WriteFile(skillPath, []byte(skillContent), 0644))

		metadata, err := ParseSkillMD(skillPath)
		require.NoError(t, err)
		assert.Equal(t, "react-best-practices", metadata.Name)
		assert.Equal(t, "Best practices for React applications with performance optimizations", metadata.Description)
		assert.Contains(t, metadata.RawContent, "# React Best Practices")
	})

	t.Run("skill without frontmatter", func(t *testing.T) {
		skillContent := `# TypeScript Skill

Guidelines for TypeScript development.

## Rules

- Use strict mode
`
		skillPath := filepath.Join(tmpDir, "ts-skill", "SKILL.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0755))
		require.NoError(t, os.WriteFile(skillPath, []byte(skillContent), 0644))

		metadata, err := ParseSkillMD(skillPath)
		require.NoError(t, err)
		assert.Empty(t, metadata.Name)
		assert.Empty(t, metadata.Description)
		assert.Contains(t, metadata.RawContent, "# TypeScript Skill")
	})

	t.Run("empty skill file", func(t *testing.T) {
		skillPath := filepath.Join(tmpDir, "empty-skill", "SKILL.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(skillPath), 0755))
		require.NoError(t, os.WriteFile(skillPath, []byte(""), 0644))

		metadata, err := ParseSkillMD(skillPath)
		require.NoError(t, err)
		assert.Empty(t, metadata.Name)
		assert.Empty(t, metadata.Description)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := ParseSkillMD("/non/existent/SKILL.md")
		assert.Error(t, err)
	})
}

func TestSkillMetadata_GetDescription(t *testing.T) {
	t.Run("description from frontmatter", func(t *testing.T) {
		sm := &SkillMetadata{
			Description: "This is a detailed description of the skill",
		}
		desc := sm.GetDescription(50)
		assert.Equal(t, "This is a detailed description of the skill", desc)
	})

	t.Run("description truncated", func(t *testing.T) {
		sm := &SkillMetadata{
			Description: "This is a very long description that should be truncated",
		}
		desc := sm.GetDescription(20)
		assert.Equal(t, "This is a very lo...", desc)
	})

	t.Run("description from content", func(t *testing.T) {
		sm := &SkillMetadata{
			RawContent: "# Title\n\nFirst paragraph of the skill description.\n\nMore content here.",
		}
		desc := sm.GetDescription(50)
		assert.Equal(t, "First paragraph of the skill description.", desc)
	})

	t.Run("empty description and content", func(t *testing.T) {
		sm := &SkillMetadata{}
		desc := sm.GetDescription(50)
		assert.Empty(t, desc)
	})
}
