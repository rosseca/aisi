package installer

import (
	"fmt"
	"path/filepath"

	"github.com/rosseca/aisi/internal/deps"
	"github.com/rosseca/aisi/internal/manifest"
)

func (i *Installer) InstallExternal(ext *manifest.External) (*InstallResult, error) {
	// Verificar dependencias del comando si están definidas (para skills externas también)
	if ext.Type == "skill" && ext.Command != "" && ext.Install != nil {
		depsMgr := deps.NewManager()
		
		// Pass progress callback to deps manager
		if i.onProgress != nil {
			depsMgr.SetProgressCallback(i.onProgress)
		}

		if !depsMgr.CheckCommand(ext.Command) {
			// Comando no existe, intentar instalar
			msg := fmt.Sprintf("📦 Installing dependency '%s' for external skill '%s'...", ext.Command, ext.Name)
			if i.onProgress != nil {
				i.onProgress(msg)
			}
			fmt.Println(msg)

			if err := depsMgr.Install(ext.Install); err != nil {
				return &InstallResult{
					Name:    ext.Name,
					Type:    manifest.AssetTypeSkill,
					Success: false,
					Error:   fmt.Errorf("failed to install dependency '%s': %w", ext.Command, err),
				}, nil
			}

			successMsg := fmt.Sprintf("✓ Dependency '%s' installed successfully", ext.Command)
			if i.onProgress != nil {
				i.onProgress(successMsg)
			}
			fmt.Println(successMsg)
		}
	}

	repoPath, err := i.repoMgr.EnsureExternalRepo(ext.Repo, ext.Ref)
	if err != nil {
		return &InstallResult{
			Name:    ext.Name,
			Type:    manifest.AssetType(ext.Type),
			Success: false,
			Error:   fmt.Errorf("failed to fetch external repo: %w", err),
		}, nil
	}

	srcPath := filepath.Join(repoPath, ext.Path)

	switch ext.Type {
	case "skill":
		return i.installExternalSkill(ext.Name, srcPath)
	case "agent":
		return i.installExternalAgent(ext.Name, srcPath)
	case "rule":
		return i.installExternalRule(ext.Name, srcPath)
	default:
		return &InstallResult{
			Name:    ext.Name,
			Type:    manifest.AssetType(ext.Type),
			Success: false,
			Error:   fmt.Errorf("unsupported external type: %s", ext.Type),
		}, nil
	}
}

func (i *Installer) installExternalSkill(name, srcPath string) (*InstallResult, error) {
	if i.target.SkillsDir == "" {
		return &InstallResult{
			Name:    name,
			Type:    manifest.AssetTypeSkill,
			Success: false,
			Error:   fmt.Errorf("target %s does not support skills", i.target.Name),
		}, nil
	}

	destDir := i.target.SkillsPath(i.projectRoot)
	if err := i.fs.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	destPath := filepath.Join(destDir, name)

	if i.fileExists(destPath) {
		if err := i.fs.Remove(destPath); err != nil {
			return nil, fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	if err := i.fs.CopyDir(srcPath, destPath); err != nil {
		return &InstallResult{
			Name:    name,
			Type:    manifest.AssetTypeSkill,
			Path:    destPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    name,
		Type:    manifest.AssetTypeSkill,
		Path:    destPath,
		Success: true,
	}, nil
}

func (i *Installer) installExternalAgent(name, srcPath string) (*InstallResult, error) {
	if i.target.AgentsDir == "" {
		return &InstallResult{
			Name:    name,
			Type:    manifest.AssetTypeAgent,
			Success: false,
			Error:   fmt.Errorf("target %s does not support agents", i.target.Name),
		}, nil
	}

	destDir := i.target.AgentsPath(i.projectRoot)
	if err := i.fs.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agents directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(srcPath))

	if err := i.fs.CopyFile(srcPath, destPath); err != nil {
		return &InstallResult{
			Name:    name,
			Type:    manifest.AssetTypeAgent,
			Path:    destPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    name,
		Type:    manifest.AssetTypeAgent,
		Path:    destPath,
		Success: true,
	}, nil
}

func (i *Installer) installExternalRule(name, srcPath string) (*InstallResult, error) {
	if i.target.RulesDir == "" {
		return &InstallResult{
			Name:    name,
			Type:    manifest.AssetTypeRule,
			Success: false,
			Error:   fmt.Errorf("target %s does not support rules", i.target.Name),
		}, nil
	}

	destDir := i.target.RulesPath(i.projectRoot)
	if err := i.fs.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rules directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(srcPath))

	if err := i.fs.CopyFile(srcPath, destPath); err != nil {
		return &InstallResult{
			Name:    name,
			Type:    manifest.AssetTypeRule,
			Path:    destPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    name,
		Type:    manifest.AssetTypeRule,
		Path:    destPath,
		Success: true,
	}, nil
}
