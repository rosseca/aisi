package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rosseca/aisi/internal/manifest"
)

type MCPConfig struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

type EnvVarConfig struct {
	VarName string
	Value   string
	UseEnv  bool
}

func (i *Installer) InstallMCP(mcp *manifest.MCP, envVars map[string]EnvVarConfig) (*InstallResult, error) {
	if i.target.MCPFile == "" {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Success: false,
			Error:   fmt.Errorf("target %s does not support MCP", i.target.Name),
		}, nil
	}

	// Verificar que los comandos requeridos estén disponibles
	if err := i.checkRequiredCommands(mcp); err != nil {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Success: false,
			Error:   err,
		}, nil
	}

	mcpPath := i.target.MCPPath(i.projectRoot)

	if err := i.fs.MkdirAll(filepath.Dir(mcpPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create MCP directory: %w", err)
	}

	newServerData, err := i.loadMCPServerConfig(mcp, envVars)
	if err != nil {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Path:    mcpPath,
			Success: false,
			Error:   err,
		}, nil
	}

	existingConfig, err := i.loadExistingMCPConfig(mcpPath)
	if err != nil {
		return nil, err
	}

	if existingConfig == nil {
		existingConfig = &MCPConfig{
			MCPServers: make(map[string]json.RawMessage),
		}
	}

	existingConfig.MCPServers[mcp.Name] = newServerData

	if err := i.saveMCPConfig(mcpPath, existingConfig); err != nil {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Path:    mcpPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    mcp.Name,
		Type:    manifest.AssetTypeMCP,
		Path:    mcpPath,
		Success: true,
	}, nil
}

func (i *Installer) InstallMCPGlobal(mcp *manifest.MCP, envVars map[string]EnvVarConfig) (*InstallResult, error) {
	if i.target.MCPFile == "" {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Success: false,
			Error:   fmt.Errorf("target %s does not support MCP", i.target.Name),
		}, nil
	}

	// Verificar que los comandos requeridos estén disponibles
	if err := i.checkRequiredCommands(mcp); err != nil {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Success: false,
			Error:   err,
		}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	mcpPath := filepath.Join(home, i.target.ConfigDir, i.target.MCPFile)

	if err := i.fs.MkdirAll(filepath.Dir(mcpPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create MCP directory: %w", err)
	}

	newServerData, err := i.loadMCPServerConfig(mcp, envVars)
	if err != nil {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Path:    mcpPath,
			Success: false,
			Error:   err,
		}, nil
	}

	existingConfig, err := i.loadExistingMCPConfig(mcpPath)
	if err != nil {
		return nil, err
	}

	if existingConfig == nil {
		existingConfig = &MCPConfig{
			MCPServers: make(map[string]json.RawMessage),
		}
	}

	existingConfig.MCPServers[mcp.Name] = newServerData

	if err := i.saveMCPConfig(mcpPath, existingConfig); err != nil {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Path:    mcpPath,
			Success: false,
			Error:   err,
		}, nil
	}

	return &InstallResult{
		Name:    mcp.Name,
		Type:    manifest.AssetTypeMCP,
		Path:    mcpPath,
		Success: true,
	}, nil
}

func (i *Installer) loadMCPServerConfig(mcp *manifest.MCP, envVars map[string]EnvVarConfig) (json.RawMessage, error) {
	srcPath := i.repoMgr.GetFilePath(mcp.Path)
	data, err := i.fs.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config: %w", err)
	}

	if len(envVars) == 0 {
		return data, nil
	}

	var serverConfig map[string]interface{}
	if err := json.Unmarshal(data, &serverConfig); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config: %w", err)
	}

	// Handle stdio type (command-based) with env
	if env, ok := serverConfig["env"].(map[string]interface{}); ok {
		for varName, cfg := range envVars {
			if cfg.UseEnv {
				env[varName] = fmt.Sprintf("${env:%s}", varName)
			} else {
				env[varName] = cfg.Value
			}
		}
	}

	// Handle HTTP type with headers - look for ${env:VAR_NAME} patterns and replace them
	if headers, ok := serverConfig["headers"].(map[string]interface{}); ok {
		for headerName, headerValue := range headers {
			if strValue, ok := headerValue.(string); ok {
				// Check if this value contains ${env:VAR_NAME} pattern
				for varName, cfg := range envVars {
					placeholder := fmt.Sprintf("${env:%s}", varName)
					if strValue == placeholder {
						// Exact match - replace with value or keep as env ref
						if cfg.UseEnv {
							headers[headerName] = placeholder
						} else {
							headers[headerName] = cfg.Value
						}
						break
					}
				}
			}
		}
	}

	modifiedData, err := json.Marshal(serverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	return modifiedData, nil
}

func (i *Installer) loadExistingMCPConfig(path string) (*MCPConfig, error) {
	data, err := i.fs.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read existing MCP config: %w", err)
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse existing MCP config: %w", err)
	}

	return &config, nil
}

func (i *Installer) saveMCPConfig(path string, config *MCPConfig) error {
	if i.fileExists(path) {
		backupPath := path + ".backup." + time.Now().Format("20060102150405")
		data, err := i.fs.ReadFile(path)
		if err == nil {
			_ = i.fs.WriteFile(backupPath, data, 0644)
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	return i.fs.WriteFile(path, data, 0644)
}

// checkRequiredCommands verifica si los comandos necesarios para un MCP están disponibles
func (i *Installer) checkRequiredCommands(mcp *manifest.MCP) error {
	srcPath := i.repoMgr.GetFilePath(mcp.Path)
	data, err := i.fs.ReadFile(srcPath)
	if err != nil {
		return nil // Si no podemos leer el archivo, no podemos verificar
	}

	var config struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil // Si no es un MCP tipo command, ignorar
	}

	if config.Command == "" {
		return nil // Es un MCP HTTP/SSE, no requiere verificación
	}

	// Verificar si el comando está en el PATH
	if _, err := exec.LookPath(config.Command); err != nil {
		var installMsg string
		switch config.Command {
		case "uvx":
			installMsg = "uvx not found in PATH. Install with: curl -LsSf https://astral.sh/uv/install.sh | sh"
		case "npx":
			installMsg = "npx not found in PATH. Install with: npm install -g npm"
		case "docker":
			installMsg = "docker not found in PATH. Install from: https://docs.docker.com/get-docker/"
		case "pip", "pip3":
			installMsg = fmt.Sprintf("%s not found in PATH. Install with: python3 -m ensurepip", config.Command)
		default:
			installMsg = fmt.Sprintf("%s not found in PATH. Please install it first.", config.Command)
		}
		return fmt.Errorf("%s", installMsg)
	}

	return nil
}
