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

// SkillEntry represents an installed skill with its source information.
// For skills installed from online repositories, Source will be set (e.g., "owner/repo").
// For skills installed from local repositories, Source will be empty.
type SkillEntry struct {
	Name   string `json:"name"`            // Skill name
	Source string `json:"source,omitempty"` // Repository source (owner/repo) for online skills
	Path   string `json:"path,omitempty"` // Subpath within the repository (if applicable)
	Commit string `json:"commit,omitempty"` // Commit hash for reproducibility
}

// UnmarshalJSON implements custom JSON unmarshaling for backward compatibility.
// It handles both the old format ([]string) and the new format ([]SkillEntry).
func (s *SkillEntry) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string (old format)
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		s.Name = str
		s.Source = ""
		s.Path = ""
		s.Commit = ""
		return nil
	}

	// Try to unmarshal as struct (new format)
	type skillEntryAlias SkillEntry // Avoid recursion
	var alias skillEntryAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*s = SkillEntry(alias)
	return nil
}

type InstalledAssets struct {
	Rules    []string     `json:"rules,omitempty"`
	Skills   []SkillEntry `json:"skills,omitempty"`
	Agents   []string     `json:"agents,omitempty"`
	Hooks    []string     `json:"hooks,omitempty"`
	MCP      []string     `json:"mcp,omitempty"`
	AgentsMD []string     `json:"agentsMd,omitempty"`
	External []string     `json:"external,omitempty"`
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
	return filepath.Join(t.projectRoot, LockFileName)
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

// RecordInstall records a single asset installation.
// For skills with source information (online repos), use RecordSkillInstall.
// This method still supports skills for backward compatibility, creating a SkillEntry without source.
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
		// For backward compatibility: create SkillEntry without source info
		entry := SkillEntry{Name: name}
		lock.Assets.Skills = addUniqueSkill(lock.Assets.Skills, entry)
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

// RecordSkillInstall records a skill installation with full source information.
// Use this instead of RecordInstall when installing skills (especially from online sources).
// This updates the project metadata (repoURL, repoCommit) - use RecordSkillInstallOnly to preserve them.
func (t *Tracker) RecordSkillInstall(entry SkillEntry, repoURL, repoCommit string) error {
	lock, err := t.Load()
	if err != nil {
		return err
	}

	lock.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	lock.RepoURL = repoURL
	lock.RepoCommit = repoCommit
	lock.Target = t.target.Name

	lock.Assets.Skills = addUniqueSkill(lock.Assets.Skills, entry)

	return t.Save(lock)
}

// RecordSkillInstallOnly records a skill installation without modifying project metadata.
// Use this when installing skills from external/online sources to preserve the project's repoURL and repoCommit.
func (t *Tracker) RecordSkillInstallOnly(entry SkillEntry) error {
	lock, err := t.Load()
	if err != nil {
		return err
	}

	lock.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	// Preserve existing repoURL and repoCommit - don't modify them
	lock.Target = t.target.Name

	lock.Assets.Skills = addUniqueSkill(lock.Assets.Skills, entry)

	return t.Save(lock)
}

// InstallRecord represents a single asset installation result.
type InstallRecord struct {
	Name   string
	Type   manifest.AssetType
	Source string // Repository source for skills (owner/repo format)
	Path   string // Subpath within the repository
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
			// Create SkillEntry with source information if available
			entry := SkillEntry{
				Name:   r.Name,
				Source: r.Source,
				Path:   r.Path,
				Commit: repoCommit,
			}
			lock.Assets.Skills = addUniqueSkill(lock.Assets.Skills, entry)
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
		return containsSkill(lock.Assets.Skills, name), nil
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
		lock.Assets.Skills = removeSkill(lock.Assets.Skills, name)
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

// SkillEntry helper functions

func addUniqueSkill(list []SkillEntry, item SkillEntry) []SkillEntry {
	for _, v := range list {
		if v.Name == item.Name {
			// Update existing entry if source is being added
			if item.Source != "" && v.Source == "" {
				v.Source = item.Source
				v.Path = item.Path
				v.Commit = item.Commit
			}
			return list
		}
	}
	return append(list, item)
}

func containsSkill(list []SkillEntry, name string) bool {
	for _, v := range list {
		if v.Name == name {
			return true
		}
	}
	return false
}

func removeSkill(list []SkillEntry, name string) []SkillEntry {
	result := make([]SkillEntry, 0, len(list))
	for _, v := range list {
		if v.Name != name {
			result = append(result, v)
		}
	}
	return result
}
