package installer

import (
	"fmt"
	"path/filepath"

	"github.com/rosseca/aisi/internal/manifest"
)

func (i *Installer) InstallRule(rule *manifest.Rule) (*InstallResult, error) {
	if i.target.RulesDir == "" {
		return &InstallResult{
			Name:    rule.Name,
			Type:    manifest.AssetTypeRule,
			Success: false,
			Error:   fmt.Errorf("target %s does not support rules", i.target.Name),
		}, nil
	}

	destDir := i.target.RulesPath(i.projectRoot)
	if err := i.fs.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rules directory: %w", err)
	}

	srcPath := i.repoMgr.GetFilePath(rule.Path)
	destPath := filepath.Join(destDir, filepath.Base(rule.Path))

	if err := i.fs.CopyFile(srcPath, destPath); err != nil {
		return &InstallResult{
			Name:    rule.Name,
			Type:    manifest.AssetTypeRule,
			Path:    destPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    rule.Name,
		Type:    manifest.AssetTypeRule,
		Path:    destPath,
		Success: true,
	}, nil
}

func (i *Installer) InstallAllRules(rules []manifest.Rule) ([]*InstallResult, error) {
	results := make([]*InstallResult, 0, len(rules))

	for _, rule := range rules {
		result, err := i.InstallRule(&rule)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}
