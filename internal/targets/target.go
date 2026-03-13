package targets

import (
	"fmt"
	"path/filepath"
)

type Target struct {
	Name                 string
	DisplayName          string
	ConfigDir            string
	RulesDir             string
	SkillsDir            string
	AgentsDir            string
	HooksFile            string
	HooksScriptsDir      string
	MCPFile              string
	SupportsAgentsMD     bool
	SupportsModeSpecific bool
}

func (t *Target) RulesPath(projectRoot string) string {
	if t.RulesDir == "" {
		return ""
	}
	return filepath.Join(projectRoot, t.ConfigDir, t.RulesDir)
}

func (t *Target) SkillsPath(projectRoot string) string {
	if t.SkillsDir == "" {
		return ""
	}
	return filepath.Join(projectRoot, t.ConfigDir, t.SkillsDir)
}

func (t *Target) AgentsPath(projectRoot string) string {
	if t.AgentsDir == "" {
		return ""
	}
	return filepath.Join(projectRoot, t.ConfigDir, t.AgentsDir)
}

func (t *Target) HooksConfigPath(projectRoot string) string {
	if t.HooksFile == "" {
		return ""
	}
	return filepath.Join(projectRoot, t.ConfigDir, t.HooksFile)
}

func (t *Target) HooksScriptsPath(projectRoot string) string {
	if t.HooksScriptsDir == "" {
		return ""
	}
	return filepath.Join(projectRoot, t.ConfigDir, t.HooksScriptsDir)
}

func (t *Target) MCPPath(projectRoot string) string {
	if t.MCPFile == "" {
		return ""
	}
	return filepath.Join(projectRoot, t.ConfigDir, t.MCPFile)
}

func (t *Target) ConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, t.ConfigDir)
}

type Registry struct {
	targets map[string]*Target
}

func NewRegistry() *Registry {
	r := &Registry{
		targets: make(map[string]*Target),
	}
	r.registerBuiltins()
	return r
}

func (r *Registry) registerBuiltins() {
	r.Register(CursorTarget)
	r.Register(KiloTarget)
	r.Register(JunieTarget)
}

func (r *Registry) Register(t *Target) {
	r.targets[t.Name] = t
}

func (r *Registry) Get(name string) (*Target, error) {
	t, ok := r.targets[name]
	if !ok {
		return nil, fmt.Errorf("unknown target: %s", name)
	}
	return t, nil
}

func (r *Registry) List() []*Target {
	result := make([]*Target, 0, len(r.targets))
	for _, t := range r.targets {
		result = append(result, t)
	}
	return result
}

func (r *Registry) Names() []string {
	result := make([]string, 0, len(r.targets))
	for name := range r.targets {
		result = append(result, name)
	}
	return result
}

var DefaultRegistry = NewRegistry()

func Get(name string) (*Target, error) {
	return DefaultRegistry.Get(name)
}

func List() []*Target {
	return DefaultRegistry.List()
}

func Names() []string {
	return DefaultRegistry.Names()
}
