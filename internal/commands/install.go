package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/rosseca/aisi/internal/installer"
	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/prompt"
	"github.com/rosseca/aisi/internal/repo"
	"github.com/rosseca/aisi/internal/tracker"
	"github.com/rosseca/aisi/internal/version"
	"github.com/spf13/cobra"
)

var (
	installType   string
	installAll    bool
	installGlobal bool
	installURL    string
	installName   string
)

var installCmd = &cobra.Command{
	Use:   "install [asset-name...]",
	Short: "Install assets from the shared repository",
	Long: `Install one or more assets (rules, skills, agents, hooks, MCP) from the
shared repository into the current project.

When run without arguments, it will install all assets recorded in the .aisi.lock
file, using the repository URL from the lock file if available.

Examples:
  aisi install                       # Install all assets from lock file
  aisi install soul typescript       # Install specific assets
  aisi install --type=rules --all    # Install all rules
  aisi install atlassian --global    # Install MCP globally`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVar(&installType, "type", "", "Asset type (rules, skills, agents, hooks, mcp)")
	installCmd.Flags().BoolVar(&installAll, "all", false, "Install all assets of the specified type")
	installCmd.Flags().BoolVar(&installGlobal, "global", false, "Install MCP to global config (home directory)")
	installCmd.Flags().StringVar(&installURL, "url", "", "Install asset from a URL (supports GitHub shorthand, tree paths, git URLs, local paths)")
	installCmd.Flags().StringVar(&installName, "name", "", "Custom name for the installed asset (used with --url)")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Handle URL-based installation
	if installURL != "" {
		if installType == "" {
			return fmt.Errorf("--type is required when using --url (e.g., --type=skills)")
		}
		return installFromURL()
	}

	// If no arguments provided, install from lock file
	if len(args) == 0 && !installAll {
		return installFromLock()
	}

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	if !cfg.IsConfigured() {
		return fmt.Errorf("repository not configured. Run: aisi config set-repo <url>")
	}

	target, err := getTarget()
	if err != nil {
		return err
	}

	repoMgr, err := repo.NewManager(cfg)
	if err != nil {
		return err
	}

	if err := repoMgr.EnsureMainRepo(); err != nil {
		return fmt.Errorf("failed to fetch repository: %w", err)
	}

	manifestPath := repoMgr.GetManifestPath()
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Check CLI version compatibility
	if err := m.CheckCLIVersion(version.Version); err != nil {
		if versionErr, ok := err.(*manifest.VersionMismatchError); ok {
			return fmt.Errorf("CLI version %s is below minimum required %s. Please update aisi to continue.", versionErr.CurrentVersion, versionErr.RequiredVersion)
		}
		return err
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	inst := installer.New(repoMgr, target, projectRoot)
	track := tracker.New(projectRoot, target)

	commit, _ := repoMgr.GetCurrentCommit()
	repoURL := cfg.Repo.URL

	if installAll && installType != "" {
		return installAllOfType(inst, track, m, installType, repoURL, commit)
	}

	for _, name := range args {
		// Try to detect if the argument is a skill from an online repository (owner/repo format)
		if looksLikeSkillRepo(name) {
			// Attempt to install as skill from URL first
			if err := installSkillFromSource(inst, track, name); err == nil {
				continue // Success, move to next argument
			}
			// If skill install fails, fall through to try as local asset
		}

		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		}
	}

	return nil
}

// looksLikeSkillRepo checks if the name matches the pattern "owner/repo" or "owner/repo/skill-name"
// which indicates it might be a skill from an online repository.
func looksLikeSkillRepo(name string) bool {
	// Check for patterns like:
	// - "owner/repo" (exactly one or two slashes, no spaces)
	// - "github.com/owner/repo"
	// - "gitlab.com/owner/repo"

	parts := strings.Split(name, "/")

	// Remove empty parts (from leading/trailing slashes)
	cleanParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			cleanParts = append(cleanParts, p)
		}
	}

	// Valid patterns: owner/repo (2 parts) or owner/repo/skill (3+ parts)
	if len(cleanParts) >= 2 {
		// Check that parts look valid (no spaces, at least one character each)
		for _, p := range cleanParts {
			if p == "" || strings.Contains(p, " ") {
				return false
			}
		}
		return true
	}
	return false
}

// installSkillFromSource attempts to install a skill from an online repository.
// It parses the name as a skill URL and installs it.
// If multiple skills are found in the repository, it shows a selection menu.
func installSkillFromSource(inst *installer.Installer, track *tracker.Tracker, source string) error {
	// Try to parse as a skill URL (owner/repo or full URL)
	skillURL, err := repo.ParseSkillURL(source)
	if err != nil {
		return err
	}

	result, err := inst.InstallSkillFromURL(skillURL, "")
	if err != nil {
		// Check if this is a MultipleSkillsError - show selection menu
		if multiErr, ok := err.(*repo.MultipleSkillsError); ok {
			return selectAndInstallSkill(inst, track, skillURL, multiErr.FoundSkills)
		}
		return err
	}

	if result.Success {
		fmt.Printf("✓ Installed %s (skill) from %s\n", result.Name, source)
		// Record with source information - use the actual discovered path
		skillEntry := tracker.SkillEntry{
			Name:   result.Name,
			Source: skillURL.RepoURL,
			Path:   result.SourcePath,
		}
		_ = track.RecordSkillInstallOnly(skillEntry)
		return nil
	}

	return result.Error
}

// selectAndInstallSkill shows an interactive menu to select which skill to install
// when multiple skills are found in a repository.
func selectAndInstallSkill(inst *installer.Installer, track *tracker.Tracker, skillURL *repo.SkillURL, skills []*repo.DiscoveredSkill) error {
	fmt.Printf("\n📦 Found %d skills in %s:\n\n", len(skills), skillURL.RepoURL)

	for i, skill := range skills {
		// Try to get description from SKILL.md
		metadata, _ := repo.ParseSkillMD(skill.SKILLMdPath)
		desc := ""
		if metadata != nil {
			desc = metadata.GetDescription(60)
		}

		fmt.Printf("  %d. %s\n", i+1, skill.Name)
		if skill.Path != "" && skill.Path != "." {
			fmt.Printf("     Path: %s/\n", skill.Path)
		}
		if desc != "" {
			fmt.Printf("     %s\n", desc)
		}
		fmt.Println()
	}

	// Ask user to select
	fmt.Print("Select skill to install (1-" + fmt.Sprintf("%d", len(skills)) + ", or 'a' for all, or 'c' to cancel): ")
	
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		// If scan fails (e.g., EOF), default to first skill
		input = "1"
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "c" || input == "cancel" {
		return fmt.Errorf("installation cancelled")
	}

	if input == "a" || input == "all" {
		// Install all skills
		for _, skill := range skills {
			if err := installDiscoveredSkill(inst, track, skillURL, skill); err != nil {
				fmt.Printf("✗ Failed to install %s: %v\n", skill.Name, err)
			}
		}
		return nil
	}

	// Parse selection number
	var selection int
	if _, err := fmt.Sscanf(input, "%d", &selection); err != nil || selection < 1 || selection > len(skills) {
		return fmt.Errorf("invalid selection: %s", input)
	}

	selectedSkill := skills[selection-1]
	return installDiscoveredSkill(inst, track, skillURL, selectedSkill)
}

// installDiscoveredSkill installs a specific discovered skill
func installDiscoveredSkill(inst *installer.Installer, track *tracker.Tracker, skillURL *repo.SkillURL, skill *repo.DiscoveredSkill) error {
	// Create a modified skillURL with the specific path
	urlCopy := *skillURL
	urlCopy.Path = skill.Path

	result, err := inst.InstallSkillFromURL(&urlCopy, "")
	if err != nil {
		return err
	}

	if result.Success {
		fmt.Printf("✓ Installed %s (skill) from %s\n", result.Name, skillURL.RepoURL)
		// Record with source information - use the actual discovered path from result
		skillEntry := tracker.SkillEntry{
			Name:   result.Name,
			Source: skillURL.RepoURL,
			Path:   result.SourcePath,
		}
		_ = track.RecordSkillInstallOnly(skillEntry)
		return nil
	}

	return result.Error
}

func installFromLock() error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	target, err := getTarget()
	if err != nil {
		return err
	}

	track := tracker.New(projectRoot, target)
	lock, err := track.Load()
	if err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Check if lock file has any assets
	if len(lock.Assets.Rules) == 0 && len(lock.Assets.Skills) == 0 &&
		len(lock.Assets.Agents) == 0 && len(lock.Assets.Hooks) == 0 &&
		len(lock.Assets.MCP) == 0 && len(lock.Assets.AgentsMD) == 0 &&
		len(lock.Assets.External) == 0 {
		return fmt.Errorf("no assets found in lock file. Run 'aisi install <asset>' first, or use 'aisi install --type=<type> --all'")
	}

	// Determine which repo to use: lock file takes precedence
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	repoURL := cfg.Repo.URL
	if lock.RepoURL != "" {
		repoURL = lock.RepoURL
		fmt.Printf("📦 Using repository from lock file: %s\n", repoURL)
	} else if repoURL == "" {
		return fmt.Errorf("repository not configured and no repository found in lock file. Run: aisi config set-repo <url>")
	}

	// Create config with repo from lock file
	lockCfg := cfg
	if lock.RepoURL != "" {
		lockCfg.SetRepo(lock.RepoURL, "")
	}

	repoMgr, err := repo.NewManager(lockCfg)
	if err != nil {
		return err
	}

	if err := repoMgr.EnsureMainRepo(); err != nil {
		return fmt.Errorf("failed to fetch repository: %w", err)
	}

	// Show commit info if available
	if lock.RepoCommit != "" {
		fmt.Printf("📋 Lock file was created from commit: %s\n", lock.RepoCommit[:7])
	}

	manifestPath := repoMgr.GetManifestPath()
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Check CLI version compatibility
	if err := m.CheckCLIVersion(version.Version); err != nil {
		if versionErr, ok := err.(*manifest.VersionMismatchError); ok {
			return fmt.Errorf("CLI version %s is below minimum required %s. Please update aisi to continue.", versionErr.CurrentVersion, versionErr.RequiredVersion)
		}
		return err
	}

	inst := installer.New(repoMgr, target, projectRoot)
	commit, _ := repoMgr.GetCurrentCommit()

	fmt.Println("\nInstalling assets from lock file...")

	successCount := 0
	totalCount := 0

	// Install rules
	for _, name := range lock.Assets.Rules {
		totalCount++
		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		} else {
			successCount++
		}
	}

	// Install skills (handle both online and local sources)
	for _, skillEntry := range lock.Assets.Skills {
		totalCount++
		if err := installSkillFromEntry(inst, track, skillEntry, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", skillEntry.Name, err)
		} else {
			successCount++
		}
	}

	// Install agents
	for _, name := range lock.Assets.Agents {
		totalCount++
		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		} else {
			successCount++
		}
	}

	// Install hooks
	for _, name := range lock.Assets.Hooks {
		totalCount++
		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		} else {
			successCount++
		}
	}

	// Install MCP servers
	for _, name := range lock.Assets.MCP {
		totalCount++
		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		} else {
			successCount++
		}
	}

	// Install AgentsMD
	for _, name := range lock.Assets.AgentsMD {
		totalCount++
		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		} else {
			successCount++
		}
	}

	// Install external assets
	for _, name := range lock.Assets.External {
		totalCount++
		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("\n✓ Installed %d/%d assets from lock file\n", successCount, totalCount)
	return nil
}

// installSkillFromEntry installs a skill from a SkillEntry.
// If the skill has a Source (online repository), it installs from there.
// Otherwise, it installs from the local repository.
func installSkillFromEntry(inst *installer.Installer, track *tracker.Tracker, entry tracker.SkillEntry, repoURL, commit string) error {
	// If skill has a remote source, install from there
	if entry.Source != "" {
		skillURL, err := repo.ParseSkillURL(entry.Source)
		if err != nil {
			return fmt.Errorf("failed to parse skill source %q: %w", entry.Source, err)
		}

		// If a specific path is recorded, use it
		if entry.Path != "" {
			skillURL.Path = entry.Path
		}

		result, err := inst.InstallSkillFromURL(skillURL, entry.Name)
		if err != nil {
			return err
		}

		if result.Success {
			fmt.Printf("✓ Installed %s (%s) from %s\n", result.Name, result.Type, entry.Source)
			// Record with source information preserved
			_ = track.RecordSkillInstall(entry, repoURL, commit)
			return nil
		}
		return result.Error
	}

	// No source - try to install from local repository
	// This handles legacy skills or skills from the main project repo
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	repoMgr, err := repo.NewManager(cfg)
	if err != nil {
		return err
	}

	if err := repoMgr.EnsureMainRepo(); err != nil {
		return fmt.Errorf("failed to fetch repository: %w", err)
	}

	manifestPath := repoMgr.GetManifestPath()
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	result, err := inst.Install(m, entry.Name)
	if err != nil {
		return err
	}

	if result.Success {
		fmt.Printf("✓ Installed %s (%s)\n", result.Name, result.Type)
		_ = track.RecordInstall(result.Type, result.Name, repoURL, commit)
		return nil
	}
	return result.Error
}

func installFromURL() error {
	if installType != "skills" {
		return fmt.Errorf("--url currently only supported for --type=skills")
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	target, err := getTarget()
	if err != nil {
		return err
	}

	// Parse the URL
	skillURL, err := repo.ParseSkillURL(installURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	var repoMgr *repo.Manager

	// Only need repo manager for git URLs (not local paths)
	if !skillURL.IsLocal {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		repoMgr, err = repo.NewManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create repository manager: %w", err)
		}
	}

	inst := installer.New(repoMgr, target, projectRoot)
	track := tracker.New(projectRoot, target)

	// Install the skill from URL
	result, err := inst.InstallSkillFromURL(skillURL, installName)
	if err != nil {
		return err
	}

	if result.Success {
		fmt.Printf("✓ Installed %s (%s) from %s\n", result.Name, result.Type, installURL)
		// Track the installation with full source information
		// Don't modify project repoURL/repoCommit when installing from external source
		skillEntry := tracker.SkillEntry{
			Name:   result.Name,
			Source: skillURL.RepoURL,
			Path:   skillURL.Path,
		}
		_ = track.RecordSkillInstallOnly(skillEntry)
	} else {
		return result.Error
	}

	return nil
}

func installAsset(inst *installer.Installer, track *tracker.Tracker, m *manifest.Manifest, name, repoURL, commit string) error {
	// Check if it's an external asset with requirements
	if ext := m.GetExternal(name); ext != nil && len(ext.Requirements) > 0 {
		fmt.Printf("\n⚠️  %s has prerequisites:\n", name)
		for _, req := range ext.Requirements {
			fmt.Printf("   • %s\n", req)
		}
		fmt.Println()
	}

	// Check if it's an MCP that needs env vars
	if mcp := m.GetMCP(name); mcp != nil && len(mcp.Env) > 0 {
		envVars, promptErr := prompt.PromptForEnvVars(mcp)
		if promptErr != nil {
			return fmt.Errorf("failed to get env vars: %v", promptErr)
		}

		var result *installer.InstallResult
		var err error

		if installGlobal {
			result, err = inst.InstallMCPGlobal(mcp, envVars)
		} else {
			result, err = inst.InstallMCP(mcp, envVars)
		}

		if err != nil {
			return err
		}

		if result.Success {
			fmt.Printf("✓ Installed %s (%s)\n", result.Name, result.Type)
			_ = track.RecordInstall(result.Type, result.Name, repoURL, commit)
		} else {
			return result.Error
		}
		return nil
	}

	result, err := inst.Install(m, name)
	if err != nil {
		return err
	}

	if result.Success {
		fmt.Printf("✓ Installed %s (%s)\n", result.Name, result.Type)
		_ = track.RecordInstall(result.Type, result.Name, repoURL, commit)
	} else {
		return result.Error
	}

	return nil
}

func installAllOfType(inst *installer.Installer, track *tracker.Tracker, m *manifest.Manifest, assetType, repoURL, commit string) error {
	var results []*installer.InstallResult
	var err error

	switch assetType {
	case "rules":
		results, err = inst.InstallAllRules(m.Rules)
	case "skills":
		results, err = inst.InstallAllSkills(m.Skills)
	case "agents":
		results, err = inst.InstallAllAgents(m.Agents)
	default:
		return fmt.Errorf("unknown asset type: %s", assetType)
	}

	if err != nil {
		return err
	}

	for _, result := range results {
		if result.Success {
			fmt.Printf("✓ Installed %s (%s)\n", result.Name, result.Type)
			_ = track.RecordInstall(result.Type, result.Name, repoURL, commit)
		} else {
			fmt.Printf("✗ %s: %v\n", result.Name, result.Error)
		}
	}

	return nil
}
