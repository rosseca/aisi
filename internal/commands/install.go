package commands

import (
	"fmt"
	"os"

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
}

func runInstall(cmd *cobra.Command, args []string) error {
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
		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		}
	}

	return nil
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

	// Install all assets from lock file
	allAssets := []string{}
	allAssets = append(allAssets, lock.Assets.Rules...)
	allAssets = append(allAssets, lock.Assets.Skills...)
	allAssets = append(allAssets, lock.Assets.Agents...)
	allAssets = append(allAssets, lock.Assets.Hooks...)
	allAssets = append(allAssets, lock.Assets.MCP...)
	allAssets = append(allAssets, lock.Assets.AgentsMD...)
	allAssets = append(allAssets, lock.Assets.External...)

	successCount := 0
	for _, name := range allAssets {
		if err := installAsset(inst, track, m, name, repoURL, commit); err != nil {
			fmt.Printf("✗ %s: %v\n", name, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("\n✓ Installed %d/%d assets from lock file\n", successCount, len(allAssets))
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
