package manifest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "manifest.json")
	m, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", m.Version)
	assert.Len(t, m.Rules, 1)
	assert.Len(t, m.Skills, 1)
	assert.Len(t, m.Agents, 1)
	assert.Len(t, m.Hooks, 1)
	assert.Len(t, m.MCP, 1)
	assert.Len(t, m.AgentsMD, 1)
	assert.Len(t, m.External, 1)
}

func TestParse(t *testing.T) {
	data := []byte(`{
		"version": "1.0.0",
		"rules": [
			{"name": "test-rule", "path": "rules/test.mdc", "description": "Test rule"}
		],
		"skills": [
			{"name": "test-skill", "path": "skills/test", "description": "Test skill"}
		],
		"external": [
			{
				"name": "ext-skill",
				"type": "skill",
				"repo": "github.com/example/repo",
				"path": "my-skill",
				"ref": "v1.0.0",
				"description": "External skill"
			}
		]
	}`)

	m, err := Parse(data)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", m.Version)
	assert.Len(t, m.Rules, 1)
	assert.Equal(t, "test-rule", m.Rules[0].Name)
	assert.Len(t, m.Skills, 1)
	assert.Equal(t, "test-skill", m.Skills[0].Name)
	assert.Len(t, m.External, 1)
	assert.Equal(t, "ext-skill", m.External[0].Name)
	assert.Equal(t, "v1.0.0", m.External[0].Ref)
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse([]byte("not json"))
	assert.Error(t, err)
}

func TestManifest_GetRule(t *testing.T) {
	m := &Manifest{
		Rules: []Rule{
			{Name: "rule1", Path: "rules/rule1.mdc"},
			{Name: "rule2", Path: "rules/rule2.mdc"},
		},
	}

	r := m.GetRule("rule1")
	require.NotNil(t, r)
	assert.Equal(t, "rules/rule1.mdc", r.Path)

	r = m.GetRule("nonexistent")
	assert.Nil(t, r)
}

func TestManifest_GetSkill(t *testing.T) {
	m := &Manifest{
		Skills: []Skill{
			{Name: "skill1", Path: "skills/skill1"},
			{Name: "skill2", Path: "skills/skill2"},
		},
	}

	s := m.GetSkill("skill2")
	require.NotNil(t, s)
	assert.Equal(t, "skills/skill2", s.Path)

	s = m.GetSkill("nonexistent")
	assert.Nil(t, s)
}

func TestManifest_GetAgent(t *testing.T) {
	m := &Manifest{
		Agents: []Agent{
			{Name: "agent1", Path: "agents/agent1.md"},
		},
	}

	a := m.GetAgent("agent1")
	require.NotNil(t, a)

	a = m.GetAgent("nonexistent")
	assert.Nil(t, a)
}

func TestManifest_GetMCP(t *testing.T) {
	m := &Manifest{
		MCP: []MCP{
			{
				Name:        "mcp1",
				Path:        "mcp/mcp1.json",
				Description: "Test MCP",
				Env: map[string]EnvVarMeta{
					"API_KEY": {Description: "API Key", Required: true, Secret: true},
				},
			},
		},
	}

	mc := m.GetMCP("mcp1")
	require.NotNil(t, mc)
	assert.True(t, mc.Env["API_KEY"].Secret)

	mc = m.GetMCP("nonexistent")
	assert.Nil(t, mc)
}

func TestManifest_GetExternal(t *testing.T) {
	m := &Manifest{
		External: []External{
			{Name: "ext1", Type: "skill", Repo: "github.com/example/repo"},
		},
	}

	e := m.GetExternal("ext1")
	require.NotNil(t, e)
	assert.Equal(t, "skill", e.Type)

	e = m.GetExternal("nonexistent")
	assert.Nil(t, e)
}

func TestManifest_FindAsset(t *testing.T) {
	m := &Manifest{
		Rules:  []Rule{{Name: "rule1", Path: "rules/rule1.mdc"}},
		Skills: []Skill{{Name: "skill1", Path: "skills/skill1"}},
		External: []External{
			{Name: "ext1", Type: "skill", Repo: "github.com/example/repo"},
		},
	}

	assetType, asset := m.FindAsset("rule1")
	assert.Equal(t, AssetTypeRule, assetType)
	assert.NotNil(t, asset)

	assetType, asset = m.FindAsset("skill1")
	assert.Equal(t, AssetTypeSkill, assetType)
	assert.NotNil(t, asset)

	assetType, asset = m.FindAsset("ext1")
	assert.Equal(t, AssetType("external:skill"), assetType)
	assert.NotNil(t, asset)

	assetType, asset = m.FindAsset("nonexistent")
	assert.Equal(t, AssetType(""), assetType)
	assert.Nil(t, asset)
}

func TestManifest_ListNames(t *testing.T) {
	m := &Manifest{
		Rules:    []Rule{{Name: "r1"}, {Name: "r2"}},
		Skills:   []Skill{{Name: "s1"}},
		Agents:   []Agent{{Name: "a1"}, {Name: "a2"}, {Name: "a3"}},
		External: []External{{Name: "e1"}},
	}

	assert.Equal(t, []string{"r1", "r2"}, m.ListRuleNames())
	assert.Equal(t, []string{"s1"}, m.ListSkillNames())
	assert.Equal(t, []string{"a1", "a2", "a3"}, m.ListAgentNames())
	assert.Equal(t, []string{"e1"}, m.ListExternalNames())
}

func TestManifest_GetExternalByType(t *testing.T) {
	m := &Manifest{
		External: []External{
			{Name: "ext-skill", Type: "skill"},
			{Name: "ext-agent", Type: "agent"},
			{Name: "ext-skill2", Type: "skill"},
		},
	}

	skills := m.GetExternalByType("skill")
	assert.Len(t, skills, 2)

	agents := m.GetExternalByType("agent")
	assert.Len(t, agents, 1)

	rules := m.GetExternalByType("rule")
	assert.Len(t, rules, 0)
}

func TestManifest_AllSkills(t *testing.T) {
	m := &Manifest{
		Skills: []Skill{{Name: "s1"}, {Name: "s2"}},
	}

	skills := m.AllSkills()
	assert.Len(t, skills, 2)

	skills[0].Name = "modified"
	assert.Equal(t, "s1", m.Skills[0].Name)
}

// Tests for parseVersion
func TestParseVersion(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		wantMajor    int
		wantMinor    int
		wantPatch    int
		wantErr      bool
		errContains  string
	}{
		{"valid full version", "1.2.3", 1, 2, 3, false, ""},
		{"valid with v prefix", "v1.2.3", 1, 2, 3, false, ""},
		{"valid no patch", "1.2", 1, 2, 0, false, ""},
		{"valid with v no patch", "v2.0", 2, 0, 0, false, ""},
		{"invalid format", "1", 0, 0, 0, true, "invalid version format"},
		{"invalid major", "abc.1.2", 0, 0, 0, true, "invalid major version"},
		{"invalid minor", "1.abc.2", 0, 0, 0, true, "invalid minor version"},
		{"invalid patch", "1.2.abc", 0, 0, 0, true, "invalid patch version"},
		{"empty string", "", 0, 0, 0, true, "invalid version format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch, err := parseVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMajor, major)
				assert.Equal(t, tt.wantMinor, minor)
				assert.Equal(t, tt.wantPatch, patch)
			}
		})
	}
}

// Tests for CheckCLIVersion
func TestManifest_CheckCLIVersion(t *testing.T) {
	tests := []struct {
		name           string
		minimumVersion string
		currentVersion string
		wantErr        bool
		wantMismatch   bool
	}{
		{"no minimum version", "", "1.0.0", false, false},
		{"dev version passes", "1.0.0", "dev", false, false},
		{"empty current passes", "1.0.0", "", false, false},
		{"exact match", "1.0.0", "1.0.0", false, false},
		{"current greater major", "1.0.0", "2.0.0", false, false},
		{"current greater minor", "1.0.0", "1.1.0", false, false},
		{"current greater patch", "1.0.0", "1.0.1", false, false},
		{"current less major", "2.0.0", "1.0.0", true, true},
		{"current less minor", "1.2.0", "1.1.0", true, true},
		{"current less patch", "1.0.2", "1.0.1", true, true},
		{"with v prefix match", "v1.0.0", "1.0.0", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manifest{MinimumCLIVersion: tt.minimumVersion}
			err := m.CheckCLIVersion(tt.currentVersion)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantMismatch {
					var mismatchErr *VersionMismatchError
					assert.True(t, assert.ErrorAs(t, err, &mismatchErr))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Tests for VersionMismatchError
func TestVersionMismatchError(t *testing.T) {
	err := &VersionMismatchError{
		CurrentVersion: "1.0.0",
		RequiredVersion: "2.0.0",
	}

	assert.Contains(t, err.Error(), "1.0.0")
	assert.Contains(t, err.Error(), "2.0.0")
	assert.Contains(t, err.Error(), "below minimum required")
}

// Tests for CheckCLIVersion with invalid versions
func TestManifest_CheckCLIVersion_InvalidVersions(t *testing.T) {
	tests := []struct {
		name           string
		minimumVersion string
		currentVersion string
		errContains    string
	}{
		{"invalid current version", "1.0.0", "invalid", "failed to parse current CLI version"},
		{"invalid minimum version", "invalid", "1.0.0", "failed to parse minimum CLI version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manifest{MinimumCLIVersion: tt.minimumVersion}
			err := m.CheckCLIVersion(tt.currentVersion)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

// Tests for GetHook
func TestManifest_GetHook(t *testing.T) {
	m := &Manifest{
		Hooks: []Hook{
			{Name: "hook1", ConfigPath: "hooks/hook1.json", ScriptsPath: "hooks/scripts"},
			{Name: "hook2", ConfigPath: "hooks/hook2.json"},
		},
	}

	h := m.GetHook("hook1")
	require.NotNil(t, h)
	assert.Equal(t, "hooks/hook1.json", h.ConfigPath)
	assert.Equal(t, "hooks/scripts", h.ScriptsPath)

	h = m.GetHook("nonexistent")
	assert.Nil(t, h)
}

// Tests for GetAgentsMD
func TestManifest_GetAgentsMD(t *testing.T) {
	m := &Manifest{
		AgentsMD: []AgentsMD{
			{Name: "agents1", Path: "agents-md/agents1.md"},
			{Name: "agents2", Path: "agents-md/agents2.md"},
		},
	}

	a := m.GetAgentsMD("agents1")
	require.NotNil(t, a)
	assert.Equal(t, "agents-md/agents1.md", a.Path)

	a = m.GetAgentsMD("nonexistent")
	assert.Nil(t, a)
}

// Tests for ListHookNames
func TestManifest_ListHookNames(t *testing.T) {
	m := &Manifest{
		Hooks: []Hook{
			{Name: "hook1"},
			{Name: "hook2"},
		},
	}

	names := m.ListHookNames()
	assert.Equal(t, []string{"hook1", "hook2"}, names)
	assert.Empty(t, (&Manifest{}).ListHookNames())
}

// Tests for ListMCPNames
func TestManifest_ListMCPNames(t *testing.T) {
	m := &Manifest{
		MCP: []MCP{
			{Name: "mcp1"},
			{Name: "mcp2"},
			{Name: "mcp3"},
		},
	}

	names := m.ListMCPNames()
	assert.Equal(t, []string{"mcp1", "mcp2", "mcp3"}, names)
	assert.Empty(t, (&Manifest{}).ListMCPNames())
}

// Tests for ListAgentsMDNames
func TestManifest_ListAgentsMDNames(t *testing.T) {
	m := &Manifest{
		AgentsMD: []AgentsMD{
			{Name: "agents1"},
			{Name: "agents2"},
		},
	}

	names := m.ListAgentsMDNames()
	assert.Equal(t, []string{"agents1", "agents2"}, names)
	assert.Empty(t, (&Manifest{}).ListAgentsMDNames())
}

// Tests for AllAgents
func TestManifest_AllAgents(t *testing.T) {
	m := &Manifest{
		Agents: []Agent{
			{Name: "agent1", Path: "agents/agent1.md"},
			{Name: "agent2", Path: "agents/agent2.md"},
		},
	}

	agents := m.AllAgents()
	assert.Len(t, agents, 2)
	assert.Equal(t, "agent1", agents[0].Name)

	// Verify copy (modifying returned slice shouldn't affect original)
	agents[0].Name = "modified"
	assert.Equal(t, "agent1", m.Agents[0].Name)
}

// Tests for AllExternalSkills
func TestManifest_AllExternalSkills(t *testing.T) {
	m := &Manifest{
		External: []External{
			{Name: "ext-skill1", Type: "skill"},
			{Name: "ext-agent1", Type: "agent"},
			{Name: "ext-skill2", Type: "skill"},
		},
	}

	skills := m.AllExternalSkills()
	assert.Len(t, skills, 2)
	assert.Equal(t, "ext-skill1", skills[0].Name)
	assert.Equal(t, "ext-skill2", skills[1].Name)
}

// Tests for AllExternalAgents
func TestManifest_AllExternalAgents(t *testing.T) {
	m := &Manifest{
		External: []External{
			{Name: "ext-skill1", Type: "skill"},
			{Name: "ext-agent1", Type: "agent"},
			{Name: "ext-agent2", Type: "agent"},
		},
	}

	agents := m.AllExternalAgents()
	assert.Len(t, agents, 2)
	assert.Equal(t, "ext-agent1", agents[0].Name)
	assert.Equal(t, "ext-agent2", agents[1].Name)
}
