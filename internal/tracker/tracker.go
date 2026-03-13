package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/targets"
)

const LockFileName = ".aisi.lock"

type InstalledAssets struct {
	Rules    []string `json:"rules,omitempty"`
	Skills   []string `json:"skills,omitempty"`
	Agents   []string `json:"agents,omitempty"`
	Hooks    []string `json:"hooks,omitempty"`
	MCP      []string `json:"mcp,omitempty"`
	AgentsMD []string `json:"agentsMd,omitempty"`
	External []string `json:"external,omitempty"`
}

type LockFile struct {
	InstalledAt string          `json:"installedAt"`
	RepoURL     string          `json:"repoURL"`
	RepoCommit  string          `json:"repoCommit"`
	Target      string          `json:"target"`
	Assets      InstalledAssets `json:"assets"`
}

type Tracker struct {
	projectRoot string
	target      *targets.Target
}

func New(projectRoot string, target *targets.Target) *Tracker {
	return &Tracker{
		projectRoot: projectRoot,
		target:      target,
	}
}

func (t *Tracker) lockFilePath() string {
	return filepath.Join(t.projectRoot, t.target.ConfigDir, LockFileName)
}

func (t *Tracker) Load() (*LockFile, error) {
	path := t.lockFilePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LockFile{
				Assets: InstalledAssets{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lock LockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	return &lock, nil
}

func (t *Tracker) Save(lock *LockFile) error {
	path := t.lockFilePath()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create lock file directory: %w", err)
	}

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

func (t *Tracker) RecordInstall(assetType manifest.AssetType, name string, repoURL, repoCommit string) error {
	lock, err := t.Load()
	if err != nil {
		return err
	}

	lock.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	lock.RepoURL = repoURL
	lock.RepoCommit = repoCommit
	lock.Target = t.target.Name

	switch assetType {
	case manifest.AssetTypeRule:
		lock.Assets.Rules = addUnique(lock.Assets.Rules, name)
	case manifest.AssetTypeSkill:
		lock.Assets.Skills = addUnique(lock.Assets.Skills, name)
	case manifest.AssetTypeAgent:
		lock.Assets.Agents = addUnique(lock.Assets.Agents, name)
	case manifest.AssetTypeHook:
		lock.Assets.Hooks = addUnique(lock.Assets.Hooks, name)
	case manifest.AssetTypeMCP:
		lock.Assets.MCP = addUnique(lock.Assets.MCP, name)
	case manifest.AssetTypeAgentsMD:
		lock.Assets.AgentsMD = addUnique(lock.Assets.AgentsMD, name)
	default:
		lock.Assets.External = addUnique(lock.Assets.External, name)
	}

	return t.Save(lock)
}

func (t *Tracker) RecordInstalls(results []InstallRecord, repoURL, repoCommit string) error {
	lock, err := t.Load()
	if err != nil {
		return err
	}

	lock.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	lock.RepoURL = repoURL
	lock.RepoCommit = repoCommit
	lock.Target = t.target.Name

	for _, r := range results {
		switch r.Type {
		case manifest.AssetTypeRule:
			lock.Assets.Rules = addUnique(lock.Assets.Rules, r.Name)
		case manifest.AssetTypeSkill:
			lock.Assets.Skills = addUnique(lock.Assets.Skills, r.Name)
		case manifest.AssetTypeAgent:
			lock.Assets.Agents = addUnique(lock.Assets.Agents, r.Name)
		case manifest.AssetTypeHook:
			lock.Assets.Hooks = addUnique(lock.Assets.Hooks, r.Name)
		case manifest.AssetTypeMCP:
			lock.Assets.MCP = addUnique(lock.Assets.MCP, r.Name)
		case manifest.AssetTypeAgentsMD:
			lock.Assets.AgentsMD = addUnique(lock.Assets.AgentsMD, r.Name)
		default:
			lock.Assets.External = addUnique(lock.Assets.External, r.Name)
		}
	}

	return t.Save(lock)
}

type InstallRecord struct {
	Name string
	Type manifest.AssetType
}

func (t *Tracker) IsInstalled(assetType manifest.AssetType, name string) (bool, error) {
	lock, err := t.Load()
	if err != nil {
		return false, err
	}

	var list []string
	switch assetType {
	case manifest.AssetTypeRule:
		list = lock.Assets.Rules
	case manifest.AssetTypeSkill:
		list = lock.Assets.Skills
	case manifest.AssetTypeAgent:
		list = lock.Assets.Agents
	case manifest.AssetTypeHook:
		list = lock.Assets.Hooks
	case manifest.AssetTypeMCP:
		list = lock.Assets.MCP
	case manifest.AssetTypeAgentsMD:
		list = lock.Assets.AgentsMD
	default:
		list = lock.Assets.External
	}

	return contains(list, name), nil
}

func (t *Tracker) GetInstalled() (*InstalledAssets, error) {
	lock, err := t.Load()
	if err != nil {
		return nil, err
	}
	return &lock.Assets, nil
}

func (t *Tracker) GetRepoCommit() (string, error) {
	lock, err := t.Load()
	if err != nil {
		return "", err
	}
	return lock.RepoCommit, nil
}

func (t *Tracker) GetRepoURL() (string, error) {
	lock, err := t.Load()
	if err != nil {
		return "", err
	}
	return lock.RepoURL, nil
}

func (t *Tracker) Remove(assetType manifest.AssetType, name string) error {
	lock, err := t.Load()
	if err != nil {
		return err
	}

	switch assetType {
	case manifest.AssetTypeRule:
		lock.Assets.Rules = remove(lock.Assets.Rules, name)
	case manifest.AssetTypeSkill:
		lock.Assets.Skills = remove(lock.Assets.Skills, name)
	case manifest.AssetTypeAgent:
		lock.Assets.Agents = remove(lock.Assets.Agents, name)
	case manifest.AssetTypeHook:
		lock.Assets.Hooks = remove(lock.Assets.Hooks, name)
	case manifest.AssetTypeMCP:
		lock.Assets.MCP = remove(lock.Assets.MCP, name)
	case manifest.AssetTypeAgentsMD:
		lock.Assets.AgentsMD = remove(lock.Assets.AgentsMD, name)
	default:
		lock.Assets.External = remove(lock.Assets.External, name)
	}

	return t.Save(lock)
}

func addUnique(list []string, item string) []string {
	for _, v := range list {
		if v == item {
			return list
		}
	}
	return append(list, item)
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func remove(list []string, item string) []string {
	result := make([]string, 0, len(list))
	for _, v := range list {
		if v != item {
			result = append(result, v)
		}
	}
	return result
}
