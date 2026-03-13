package installer

import (
	"fmt"
	"path/filepath"

	"github.com/rosseca/aisi/internal/manifest"
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
