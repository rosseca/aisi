package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/repo"
	"github.com/rosseca/aisi/internal/targets"
)

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	CopyFile(src, dst string) error
	CopyDir(src, dst string) error
	Remove(path string) error
}

type DefaultFS struct{}

func (d *DefaultFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (d *DefaultFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (d *DefaultFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (d *DefaultFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (d *DefaultFS) Remove(path string) error {
	return os.RemoveAll(path)
}

func (d *DefaultFS) CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

func (d *DefaultFS) CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return d.CopyFile(path, dstPath)
	})
}

type Installer struct {
	fs          FileSystem
	repoMgr     *repo.Manager
	target      *targets.Target
	projectRoot string
}

type InstallResult struct {
	Name       string
	Type       manifest.AssetType
	Path       string // Destination path where the skill was installed
	SourcePath string // Relative path within the source repo (for lock file)
	Success    bool
	Error      error
}

func New(repoMgr *repo.Manager, target *targets.Target, projectRoot string) *Installer {
	return &Installer{
		fs:          &DefaultFS{},
		repoMgr:     repoMgr,
		target:      target,
		projectRoot: projectRoot,
	}
}

func NewWithFS(repoMgr *repo.Manager, target *targets.Target, projectRoot string, fs FileSystem) *Installer {
	return &Installer{
		fs:          fs,
		repoMgr:     repoMgr,
		target:      target,
		projectRoot: projectRoot,
	}
}

func (i *Installer) EnsureConfigDir() error {
	configDir := i.target.ConfigPath(i.projectRoot)
	return i.fs.MkdirAll(configDir, 0755)
}

func (i *Installer) Install(m *manifest.Manifest, assetName string) (*InstallResult, error) {
	assetType, asset := m.FindAsset(assetName)
	if asset == nil {
		return nil, fmt.Errorf("asset not found: %s", assetName)
	}

	var result *InstallResult
	var err error

	switch assetType {
	case manifest.AssetTypeRule:
		result, err = i.InstallRule(asset.(*manifest.Rule))
	case manifest.AssetTypeSkill:
		result, err = i.InstallSkill(asset.(*manifest.Skill))
	case manifest.AssetTypeAgent:
		result, err = i.InstallAgent(asset.(*manifest.Agent))
	case manifest.AssetTypeHook:
		result, err = i.InstallHook(asset.(*manifest.Hook))
	case manifest.AssetTypeMCP:
		result, err = i.InstallMCP(asset.(*manifest.MCP), nil)
	case manifest.AssetTypeAgentsMD:
		result, err = i.InstallAgentsMD(asset.(*manifest.AgentsMD))
	default:
		// Handle external assets (type starts with "external:")
		if ext, ok := asset.(*manifest.External); ok {
			result, err = i.InstallExternal(ext)
		} else {
			return nil, fmt.Errorf("unsupported asset type: %s", assetType)
		}
	}

	return result, err
}

func (i *Installer) fileExists(path string) bool {
	_, err := i.fs.Stat(path)
	return err == nil
}

// Uninstall methods remove assets from the target directory

func (i *Installer) UninstallRule(name string) error {
	if i.target.RulesDir == "" {
		return fmt.Errorf("target %s does not support rules", i.target.Name)
	}

	rulesDir := i.target.RulesPath(i.projectRoot)
	// Find the rule file - could be name.mdc or variations
	patterns := []string{
		filepath.Join(rulesDir, name+".mdc"),
		filepath.Join(rulesDir, name),
	}

	for _, path := range patterns {
		if i.fileExists(path) {
			if err := i.fs.Remove(path); err != nil {
				return fmt.Errorf("failed to remove rule %s: %w", name, err)
			}
			return nil
		}
	}

	return fmt.Errorf("rule %s not found in %s", name, rulesDir)
}

func (i *Installer) UninstallSkill(name string) error {
	if i.target.SkillsDir == "" {
		return fmt.Errorf("target %s does not support skills", i.target.Name)
	}

	skillsDir := i.target.SkillsPath(i.projectRoot)
	skillPath := filepath.Join(skillsDir, name)

	if !i.fileExists(skillPath) {
		return fmt.Errorf("skill %s not found in %s", name, skillsDir)
	}

	if err := i.fs.Remove(skillPath); err != nil {
		return fmt.Errorf("failed to remove skill %s: %w", name, err)
	}

	return nil
}

func (i *Installer) UninstallAgent(name string) error {
	if i.target.AgentsDir == "" {
		return fmt.Errorf("target %s does not support agents", i.target.Name)
	}

	agentsDir := i.target.AgentsPath(i.projectRoot)
	// Find the agent file
	patterns := []string{
		filepath.Join(agentsDir, name),
		filepath.Join(agentsDir, name+".md"),
	}

	for _, path := range patterns {
		if i.fileExists(path) {
			if err := i.fs.Remove(path); err != nil {
				return fmt.Errorf("failed to remove agent %s: %w", name, err)
			}
			return nil
		}
	}

	return fmt.Errorf("agent %s not found in %s", name, agentsDir)
}

func (i *Installer) UninstallAgentsMD() error {
	if !i.target.SupportsAgentsMD {
		return nil // Nothing to do if target doesn't support it
	}

	agentsMDPath := filepath.Join(i.projectRoot, "AGENTS.md")
	if !i.fileExists(agentsMDPath) {
		// Also check in config dir for Junie
		agentsMDPath = filepath.Join(i.target.ConfigPath(i.projectRoot), "AGENTS.md")
		if !i.fileExists(agentsMDPath) {
			return nil // File doesn't exist, nothing to uninstall
		}
	}

	if err := i.fs.Remove(agentsMDPath); err != nil {
		return fmt.Errorf("failed to remove AGENTS.md: %w", err)
	}

	return nil
}

func (i *Installer) UninstallMCP(name string) error {
	if i.target.MCPFile == "" {
		return fmt.Errorf("target %s does not support MCP", i.target.Name)
	}

	mcpPath := i.target.MCPPath(i.projectRoot)

	// Load existing config
	existingConfig, err := i.loadExistingMCPConfig(mcpPath)
	if err != nil {
		return err
	}

	if existingConfig == nil || existingConfig.MCPServers[name] == nil {
		return fmt.Errorf("MCP %s not found in config", name)
	}

	// Remove the server from config
	delete(existingConfig.MCPServers, name)

	// Save back
	if err := i.saveMCPConfig(mcpPath, existingConfig); err != nil {
		return fmt.Errorf("failed to update MCP config: %w", err)
	}

	return nil
}

func (i *Installer) UninstallHook(name string) error {
	// For hooks, we just note it - actual hook removal is complex
	// as it involves modifying hooks.json
	return fmt.Errorf("hook uninstallation not yet implemented")
}

func (i *Installer) UninstallExternal(name string, extType string) error {
	// External assets are installed like regular skills/agents
	switch extType {
	case "skill":
		return i.UninstallSkill(name)
	case "agent":
		return i.UninstallAgent(name)
	case "rule":
		return i.UninstallRule(name)
	default:
		return fmt.Errorf("unsupported external type: %s", extType)
	}
}

func (i *Installer) Uninstall(assetType manifest.AssetType, name string) error {
	switch assetType {
	case manifest.AssetTypeRule:
		return i.UninstallRule(name)
	case manifest.AssetTypeSkill:
		return i.UninstallSkill(name)
	case manifest.AssetTypeAgent:
		return i.UninstallAgent(name)
	case manifest.AssetTypeHook:
		return i.UninstallHook(name)
	case manifest.AssetTypeMCP:
		return i.UninstallMCP(name)
	case manifest.AssetTypeAgentsMD:
		return i.UninstallAgentsMD()
	default:
		// Handle external types
		if strings.HasPrefix(string(assetType), "external:") {
			extType := strings.TrimPrefix(string(assetType), "external:")
			return i.UninstallExternal(name, extType)
		}
		return fmt.Errorf("unsupported asset type: %s", assetType)
	}
}
