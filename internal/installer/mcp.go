package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rosseca/aisi/internal/deps"
	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/repo"
)

type MCPConfig struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

type EnvVarConfig struct {
	VarName string
	Value   string
	UseEnv  bool
}

func (i *Installer) InstallMCP(mcp *manifest.MCP, envVars map[string]EnvVarConfig, m *manifest.Manifest) (*InstallResult, error) {
	if i.target.MCPFile == "" {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Success: false,
			Error:   fmt.Errorf("target %s does not support MCP", i.target.Name),
		}, nil
	}

	mcpPath := i.target.MCPPath(i.projectRoot)
	return i.installMCPInternal(mcp, envVars, m, mcpPath, false)
}

func (i *Installer) InstallMCPGlobal(mcp *manifest.MCP, envVars map[string]EnvVarConfig, m *manifest.Manifest) (*InstallResult, error) {
	if i.target.MCPFile == "" {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Success: false,
			Error:   fmt.Errorf("target %s does not support MCP", i.target.Name),
		}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	mcpPath := filepath.Join(home, i.target.ConfigDir, i.target.MCPFile)
	return i.installMCPInternal(mcp, envVars, m, mcpPath, true)
}

// installMCPInternal contiene la lógica compartida de instalación de MCP
func (i *Installer) installMCPInternal(mcp *manifest.MCP, envVars map[string]EnvVarConfig, m *manifest.Manifest, mcpPath string, isGlobal bool) (*InstallResult, error) {
	// 1. Verificar e instalar dependencias del sistema (command + install)
	if mcp.Command != "" && mcp.Install != nil {
		depsMgr := deps.NewManager()
		
		// Pass progress callback to deps manager
		if i.onProgress != nil {
			depsMgr.SetProgressCallback(i.onProgress)
		}
		
		if !depsMgr.CheckCommand(mcp.Command) {
			// Comando no existe, intentar instalar
			msg := fmt.Sprintf("📦 Installing dependency '%s' for MCP '%s'...", mcp.Command, mcp.Name)
			if i.onProgress != nil {
				i.onProgress(msg)
			}
			fmt.Println(msg)
			
			if err := depsMgr.Install(mcp.Install); err != nil {
				return &InstallResult{
					Name:    mcp.Name,
					Type:    manifest.AssetTypeMCP,
					Success: false,
					Error:   fmt.Errorf("command '%s' not available and installation failed: %w", mcp.Command, err),
				}, nil
			}
			
			successMsg := fmt.Sprintf("✓ Dependency '%s' installed successfully", mcp.Command)
			if i.onProgress != nil {
				i.onProgress(successMsg)
			}
			fmt.Println(successMsg)
		}
	}

	// 2. Verificar comando del JSON
	if err := i.checkRequiredCommands(mcp); err != nil {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Success: false,
			Error:   err,
		}, nil
	}

	// 3. Crear directorio si no existe
	if err := i.fs.MkdirAll(filepath.Dir(mcpPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create MCP directory: %w", err)
	}

	// 4. Cargar configuración del servidor MCP
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

	// 5. Cargar configuración existente y mergear
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

	// 6. Guardar configuración
	if err := i.saveMCPConfig(mcpPath, existingConfig); err != nil {
		return &InstallResult{
			Name:    mcp.Name,
			Type:    manifest.AssetTypeMCP,
			Path:    mcpPath,
			Success: false,
			Error:   err,
		}, nil
	}

	// MCP instalado exitosamente
	result := &InstallResult{
		Name:    mcp.Name,
		Type:    manifest.AssetTypeMCP,
		Path:    mcpPath,
		Success: true,
	}

	// 7. Ejecutar postInstall si está definido
	if mcp.PostInstall != nil {
		postInstallResult := i.executePostInstall(mcp, envVars)
		result.AdditionalResults = append(result.AdditionalResults, postInstallResult)
	}

	// 8. Instalar skill asociada si está definida
	if mcp.Skill != nil {
		skillResult := i.installAssociatedSkill(mcp, m)
		result.AdditionalResults = append(result.AdditionalResults, skillResult)
	}

	return result, nil
}

// executePostInstall ejecuta el comando post-install definido en el MCP
func (i *Installer) executePostInstall(mcp *manifest.MCP, envVars map[string]EnvVarConfig) *InstallResult {
	postInstall := mcp.PostInstall

	// Reportar inicio del post-install
	cmdStr := postInstall.Command
	if len(postInstall.Args) > 0 {
		cmdStr += " " + strings.Join(postInstall.Args, " ")
	}
	msg := fmt.Sprintf("🔧 Running MCP post-install: %s", cmdStr)
	if i.onProgress != nil {
		i.onProgress(msg)
	}
	fmt.Println(msg)

	cmd := exec.Command(postInstall.Command, postInstall.Args...)

	// Configurar variables de entorno
	cmd.Env = os.Environ()

	// Añadir env vars del MCP si están definidas
	if mcp.Env != nil {
		for varName := range mcp.Env {
			if val, ok := envVars[varName]; ok {
				if val.UseEnv {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", varName, os.Getenv(varName)))
				} else {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", varName, val.Value))
				}
			}
		}
	}

	// Añadir env vars específicas del postInstall
	for k, v := range postInstall.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Ejecutar el comando
	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("✗ Post-install failed: %v", err)
		if i.onProgress != nil {
			i.onProgress(errMsg)
		}
		return &InstallResult{
			Name:    mcp.Name + "-postinstall",
			Type:    "postinstall",
			Success: false,
			Error:   fmt.Errorf("post-install failed: %w", err),
		}
	}

	successMsg := "✓ MCP post-install completed"
	if i.onProgress != nil {
		i.onProgress(successMsg)
	}

	return &InstallResult{
		Name:    mcp.Name + "-postinstall",
		Type:    "postinstall",
		Success: true,
	}
}

// installAssociatedSkill instala la skill asociada al MCP
func (i *Installer) installAssociatedSkill(mcp *manifest.MCP, m *manifest.Manifest) *InstallResult {
	skillRef := mcp.Skill

	if skillRef.IsLocal() {
		// Buscar skill local en el manifest
		skill := m.GetSkill(skillRef.Name)
		if skill == nil {
			return &InstallResult{
				Name:    skillRef.Name,
				Type:    manifest.AssetTypeSkill,
				Success: false,
				Error:   fmt.Errorf("associated skill '%s' not found in manifest", skillRef.Name),
			}
		}

		skillResult, _ := i.InstallSkill(skill)
		return skillResult
	}

	// Skill externa - construir URL
	skillURL := &repo.SkillURL{
		RepoURL: skillRef.Repo,
		Path:    skillRef.Path,
		Ref:     skillRef.Ref,
	}
	if skillRef.Ref == "" {
		skillURL.Ref = "main" // default
	}

	// Determinar nombre de la skill
	skillName := skillRef.Name
	if skillName == "" && skillRef.Path != "" {
		skillName = filepath.Base(skillRef.Path)
	}

	skillResult, _ := i.InstallSkillFromURL(skillURL, skillName)
	return skillResult
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
