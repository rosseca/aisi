package installer

import (
	"path/filepath"

	"github.com/rosseca/aisi/internal/manifest"
)

func (i *Installer) InstallAgentsMD(agentsMD *manifest.AgentsMD) (*InstallResult, error) {
	if !i.target.SupportsAgentsMD {
		return &InstallResult{
			Name:    agentsMD.Name,
			Type:    manifest.AssetTypeAgentsMD,
			Success: true,
		}, nil
	}

	destPath := filepath.Join(i.projectRoot, "AGENTS.md")

	srcPath := i.repoMgr.GetFilePath(agentsMD.Path)

	if err := i.fs.CopyFile(srcPath, destPath); err != nil {
		return &InstallResult{
			Name:    agentsMD.Name,
			Type:    manifest.AssetTypeAgentsMD,
			Path:    destPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    agentsMD.Name,
		Type:    manifest.AssetTypeAgentsMD,
		Path:    destPath,
		Success: true,
	}, nil
}
