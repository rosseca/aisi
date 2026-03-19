package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDirName  = ".aisi"
	ConfigFileName = "config.yaml"
	CacheDirName   = "cache"
)

type RepoConfig struct {
	URL    string `yaml:"url"`
	Branch string `yaml:"branch"`
}

type CustomTarget struct {
	DisplayName     string `yaml:"displayName"`
	ConfigDir       string `yaml:"configDir"`
	RulesDir        string `yaml:"rulesDir"`
	SkillsDir       string `yaml:"skillsDir"`
	AgentsDir       string `yaml:"agentsDir"`
	HooksFile       string `yaml:"hooksFile"`
	HooksScriptsDir string `yaml:"hooksScriptsDir"`
	MCPFile         string `yaml:"mcpFile"`
}

type Config struct {
	Repo           RepoConfig              `yaml:"repo"`
	HTTPSToken     string                  `yaml:"httpsToken,omitempty"`
	SkillsMPAPIKey string                  `yaml:"skillsmpApiKey,omitempty"`
	ActiveTarget   string                  `yaml:"activeTarget"`
	CustomTargets  map[string]CustomTarget `yaml:"customTargets,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		Repo: RepoConfig{
			URL:    "",
			Branch: "main",
		},
		ActiveTarget:  "cursor",
		CustomTargets: make(map[string]CustomTarget),
	}
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ConfigDirName), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

func CacheDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, CacheDirName), nil
}

func ExternalCacheDir() (string, error) {
	cache, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "external"), nil
}

func Load() (*Config, error) {
	cfg, _, err := LoadWithExists()
	return cfg, err
}

// LoadWithExists loads the config and returns whether the config file existed
func LoadWithExists() (*Config, bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, false, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), false, nil
		}
		return nil, false, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, false, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.ActiveTarget == "" {
		cfg.ActiveTarget = "cursor"
	}
	if cfg.Repo.Branch == "" {
		cfg.Repo.Branch = "main"
	}
	if cfg.CustomTargets == nil {
		cfg.CustomTargets = make(map[string]CustomTarget)
	}

	return &cfg, true, nil
}

func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *Config) SetRepo(url, branch string) {
	c.Repo.URL = url
	if branch != "" {
		c.Repo.Branch = branch
	}
}

func (c *Config) SetActiveTarget(target string) {
	c.ActiveTarget = target
}

func (c *Config) SetHTTPSToken(token string) {
	c.HTTPSToken = token
}

func (c *Config) GetToken() string {
	if c.HTTPSToken != "" {
		return c.HTTPSToken
	}
	return os.Getenv("GITHUB_TOKEN")
}

func (c *Config) SetSkillsMPAPIKey(key string) {
	c.SkillsMPAPIKey = key
}

func (c *Config) GetSkillsMPAPIKey() string {
	if c.SkillsMPAPIKey != "" {
		return c.SkillsMPAPIKey
	}
	return os.Getenv("SKILLSMP_API_KEY")
}

func (c *Config) IsConfigured() bool {
	return c.Repo.URL != ""
}

func EnsureConfigDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

func EnsureCacheDir() error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}
