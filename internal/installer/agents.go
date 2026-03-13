package installer

import (
	"fmt"
	"path/filepath"

	"github.com/rosseca/aisi/internal/manifest"
)

func (i *Installer) InstallAgent(agent *manifest.Agent) (*InstallResult, error) {
	if i.target.AgentsDir == "" {
		return &InstallResult{
			Name:    agent.Name,
			Type:    manifest.AssetTypeAgent,
			Success: false,
			Error:   fmt.Errorf("target %s does not support agents", i.target.Name),
		}, nil
	}

	destDir := i.target.AgentsPath(i.projectRoot)
	if err := i.fs.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agents directory: %w", err)
	}

	srcPath := i.repoMgr.GetFilePath(agent.Path)
	destPath := filepath.Join(destDir, filepath.Base(agent.Path))

	if err := i.fs.CopyFile(srcPath, destPath); err != nil {
		return &InstallResult{
			Name:    agent.Name,
			Type:    manifest.AssetTypeAgent,
			Path:    destPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    agent.Name,
		Type:    manifest.AssetTypeAgent,
		Path:    destPath,
		Success: true,
	}, nil
}

func (i *Installer) InstallAllAgents(agents []manifest.Agent) ([]*InstallResult, error) {
	results := make([]*InstallResult, 0, len(agents))

	for _, agent := range agents {
		result, err := i.InstallAgent(&agent)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}
