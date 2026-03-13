package targets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name        string
		targetName  string
		expectError bool
	}{
		{"cursor exists", "cursor", false},
		{"kilo exists", "kilo", false},
		{"junie exists", "junie", false},
		{"windsurf exists", "windsurf", false},
		{"unknown target", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := registry.Get(tt.targetName)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, target)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, target)
				assert.Equal(t, tt.targetName, target.Name)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()
	targets := registry.List()

	assert.GreaterOrEqual(t, len(targets), 4)
}

func TestRegistry_Names(t *testing.T) {
	registry := NewRegistry()
	names := registry.Names()

	assert.Contains(t, names, "cursor")
	assert.Contains(t, names, "kilo")
	assert.Contains(t, names, "junie")
	assert.Contains(t, names, "windsurf")
}

func TestTarget_Paths(t *testing.T) {
	target := CursorTarget
	projectRoot := "/home/user/project"

	assert.Equal(t, "/home/user/project/.cursor/rules", target.RulesPath(projectRoot))
	assert.Equal(t, "/home/user/project/.cursor/skills", target.SkillsPath(projectRoot))
	assert.Equal(t, "/home/user/project/.cursor/agents", target.AgentsPath(projectRoot))
	assert.Equal(t, "/home/user/project/.cursor/hooks.json", target.HooksConfigPath(projectRoot))
	assert.Equal(t, "/home/user/project/.cursor/hooks", target.HooksScriptsPath(projectRoot))
	assert.Equal(t, "/home/user/project/.cursor/mcp.json", target.MCPPath(projectRoot))
	assert.Equal(t, "/home/user/project/.cursor", target.ConfigPath(projectRoot))
}

func TestTarget_EmptyPaths(t *testing.T) {
	target := JunieTarget
	projectRoot := "/home/user/project"

	assert.Equal(t, "", target.RulesPath(projectRoot))
	assert.Equal(t, "", target.HooksConfigPath(projectRoot))
	assert.Equal(t, "", target.HooksScriptsPath(projectRoot))
	assert.Equal(t, "/home/user/project/.junie/skills", target.SkillsPath(projectRoot))
	assert.Equal(t, "/home/user/project/.junie/agents", target.AgentsPath(projectRoot))
	assert.Equal(t, "/home/user/project/.junie/mcp/mcp.json", target.MCPPath(projectRoot))
}

func TestDefaultRegistry(t *testing.T) {
	target, err := Get("cursor")
	require.NoError(t, err)
	assert.Equal(t, "Cursor", target.DisplayName)

	targets := List()
	assert.GreaterOrEqual(t, len(targets), 4)

	names := Names()
	assert.Contains(t, names, "cursor")
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	customTarget := &Target{
		Name:        "custom",
		DisplayName: "Custom Target",
		ConfigDir:   ".custom",
		RulesDir:    "rules",
		SkillsDir:   "skills",
	}

	registry.Register(customTarget)

	target, err := registry.Get("custom")
	require.NoError(t, err)
	assert.Equal(t, "Custom Target", target.DisplayName)
}
