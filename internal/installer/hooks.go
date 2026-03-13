package installer

import (
	"fmt"

	"github.com/rosseca/aisi/internal/manifest"
)

func (i *Installer) InstallHook(hook *manifest.Hook) (*InstallResult, error) {
	if i.target.HooksFile == "" {
		return &InstallResult{
			Name:    hook.Name,
			Type:    manifest.AssetTypeHook,
			Success: false,
			Error:   fmt.Errorf("target %s does not support hooks", i.target.Name),
		}, nil
	}

	configDir := i.target.ConfigPath(i.projectRoot)
	if err := i.fs.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	srcConfigPath := i.repoMgr.GetFilePath(hook.ConfigPath)
	destConfigPath := i.target.HooksConfigPath(i.projectRoot)

	if err := i.fs.CopyFile(srcConfigPath, destConfigPath); err != nil {
		return &InstallResult{
			Name:    hook.Name,
			Type:    manifest.AssetTypeHook,
			Path:    destConfigPath,
			Success: false,
			Error:   err,
		}, nil
	}

	if hook.ScriptsPath != "" && i.target.HooksScriptsDir != "" {
		srcScriptsPath := i.repoMgr.GetFilePath(hook.ScriptsPath)
		destScriptsPath := i.target.HooksScriptsPath(i.projectRoot)

		if err := i.fs.CopyDir(srcScriptsPath, destScriptsPath); err != nil {
			return &InstallResult{
				Name:    hook.Name,
				Type:    manifest.AssetTypeHook,
				Path:    destConfigPath,
				Success: false,
				Error:   fmt.Errorf("failed to copy hook scripts: %w", err),
			}, nil
		}
	}

	return &InstallResult{
		Name:    hook.Name,
		Type:    manifest.AssetTypeHook,
		Path:    destConfigPath,
		Success: true,
	}, nil
}
