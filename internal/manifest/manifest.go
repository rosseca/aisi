package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type AssetType string

const (
	AssetTypeRule     AssetType = "rule"
	AssetTypeSkill    AssetType = "skill"
	AssetTypeAgent    AssetType = "agent"
	AssetTypeHook     AssetType = "hook"
	AssetTypeMCP      AssetType = "mcp"
	AssetTypeAgentsMD AssetType = "agentsMd"
)

type Rule struct {
	Name        string   `json:"name" yaml:"name"`
	Path        string   `json:"path" yaml:"path"`
	Description string   `json:"description" yaml:"description"`
	Categories  []string `json:"categories,omitempty" yaml:"categories,omitempty"`
	AlwaysApply bool     `json:"alwaysApply,omitempty" yaml:"alwaysApply,omitempty"`
	Globs       []string `json:"globs,omitempty" yaml:"globs,omitempty"`
}

type Skill struct {
	Name        string         `json:"name" yaml:"name"`
	Path        string         `json:"path" yaml:"path"`
	Description string         `json:"description" yaml:"description"`
	Categories  []string       `json:"categories,omitempty" yaml:"categories,omitempty"`
	Command     string         `json:"command,omitempty" yaml:"command,omitempty"`
	Install     *InstallConfig `json:"install,omitempty" yaml:"install,omitempty"`
}

// InstallConfig defines how to install the required command per OS or globally
type InstallConfig struct {
	MacOS       *MacOSInstall `json:"macos,omitempty" yaml:"macos,omitempty"`
	Linux       *LinuxInstall `json:"linux,omitempty" yaml:"linux,omitempty"`
	Npm         *NpmInstall   `json:"npm,omitempty" yaml:"npm,omitempty"`              // npm global (multiplataforma)
	PostInstall string        `json:"post_install,omitempty" yaml:"post_install,omitempty"` // Optional: command to run after successful installation
}

// MacOSInstall defines macOS-specific installation via brew
type MacOSInstall struct {
	Brew *BrewInstall `json:"brew,omitempty" yaml:"brew,omitempty"`
}

// BrewInstall defines brew package installation
type BrewInstall struct {
	Tap     string `json:"tap,omitempty" yaml:"tap,omitempty"`         // Optional: brew tap
	Command string `json:"command" yaml:"command"`                     // Required: package name
}

// LinuxInstall defines Linux-specific installation via apt
type LinuxInstall struct {
	Apt *AptInstall `json:"apt,omitempty" yaml:"apt,omitempty"`
}

// AptInstall defines apt package installation
type AptInstall struct {
	Sources     []string `json:"sources,omitempty" yaml:"sources,omitempty"`     // Optional: apt sources
	Command     string   `json:"command" yaml:"command"`                         // Required: package name
	PostInstall string   `json:"post_install,omitempty" yaml:"post_install,omitempty"` // Optional: post-install command
}

// NpmPackage represents a single npm package to install
type NpmPackage struct {
	Package string `json:"package" yaml:"package"`                    // Required: npm package name
	Version string `json:"version,omitempty" yaml:"version,omitempty"` // Optional: specific version (e.g., "^1.0.0", "latest")
}

// NpmInstall defines npm global package installation (multiplataforma)
type NpmInstall struct {
	Packages []NpmPackage `json:"packages" yaml:"packages"` // Required: list of npm packages to install
}

type Agent struct {
	Name        string   `json:"name" yaml:"name"`
	Path        string   `json:"path" yaml:"path"`
	Description string   `json:"description" yaml:"description"`
	Categories  []string `json:"categories,omitempty" yaml:"categories,omitempty"`
	Model       string   `json:"model,omitempty" yaml:"model,omitempty"`
}

type Hook struct {
	Name        string   `json:"name" yaml:"name"`
	ConfigPath  string   `json:"configPath" yaml:"configPath"`
	ScriptsPath string   `json:"scriptsPath" yaml:"scriptsPath"`
	Description string   `json:"description" yaml:"description"`
	Categories  []string `json:"categories,omitempty" yaml:"categories,omitempty"`
}

type EnvVarMeta struct {
	Description string `json:"description" yaml:"description"`
	Example     string `json:"example,omitempty" yaml:"example,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Secret      bool   `json:"secret,omitempty" yaml:"secret,omitempty"`
	HelpURL     string `json:"helpUrl,omitempty" yaml:"helpUrl,omitempty"`
}

type SkillRef struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Repo string `json:"repo,omitempty" yaml:"repo,omitempty"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty"`
}

// IsLocal devuelve true si es una referencia local (solo Name)
func (s SkillRef) IsLocal() bool {
	return s.Name != "" && s.Repo == ""
}

// IsExternal devuelve true si es una referencia externa (tiene Repo)
func (s SkillRef) IsExternal() bool {
	return s.Repo != ""
}

// PostInstallConfig define un comando a ejecutar después de instalar el MCP
type PostInstallConfig struct {
	Command     string            `json:"command" yaml:"command"`
	Args        []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

type MCP struct {
	Name        string                `json:"name" yaml:"name"`
	Path        string                `json:"path" yaml:"path"`
	Description string                `json:"description" yaml:"description"`
	Categories  []string              `json:"categories,omitempty" yaml:"categories,omitempty"`
	Env         map[string]EnvVarMeta `json:"env,omitempty" yaml:"env,omitempty"`
	Skill       *SkillRef             `json:"skill,omitempty" yaml:"skill,omitempty"`
	Command     string                `json:"command,omitempty" yaml:"command,omitempty"`
	Install     *InstallConfig        `json:"install,omitempty" yaml:"install,omitempty"`
	PostInstall *PostInstallConfig    `json:"postInstall,omitempty" yaml:"postInstall,omitempty"`
}

type AgentsMD struct {
	Name        string   `json:"name" yaml:"name"`
	Path        string   `json:"path" yaml:"path"`
	Description string   `json:"description" yaml:"description"`
	Categories  []string `json:"categories,omitempty" yaml:"categories,omitempty"`
}

type External struct {
	Name         string         `json:"name" yaml:"name"`
	Type         string         `json:"type" yaml:"type"`
	Repo         string         `json:"repo" yaml:"repo"`
	Path         string         `json:"path" yaml:"path"`
	Ref          string         `json:"ref,omitempty" yaml:"ref,omitempty"`
	Description  string         `json:"description" yaml:"description"`
	Categories   []string       `json:"categories,omitempty" yaml:"categories,omitempty"`
	Requirements []string       `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	Command      string         `json:"command,omitempty" yaml:"command,omitempty"`
	Install      *InstallConfig `json:"install,omitempty" yaml:"install,omitempty"`
}

type Manifest struct {
	Version           string     `json:"version" yaml:"version"`
	MinimumCLIVersion string     `json:"minimumCliVersion,omitempty" yaml:"minimumCliVersion,omitempty"`
	Rules             []Rule     `json:"rules,omitempty" yaml:"rules,omitempty"`
	Skills            []Skill    `json:"skills,omitempty" yaml:"skills,omitempty"`
	Agents            []Agent    `json:"agents,omitempty" yaml:"agents,omitempty"`
	Hooks             []Hook     `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	MCP               []MCP      `json:"mcp,omitempty" yaml:"mcp,omitempty"`
	AgentsMD          []AgentsMD `json:"agentsMd,omitempty" yaml:"agentsMd,omitempty"`
	External          []External `json:"external,omitempty" yaml:"external,omitempty"`
}

// VersionMismatchError is returned when CLI version is below minimum required
type VersionMismatchError struct {
	CurrentVersion  string
	RequiredVersion string
}

func (e *VersionMismatchError) Error() string {
	return fmt.Sprintf("CLI version %s is below minimum required %s",
		e.CurrentVersion, e.RequiredVersion)
}

// parseVersion parses a semantic version string (e.g., "1.2.3") into comparable components
func parseVersion(v string) (major, minor, patch int, err error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")

	if len(parts) < 2 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", v)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch = 0
	if len(parts) >= 3 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid patch version: %s", parts[2])
		}
	}

	return major, minor, patch, nil
}

// CheckCLIVersion compares current CLI version with minimum required version
// Returns nil if current >= required, or VersionMismatchError if current < required
// Special cases: "dev", "snapshot" versions always pass (development builds)
func (m *Manifest) CheckCLIVersion(currentVersion string) error {
	if m.MinimumCLIVersion == "" {
		return nil // No minimum version specified
	}

	// Dev/snapshot builds always pass version check (for development/testing)
	if currentVersion == "dev" || currentVersion == "" || strings.Contains(currentVersion, "snapshot") {
		return nil
	}

	currentMajor, currentMinor, currentPatch, err := parseVersion(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to parse current CLI version: %w", err)
	}

	requiredMajor, requiredMinor, requiredPatch, err := parseVersion(m.MinimumCLIVersion)
	if err != nil {
		return fmt.Errorf("failed to parse minimum CLI version from manifest: %w", err)
	}

	// Compare versions
	if currentMajor < requiredMajor {
		return &VersionMismatchError{CurrentVersion: currentVersion, RequiredVersion: m.MinimumCLIVersion}
	}
	if currentMajor == requiredMajor && currentMinor < requiredMinor {
		return &VersionMismatchError{CurrentVersion: currentVersion, RequiredVersion: m.MinimumCLIVersion}
	}
	if currentMajor == requiredMajor && currentMinor == requiredMinor && currentPatch < requiredPatch {
		return &VersionMismatchError{CurrentVersion: currentVersion, RequiredVersion: m.MinimumCLIVersion}
	}

	return nil
}

func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	return Parse(data, path)
}

func Parse(data []byte, path ...string) (*Manifest, error) {
	var m Manifest
	// Determine format from file extension if path provided
	isYAML := false
	if len(path) > 0 {
		ext := strings.ToLower(filepath.Ext(path[0]))
		// Handle .yaml and .yml extensions
		if ext == ".yaml" || ext == ".yml" {
			isYAML = true
		}
	}

	if isYAML {
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("failed to parse YAML manifest: %w", err)
		}
	} else {
		// Default to YAML parsing as it's more lenient and handles both formats
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}
	}
	return &m, nil
}

func (m *Manifest) GetRule(name string) *Rule {
	for i := range m.Rules {
		if m.Rules[i].Name == name {
			return &m.Rules[i]
		}
	}
	return nil
}

func (m *Manifest) GetSkill(name string) *Skill {
	for i := range m.Skills {
		if m.Skills[i].Name == name {
			return &m.Skills[i]
		}
	}
	return nil
}

func (m *Manifest) GetAgent(name string) *Agent {
	for i := range m.Agents {
		if m.Agents[i].Name == name {
			return &m.Agents[i]
		}
	}
	return nil
}

func (m *Manifest) GetHook(name string) *Hook {
	for i := range m.Hooks {
		if m.Hooks[i].Name == name {
			return &m.Hooks[i]
		}
	}
	return nil
}

func (m *Manifest) GetMCP(name string) *MCP {
	for i := range m.MCP {
		if m.MCP[i].Name == name {
			return &m.MCP[i]
		}
	}
	return nil
}

func (m *Manifest) GetAgentsMD(name string) *AgentsMD {
	for i := range m.AgentsMD {
		if m.AgentsMD[i].Name == name {
			return &m.AgentsMD[i]
		}
	}
	return nil
}

func (m *Manifest) GetExternal(name string) *External {
	for i := range m.External {
		if m.External[i].Name == name {
			return &m.External[i]
		}
	}
	return nil
}

func (m *Manifest) FindAsset(name string) (AssetType, interface{}) {
	if r := m.GetRule(name); r != nil {
		return AssetTypeRule, r
	}
	if s := m.GetSkill(name); s != nil {
		return AssetTypeSkill, s
	}
	if a := m.GetAgent(name); a != nil {
		return AssetTypeAgent, a
	}
	if h := m.GetHook(name); h != nil {
		return AssetTypeHook, h
	}
	if mc := m.GetMCP(name); mc != nil {
		return AssetTypeMCP, mc
	}
	if am := m.GetAgentsMD(name); am != nil {
		return AssetTypeAgentsMD, am
	}
	if e := m.GetExternal(name); e != nil {
		// Return a special asset type for external assets
		return AssetType("external:" + e.Type), e
	}
	return "", nil
}

func (m *Manifest) ListRuleNames() []string {
	names := make([]string, len(m.Rules))
	for i, r := range m.Rules {
		names[i] = r.Name
	}
	return names
}

func (m *Manifest) ListSkillNames() []string {
	names := make([]string, len(m.Skills))
	for i, s := range m.Skills {
		names[i] = s.Name
	}
	return names
}

func (m *Manifest) ListAgentNames() []string {
	names := make([]string, len(m.Agents))
	for i, a := range m.Agents {
		names[i] = a.Name
	}
	return names
}

func (m *Manifest) ListHookNames() []string {
	names := make([]string, len(m.Hooks))
	for i, h := range m.Hooks {
		names[i] = h.Name
	}
	return names
}

func (m *Manifest) ListMCPNames() []string {
	names := make([]string, len(m.MCP))
	for i, mc := range m.MCP {
		names[i] = mc.Name
	}
	return names
}

func (m *Manifest) ListAgentsMDNames() []string {
	names := make([]string, len(m.AgentsMD))
	for i, am := range m.AgentsMD {
		names[i] = am.Name
	}
	return names
}

func (m *Manifest) ListExternalNames() []string {
	names := make([]string, len(m.External))
	for i, e := range m.External {
		names[i] = e.Name
	}
	return names
}

func (m *Manifest) GetExternalByType(assetType string) []External {
	var result []External
	for _, e := range m.External {
		if e.Type == assetType {
			result = append(result, e)
		}
	}
	return result
}

func (m *Manifest) AllSkills() []Skill {
	skills := make([]Skill, len(m.Skills))
	copy(skills, m.Skills)
	return skills
}

func (m *Manifest) AllExternalSkills() []External {
	return m.GetExternalByType("skill")
}

func (m *Manifest) AllAgents() []Agent {
	agents := make([]Agent, len(m.Agents))
	copy(agents, m.Agents)
	return agents
}

func (m *Manifest) AllExternalAgents() []External {
	return m.GetExternalByType("agent")
}
