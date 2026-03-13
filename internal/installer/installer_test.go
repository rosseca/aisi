package installer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rosseca/aisi/internal/config"
	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/repo"
	"github.com/rosseca/aisi/internal/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockGitRunner struct{}

// MockFileSystem is a mock implementation of FileSystem for testing
type MockFileSystem struct {
	files   map[string][]byte
	dirs    map[string]bool
	removed map[string]bool
	errors  map[string]error
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:   make(map[string][]byte),
		dirs:    make(map[string]bool),
		removed: make(map[string]bool),
		errors:  make(map[string]error),
	}
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	if err, ok := m.errors["read:"+path]; ok {
		return nil, err
	}
	if data, ok := m.files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if err, ok := m.errors["write:"+path]; ok {
		return err
	}
	m.files[path] = data
	return nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err, ok := m.errors["mkdir:"+path]; ok {
		return err
	}
	m.dirs[path] = true
	return nil
}

func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	if err, ok := m.errors["stat:"+path]; ok {
		return nil, err
	}
	if _, ok := m.files[path]; ok {
		return &mockFileInfo{name: filepath.Base(path)}, nil
	}
	if m.dirs[path] {
		return &mockFileInfo{name: filepath.Base(path), isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) CopyFile(src, dst string) error {
	if err, ok := m.errors["copy:"+src]; ok {
		return err
	}
	data, err := m.ReadFile(src)
	if err != nil {
		return err
	}
	m.files[dst] = data
	return nil
}

func (m *MockFileSystem) CopyDir(src, dst string) error {
	if err, ok := m.errors["copydir:"+src]; ok {
		return err
	}
	// Simple mock: copy all files with src prefix
	for path, data := range m.files {
		if strings.HasPrefix(path, src) {
			rel, _ := filepath.Rel(src, path)
			m.files[filepath.Join(dst, rel)] = data
		}
	}
	return nil
}

func (m *MockFileSystem) Remove(path string) error {
	if err, ok := m.errors["remove:"+path]; ok {
		return err
	}
	delete(m.files, path)
	delete(m.dirs, path)
	m.removed[path] = true
	return nil
}

// Helper methods for mock setup
func (m *MockFileSystem) AddFile(path string, content []byte) {
	m.files[path] = content
}

func (m *MockFileSystem) AddDir(path string) {
	m.dirs[path] = true
}

func (m *MockFileSystem) SetError(op, path string, err error) {
	m.errors[op+":"+path] = err
}

type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func (m *MockGitRunner) Clone(url, dest string, depth int) error {
	return os.MkdirAll(filepath.Join(dest, ".git"), 0755)
}

func (m *MockGitRunner) Pull(repoPath string) error {
	return nil
}

func (m *MockGitRunner) GetRemoteURL(repoPath string) (string, error) {
	return "git@github.com:test/repo.git", nil
}

func (m *MockGitRunner) GetCurrentCommit(repoPath string) (string, error) {
	return "abc123", nil
}

func (m *MockGitRunner) Checkout(repoPath, ref string) error {
	return nil
}

func (m *MockGitRunner) VerifyRepoAccess(url string) error {
	return nil
}

func setupTestInstaller(t *testing.T) (*Installer, string, string) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})

	cacheDir := filepath.Join(tmpDir, ".aisi", "cache", "test-test-repo")
	rulesDir := filepath.Join(cacheDir, "rules")
	skillsDir := filepath.Join(cacheDir, "skills", "test-skill")
	agentsDir := filepath.Join(cacheDir, "agents")
	hooksDir := filepath.Join(cacheDir, "hooks")
	scriptsDir := filepath.Join(hooksDir, "scripts")
	mcpDir := filepath.Join(cacheDir, "mcp")
	agentsMdDir := filepath.Join(cacheDir, "agents-md")

	require.NoError(t, os.MkdirAll(rulesDir, 0755))
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.MkdirAll(agentsDir, 0755))
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))
	require.NoError(t, os.MkdirAll(mcpDir, 0755))
	require.NoError(t, os.MkdirAll(agentsMdDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(cacheDir, ".git"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "test.mdc"), []byte("# Test Rule"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# Test Skill"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte("# Test Agent"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "hooks.json"), []byte(`{"version": 1}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "test.sh"), []byte("#!/bin/bash"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(mcpDir, "test.json"), []byte(`{"command": "test"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(mcpDir, "http-mcp.json"), []byte(`{"url": "https://api.example.com/mcp", "headers": {"Authorization": "${env:API_KEY}", "X-Custom-Header": "${env:CUSTOM_VAR}"}}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(agentsMdDir, "test.md"), []byte("# Test AGENTS.md"), 0644))

	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	cfg := config.DefaultConfig()
	cfg.SetRepo("git@github.com:test/test-repo.git", "main")

	mgr, err := repo.NewManagerWithGit(cfg, &MockGitRunner{})
	require.NoError(t, err)

	installer := New(mgr, targets.CursorTarget, projectDir)

	return installer, projectDir, cacheDir
}

func TestInstaller_InstallRule(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	rule := &manifest.Rule{
		Name:        "test",
		Path:        "rules/test.mdc",
		Description: "Test rule",
	}

	result, err := installer.InstallRule(rule)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, manifest.AssetTypeRule, result.Type)

	installedPath := filepath.Join(projectDir, ".cursor", "rules", "test.mdc")
	assert.FileExists(t, installedPath)

	content, err := os.ReadFile(installedPath)
	require.NoError(t, err)
	assert.Equal(t, "# Test Rule", string(content))
}

func TestInstaller_InstallSkill(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	skill := &manifest.Skill{
		Name:        "test-skill",
		Path:        "skills/test-skill",
		Description: "Test skill",
	}

	result, err := installer.InstallSkill(skill)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	installedPath := filepath.Join(projectDir, ".cursor", "skills", "test-skill", "SKILL.md")
	assert.FileExists(t, installedPath)
}

func TestInstaller_InstallAgent(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	agent := &manifest.Agent{
		Name:        "test",
		Path:        "agents/test.md",
		Description: "Test agent",
	}

	result, err := installer.InstallAgent(agent)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	installedPath := filepath.Join(projectDir, ".cursor", "agents", "test.md")
	assert.FileExists(t, installedPath)
}

func TestInstaller_InstallHook(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	hook := &manifest.Hook{
		Name:        "test",
		ConfigPath:  "hooks/hooks.json",
		ScriptsPath: "hooks/scripts",
		Description: "Test hook",
	}

	result, err := installer.InstallHook(hook)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	hooksConfigPath := filepath.Join(projectDir, ".cursor", "hooks.json")
	assert.FileExists(t, hooksConfigPath)

	scriptsPath := filepath.Join(projectDir, ".cursor", "hooks", "test.sh")
	assert.FileExists(t, scriptsPath)
}

func TestInstaller_InstallMCP(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	mcp := &manifest.MCP{
		Name:        "test",
		Path:        "mcp/test.json",
		Description: "Test MCP",
	}

	result, err := installer.InstallMCP(mcp, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	mcpPath := filepath.Join(projectDir, ".cursor", "mcp.json")
	assert.FileExists(t, mcpPath)

	data, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var config MCPConfig
	require.NoError(t, json.Unmarshal(data, &config))
	assert.Contains(t, config.MCPServers, "test")
}

func TestInstaller_InstallMCP_Merge(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	mcpPath := filepath.Join(projectDir, ".cursor", "mcp.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(mcpPath), 0755))
	require.NoError(t, os.WriteFile(mcpPath, []byte(`{"mcpServers":{"existing":{"command":"existing"}}}`), 0644))

	mcp := &manifest.MCP{
		Name:        "test",
		Path:        "mcp/test.json",
		Description: "Test MCP",
	}

	result, err := installer.InstallMCP(mcp, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)

	data, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var config MCPConfig
	require.NoError(t, json.Unmarshal(data, &config))
	assert.Contains(t, config.MCPServers, "test")
	assert.Contains(t, config.MCPServers, "existing")
}

func TestInstaller_InstallAgentsMD(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	agentsMD := &manifest.AgentsMD{
		Name:        "test",
		Path:        "agents-md/test.md",
		Description: "Test AGENTS.md",
	}

	result, err := installer.InstallAgentsMD(agentsMD)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	installedPath := filepath.Join(projectDir, "AGENTS.md")
	assert.FileExists(t, installedPath)
}

func TestInstaller_TargetNotSupported(t *testing.T) {
	installer, _, _ := setupTestInstaller(t)
	installer.target = targets.JunieTarget

	rule := &manifest.Rule{
		Name: "test",
		Path: "rules/test.mdc",
	}

	result, err := installer.InstallRule(rule)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error.Error(), "does not support rules")
}

func TestInstaller_Install(t *testing.T) {
	installer, _, _ := setupTestInstaller(t)

	m := &manifest.Manifest{
		Rules: []manifest.Rule{
			{Name: "test-rule", Path: "rules/test.mdc"},
		},
		Skills: []manifest.Skill{
			{Name: "test-skill", Path: "skills/test-skill"},
		},
	}

	result, err := installer.Install(m, "test-rule")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, manifest.AssetTypeRule, result.Type)

	result, err = installer.Install(m, "test-skill")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, manifest.AssetTypeSkill, result.Type)

	_, err = installer.Install(m, "nonexistent")
	assert.Error(t, err)
}

func TestInstaller_InstallMCP_MissingCommand(t *testing.T) {
	installer, projectDir, cacheDir := setupTestInstaller(t)

	// Crear un MCP que requiere un comando que no existe
	nonexistentMCP := filepath.Join(cacheDir, "mcp", "nonexistent-cmd.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(nonexistentMCP), 0755))
	require.NoError(t, os.WriteFile(nonexistentMCP, []byte(`{
		"command": "this-command-definitely-does-not-exist-12345",
		"args": ["test"]
	}`), 0644))

	mcp := &manifest.MCP{
		Name:        "nonexistent-cmd",
		Path:        "mcp/nonexistent-cmd.json",
		Description: "MCP with nonexistent command",
	}

	result, err := installer.InstallMCP(mcp, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error.Error(), "not found in PATH")

	// Verificar que no se creó el mcp.json
	mcpPath := filepath.Join(projectDir, ".cursor", "mcp.json")
	_, statErr := os.Stat(mcpPath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestInstaller_InstallMCP_HTTPWithEnvHeaders(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	mcp := &manifest.MCP{
		Name:        "http-mcp",
		Path:        "mcp/http-mcp.json",
		Description: "HTTP MCP with env headers",
		Env: map[string]manifest.EnvVarMeta{
			"API_KEY": {
				Description: "API key for authorization",
				Required:    true,
			},
			"CUSTOM_VAR": {
				Description: "Custom header value",
				Required:    false,
			},
		},
	}

	envVars := map[string]EnvVarConfig{
		"API_KEY": {
			VarName: "API_KEY",
			Value:   "my-secret-api-key",
			UseEnv:  false,
		},
		"CUSTOM_VAR": {
			VarName: "CUSTOM_VAR",
			Value:   "custom-value",
			UseEnv:  false,
		},
	}

	result, err := installer.InstallMCP(mcp, envVars)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	mcpPath := filepath.Join(projectDir, ".cursor", "mcp.json")
	assert.FileExists(t, mcpPath)

	data, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var config MCPConfig
	require.NoError(t, json.Unmarshal(data, &config))
	assert.Contains(t, config.MCPServers, "http-mcp")

	// Parse the server config to check headers
	var serverConfig map[string]interface{}
	require.NoError(t, json.Unmarshal(config.MCPServers["http-mcp"], &serverConfig))

	headers, ok := serverConfig["headers"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-secret-api-key", headers["Authorization"])
	assert.Equal(t, "custom-value", headers["X-Custom-Header"])
	assert.Equal(t, "https://api.example.com/mcp", serverConfig["url"])
}

func TestInstaller_InstallMCP_HTTPWithEnvRefHeaders(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	mcp := &manifest.MCP{
		Name:        "http-mcp",
		Path:        "mcp/http-mcp.json",
		Description: "HTTP MCP with env headers",
		Env: map[string]manifest.EnvVarMeta{
			"API_KEY": {
				Description: "API key for authorization",
				Required:    true,
			},
		},
	}

	envVars := map[string]EnvVarConfig{
		"API_KEY": {
			VarName: "API_KEY",
			Value:   "",
			UseEnv:  true, // Keep as ${env:API_KEY}
		},
	}

	result, err := installer.InstallMCP(mcp, envVars)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	mcpPath := filepath.Join(projectDir, ".cursor", "mcp.json")
	data, err := os.ReadFile(mcpPath)
	require.NoError(t, err)

	var config MCPConfig
	require.NoError(t, json.Unmarshal(data, &config))

	var serverConfig map[string]interface{}
	require.NoError(t, json.Unmarshal(config.MCPServers["http-mcp"], &serverConfig))

	headers, ok := serverConfig["headers"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "${env:API_KEY}", headers["Authorization"])
}

// ==================== UNINSTALL TESTS ====================

func TestInstaller_UninstallRule_Success(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddFile("/project/.cursor/rules/test.mdc", []byte("# Rule"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallRule("test")
	require.NoError(t, err)
	assert.True(t, mockFS.removed["/project/.cursor/rules/test.mdc"])
}

func TestInstaller_UninstallRule_TargetNotSupported(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.JunieTarget, // Junie doesn't support rules
		projectRoot: "/project",
	}

	err := installer.UninstallRule("test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support rules")
}

func TestInstaller_UninstallRule_NotFound(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallRule("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInstaller_UninstallSkill_Success(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddDir("/project/.cursor/skills/test-skill")

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallSkill("test-skill")
	require.NoError(t, err)
	assert.True(t, mockFS.removed["/project/.cursor/skills/test-skill"])
}

func TestInstaller_UninstallSkill_TargetNotSupported(t *testing.T) {
	mockFS := NewMockFileSystem()

	// Create a target without skills support
	noSkillsTarget := &targets.Target{
		Name:      "no-skills",
		SkillsDir: "", // Empty means no support
	}

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      noSkillsTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallSkill("test-skill")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support skills")
}

func TestInstaller_UninstallSkill_NotFound(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallSkill("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInstaller_UninstallAgent_Success(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddFile("/project/.cursor/agents/test.md", []byte("# Agent"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallAgent("test.md")
	require.NoError(t, err)
	assert.True(t, mockFS.removed["/project/.cursor/agents/test.md"])
}

func TestInstaller_UninstallAgent_TargetNotSupported(t *testing.T) {
	mockFS := NewMockFileSystem()

	// KiloTarget doesn't have agents support
	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.KiloTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallAgent("test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support agents")
}

func TestInstaller_UninstallAgent_NotFound(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallAgent("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInstaller_UninstallAgentsMD_Success(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddFile("/project/AGENTS.md", []byte("# AGENTS"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallAgentsMD()
	require.NoError(t, err)
	assert.True(t, mockFS.removed["/project/AGENTS.md"])
}

func TestInstaller_UninstallAgentsMD_NotSupported(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.JunieTarget, // Junie doesn't support AGENTS.md in project root
		projectRoot: "/project",
	}

	err := installer.UninstallAgentsMD()
	require.NoError(t, err) // Should return nil when not supported
}

func TestInstaller_UninstallAgentsMD_NotFound(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallAgentsMD()
	require.NoError(t, err) // Should return nil when file doesn't exist
}

func TestInstaller_UninstallMCP_Success(t *testing.T) {
	mockFS := NewMockFileSystem()
	mcpConfig := `{"mcpServers":{"test":{"command":"test"},"existing":{"command":"existing"}}}`
	mockFS.AddFile("/project/.cursor/mcp.json", []byte(mcpConfig))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallMCP("test")
	require.NoError(t, err)

	// Verify the MCP was removed
	data, _ := mockFS.ReadFile("/project/.cursor/mcp.json")
	var config MCPConfig
	_ = json.Unmarshal(data, &config)
	assert.Nil(t, config.MCPServers["test"])
	assert.NotNil(t, config.MCPServers["existing"])
}

func TestInstaller_UninstallMCP_TargetNotSupported(t *testing.T) {
	mockFS := NewMockFileSystem()

	// Create a target without MCP support
	noMCPTarget := &targets.Target{
		Name:    "no-mcp",
		MCPFile: "", // Empty means no support
	}

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      noMCPTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallMCP("test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support MCP")
}

func TestInstaller_UninstallMCP_NotFound(t *testing.T) {
	mockFS := NewMockFileSystem()
	mcpConfig := `{"mcpServers":{"existing":{"command":"existing"}}}`
	mockFS.AddFile("/project/.cursor/mcp.json", []byte(mcpConfig))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallMCP("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInstaller_UninstallMCP_NoConfigFile(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallMCP("test")
	require.Error(t, err)
}

func TestInstaller_UninstallHook_NotImplemented(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallHook("test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

// ==================== UNINSTALL GENERIC TESTS ====================

func TestInstaller_Uninstall_Generic(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddFile("/project/.cursor/rules/test.mdc", []byte("# Rule"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetTypeRule, "test")
	require.NoError(t, err)
}

func TestInstaller_Uninstall_Skill(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddDir("/project/.cursor/skills/test-skill")

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetTypeSkill, "test-skill")
	require.NoError(t, err)
}

func TestInstaller_Uninstall_Agent(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddFile("/project/.cursor/agents/test.md", []byte("# Agent"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetTypeAgent, "test.md")
	require.NoError(t, err)
}

func TestInstaller_Uninstall_MCP(t *testing.T) {
	mockFS := NewMockFileSystem()
	mcpConfig := `{"mcpServers":{"test":{"command":"test"}}}`
	mockFS.AddFile("/project/.cursor/mcp.json", []byte(mcpConfig))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetTypeMCP, "test")
	require.NoError(t, err)
}

func TestInstaller_Uninstall_Hook(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetTypeHook, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestInstaller_Uninstall_AgentsMD(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddFile("/project/AGENTS.md", []byte("# AGENTS"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetTypeAgentsMD, "")
	require.NoError(t, err)
}

func TestInstaller_Uninstall_ExternalSkill(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddDir("/project/.cursor/skills/ext-skill")

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetType("external:skill"), "ext-skill")
	require.NoError(t, err)
}

func TestInstaller_Uninstall_ExternalAgent(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddFile("/project/.cursor/agents/ext-agent.md", []byte("# Agent"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetType("external:agent"), "ext-agent.md")
	require.NoError(t, err)
}

func TestInstaller_Uninstall_ExternalRule(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.AddFile("/project/.cursor/rules/ext-rule.mdc", []byte("# Rule"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetType("external:rule"), "ext-rule.mdc")
	require.NoError(t, err)
}

func TestInstaller_Uninstall_UnsupportedExternalType(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.UninstallExternal("test", "hook")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported external type")
}

func TestInstaller_Uninstall_UnsupportedType(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.Uninstall(manifest.AssetType("unknown"), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported asset type")
}

// ==================== NEW WITH FS TESTS ====================

func TestNewWithFS(t *testing.T) {
	mockFS := NewMockFileSystem()
	repoMgr := &repo.Manager{}

	installer := NewWithFS(repoMgr, targets.CursorTarget, "/project", mockFS)

	assert.NotNil(t, installer)
	assert.Equal(t, "/project", installer.projectRoot)
	assert.Equal(t, targets.CursorTarget, installer.target)
	assert.Equal(t, repoMgr, installer.repoMgr)
}

// ==================== ENSURE CONFIG DIR TESTS ====================

func TestInstaller_EnsureConfigDir(t *testing.T) {
	mockFS := NewMockFileSystem()

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.EnsureConfigDir()
	require.NoError(t, err)
	assert.True(t, mockFS.dirs["/project/.cursor"])
}

func TestInstaller_EnsureConfigDir_Error(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.SetError("mkdir", "/project/.cursor", errors.New("permission denied"))

	installer := &Installer{
		fs:          mockFS,
		repoMgr:     nil,
		target:      targets.CursorTarget,
		projectRoot: "/project",
	}

	err := installer.EnsureConfigDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

// ==================== INSTALL ALL TESTS ====================

func TestInstaller_InstallAllRules(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	rules := []manifest.Rule{
		{Name: "rule1", Path: "rules/test.mdc"},
		{Name: "rule2", Path: "rules/test.mdc"},
	}

	results, err := installer.InstallAllRules(rules)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)
	assert.True(t, results[1].Success)

	// Verify both rules were installed
	assert.FileExists(t, filepath.Join(projectDir, ".cursor", "rules", "test.mdc"))
}


func TestInstaller_InstallAllSkills(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	skills := []manifest.Skill{
		{Name: "skill1", Path: "skills/test-skill"},
		{Name: "skill2", Path: "skills/test-skill"},
	}

	results, err := installer.InstallAllSkills(skills)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)

	// Verify skills were installed
	assert.FileExists(t, filepath.Join(projectDir, ".cursor", "skills", "skill1", "SKILL.md"))
	assert.FileExists(t, filepath.Join(projectDir, ".cursor", "skills", "skill2", "SKILL.md"))
}


func TestInstaller_InstallAllAgents(t *testing.T) {
	installer, projectDir, _ := setupTestInstaller(t)

	agents := []manifest.Agent{
		{Name: "agent1", Path: "agents/test.md"},
		{Name: "agent2", Path: "agents/test.md"},
	}

	results, err := installer.InstallAllAgents(agents)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)

	// Verify agents were installed
	assert.FileExists(t, filepath.Join(projectDir, ".cursor", "agents", "test.md"))
}


// MockGitRunnerFailing simulates git failures
type MockGitRunnerFailing struct{}

func (m *MockGitRunnerFailing) Clone(url, dest string, depth int) error {
	return errors.New("clone failed")
}

func (m *MockGitRunnerFailing) Pull(repoPath string) error {
	return errors.New("pull failed")
}

func (m *MockGitRunnerFailing) GetRemoteURL(repoPath string) (string, error) {
	return "", errors.New("get remote url failed")
}

func (m *MockGitRunnerFailing) GetCurrentCommit(repoPath string) (string, error) {
	return "", errors.New("get commit failed")
}

func (m *MockGitRunnerFailing) Checkout(repoPath, ref string) error {
	return errors.New("checkout failed")
}

func (m *MockGitRunnerFailing) VerifyRepoAccess(url string) error {
	return errors.New("verify access failed")
}

// ==================== INSTALL EXTERNAL TESTS ====================

func TestInstaller_InstallExternal_FetchError(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})

	cfg := config.DefaultConfig()
	cfg.SetRepo("git@github.com:test/test-repo.git", "main")

	// Use a mock git runner that will fail
	failingGitRunner := &MockGitRunnerFailing{}

	mgr, err := repo.NewManagerWithGit(cfg, failingGitRunner)
	require.NoError(t, err)

	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	installer := New(mgr, targets.CursorTarget, projectDir)

	ext := &manifest.External{
		Name:        "ext-skill",
		Type:        "skill",
		Repo:        "github.com/nonexistent/repo",
		Path:        "skill",
		Description: "External skill",
	}

	result, err := installer.InstallExternal(ext)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error.Error(), "failed to fetch external repo")
}

func TestInstaller_InstallExternal_UnsupportedType(t *testing.T) {
	installer, _, _ := setupTestInstaller(t)

	ext := &manifest.External{
		Name:        "ext-hook",
		Type:        "hook",
		Repo:        "github.com/example/ext-repo",
		Path:        "hooks",
		Description: "External hook (unsupported)",
	}

	result, err := installer.InstallExternal(ext)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error.Error(), "unsupported external type")
}

func TestInstaller_InstallExternal_TargetNotSupported(t *testing.T) {
	installer, _, _ := setupTestInstaller(t)

	// Create a target without skills support
	noSkillsTarget := &targets.Target{
		Name:      "no-skills",
		SkillsDir: "", // Empty means no support
	}
	installer.target = noSkillsTarget

	ext := &manifest.External{
		Name:        "ext-skill",
		Type:        "skill",
		Repo:        "github.com/example/ext-repo",
		Path:        "my-skill",
		Description: "External skill",
	}

	result, err := installer.InstallExternal(ext)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error.Error(), "does not support skills")
}
