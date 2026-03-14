package installer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rosseca/aisi/internal/deps"
	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/repo"
)

func (i *Installer) InstallSkill(skill *manifest.Skill) (*InstallResult, error) {
	if i.target.SkillsDir == "" {
		return &InstallResult{
			Name:    skill.Name,
			Type:    manifest.AssetTypeSkill,
			Success: false,
			Error:   fmt.Errorf("target %s does not support skills", i.target.Name),
		}, nil
	}

	// Verificar dependencias del comando si están definidas
	if skill.Command != "" && skill.Install != nil {
		depsMgr := deps.NewManager()
		
		// Pass progress callback to deps manager
		if i.onProgress != nil {
			depsMgr.SetProgressCallback(i.onProgress)
		}

		if !depsMgr.CheckCommand(skill.Command) {
			// Comando no existe, intentar instalar
			msg := fmt.Sprintf("📦 Installing dependency '%s' for skill '%s'...", skill.Command, skill.Name)
			if i.onProgress != nil {
				i.onProgress(msg)
			}
			fmt.Println(msg) // Also print for non-TUI usage
			
			if err := depsMgr.Install(skill.Install); err != nil {
				return &InstallResult{
					Name:    skill.Name,
					Type:    manifest.AssetTypeSkill,
					Success: false,
					Error:   fmt.Errorf("failed to install dependency '%s': %w", skill.Command, err),
				}, nil
			}
			
			successMsg := fmt.Sprintf("✓ Dependency '%s' installed successfully", skill.Command)
			if i.onProgress != nil {
				i.onProgress(successMsg)
			}
			fmt.Println(successMsg) // Also print for non-TUI usage
		}
	}

	destDir := i.target.SkillsPath(i.projectRoot)
	if err := i.fs.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	srcPath := i.repoMgr.GetFilePath(skill.Path)
	destPath := filepath.Join(destDir, skill.Name)

	if i.fileExists(destPath) {
		if err := i.fs.Remove(destPath); err != nil {
			return nil, fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	if err := i.fs.CopyDir(srcPath, destPath); err != nil {
		return &InstallResult{
			Name:    skill.Name,
			Type:    manifest.AssetTypeSkill,
			Path:    destPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    skill.Name,
		Type:    manifest.AssetTypeSkill,
		Path:    destPath,
		Success: true,
	}, nil
}

func (i *Installer) InstallAllSkills(skills []manifest.Skill) ([]*InstallResult, error) {
	results := make([]*InstallResult, 0, len(skills))

	for _, skill := range skills {
		result, err := i.InstallSkill(&skill)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}

// InstallSkillFromURL installs a skill from a URL (git repo or local path)
func (i *Installer) InstallSkillFromURL(skillURL *repo.SkillURL, overrideName string) (*InstallResult, error) {
	if i.target.SkillsDir == "" {
		return &InstallResult{
			Name:    "",
			Type:    manifest.AssetTypeSkill,
			Success: false,
			Error:   fmt.Errorf("target %s does not support skills", i.target.Name),
		}, nil
	}

	// Determine skill name
	skillName := overrideName
	if skillName == "" {
		skillName = skillURL.GetSkillName()
	}

	destDir := i.target.SkillsPath(i.projectRoot)
	if err := i.fs.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	destPath := filepath.Join(destDir, skillName)

	var srcPath string
	var sourcePath string // Relative path within the repo for lock file

	if skillURL.IsLocal {
		// Local path: use directly
		srcPath = skillURL.Path
		sourcePath = skillURL.Path
	} else {
		// Git URL: need to clone/fetch the repo
		if i.repoMgr == nil {
			return &InstallResult{
				Name:    skillName,
				Type:    manifest.AssetTypeSkill,
				Success: false,
				Error:   fmt.Errorf("repository manager not available for git URL installation"),
			}, nil
		}

		// Clone or fetch the external repo
		repoPath, err := i.repoMgr.EnsureExternalRepo(skillURL.RepoURL, skillURL.Ref)
		if err != nil {
			return &InstallResult{
				Name:    skillName,
				Type:    manifest.AssetTypeSkill,
				Success: false,
				Error:   fmt.Errorf("failed to fetch repository: %w", err),
			}, nil
		}

		// Try the direct path first (for explicit paths like tree/blob URLs)
		directPath := filepath.Join(repoPath, skillURL.Path)
		if info, err := os.Stat(directPath); err == nil && info.IsDir() {
			// Direct path exists, use it
			srcPath = directPath
			sourcePath = skillURL.Path
		} else {
			// Direct path doesn't exist, try to discover the skill
			discovery := repo.NewSkillDiscovery(repoPath)
			discovered, err := discovery.FindSkillByName(skillName)
			if err != nil {
				// Check if it's a MultipleSkillsError - propagate it as-is so caller can show selection
				if _, ok := err.(*repo.MultipleSkillsError); ok {
					return nil, err
				}
				return &InstallResult{
					Name:    skillName,
					Type:    manifest.AssetTypeSkill,
					Success: false,
					Error:   fmt.Errorf("skill '%s' not found in repository: %w", skillName, err),
				}, nil
			}
			srcPath = discovered.FullPath
			sourcePath = discovered.Path // Use the discovered relative path
		}
	}

	// Verify source exists
	if _, err := os.Stat(srcPath); err != nil {
		return &InstallResult{
			Name:    skillName,
			Type:    manifest.AssetTypeSkill,
			Success: false,
			Error:   fmt.Errorf("skill source not found at %s: %w", srcPath, err),
		}, nil
	}

	// Remove existing skill if present
	if i.fileExists(destPath) {
		if err := i.fs.Remove(destPath); err != nil {
			return nil, fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	// Copy the skill
	if err := i.fs.CopyDir(srcPath, destPath); err != nil {
		return &InstallResult{
			Name:    skillName,
			Type:    manifest.AssetTypeSkill,
			Path:    destPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:       skillName,
		Type:       manifest.AssetTypeSkill,
		Path:       destPath,
		SourcePath: sourcePath,
		Success:    true,
	}, nil
}
