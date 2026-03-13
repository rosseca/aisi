package commands

import (
	"fmt"
	"os"

	"github.com/rosseca/aisi/internal/installer"
	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/repo"
	"github.com/rosseca/aisi/internal/tracker"
	"github.com/rosseca/aisi/internal/version"
	"github.com/spf13/cobra"
)

var updateExternal bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update all installed assets to latest versions",
	Long: `Pull latest changes from the shared repository and reinstall 
all previously installed assets.

Examples:
  aisi update             # Update installed assets
  aisi update --external  # Also update external repos`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateExternal, "external", false, "Also update external repositories")
}

func runUpdate(cmd *cobra.Command, args []string) error {
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

	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	repoMgr, err := repo.NewManager(cfg)
	if err != nil {
		return err
	}

	fmt.Println("Updating repository...")
	if err := repoMgr.UpdateMainRepo(); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
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

	track := tracker.New(projectRoot, target)
	installed, err := track.GetInstalled()
	if err != nil {
		return err
	}

	inst := installer.New(repoMgr, target, projectRoot)
	commit, _ := repoMgr.GetCurrentCommit()

	updated := 0

	for _, name := range installed.Rules {
		if rule := m.GetRule(name); rule != nil {
			result, err := inst.InstallRule(rule)
			if err == nil && result.Success {
				fmt.Printf("✓ Updated rule: %s\n", name)
				updated++
			}
		}
	}

	for _, name := range installed.Skills {
		if skill := m.GetSkill(name); skill != nil {
			result, err := inst.InstallSkill(skill)
			if err == nil && result.Success {
				fmt.Printf("✓ Updated skill: %s\n", name)
				updated++
			}
		}
	}

	for _, name := range installed.Agents {
		if agent := m.GetAgent(name); agent != nil {
			result, err := inst.InstallAgent(agent)
			if err == nil && result.Success {
				fmt.Printf("✓ Updated agent: %s\n", name)
				updated++
			}
		}
	}

	for _, name := range installed.Hooks {
		if hook := m.GetHook(name); hook != nil {
			result, err := inst.InstallHook(hook)
			if err == nil && result.Success {
				fmt.Printf("✓ Updated hook: %s\n", name)
				updated++
			}
		}
	}

	lock, _ := track.Load()
	lock.RepoCommit = commit
	_ = track.Save(lock)

	if updated == 0 {
		fmt.Println("No assets to update.")
	} else {
		fmt.Printf("\nUpdated %d asset(s) to commit %s\n", updated, commit[:7])
	}

	return nil
}
