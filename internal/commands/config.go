package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rosseca/aisi/internal/config"
	"github.com/rosseca/aisi/internal/targets"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `View and modify the MAU Intelligence CLI configuration.`,
}

var configSetRepoCmd = &cobra.Command{
	Use:   "set-repo <url>",
	Short: "Set the shared repository URL",
	Long: `Set the Git repository URL for shared assets.

Examples:
  aisi config set-repo git@github.com:company/mau-shared-agent-intelligence.git
  aisi config set-repo https://github.com/company/repo.git`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigSetRepo,
}

var configSetTargetCmd = &cobra.Command{
	Use:   "set-target <target>",
	Short: "Set the default target",
	Long: `Set the default AI agent target.

Available targets: cursor, kilo, junie, windsurf

Examples:
  aisi config set-target cursor
  aisi config set-target kilo`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigSetTarget,
}

var configSetTokenCmd = &cobra.Command{
	Use:   "set-token <token>",
	Short: "Set GitHub token for HTTPS access",
	Long: `Set a GitHub personal access token for HTTPS repository access.
This is used as a fallback when SSH access is not available.

Examples:
  aisi config set-token ghp_xxxxxxxxxxxxxxxxxxxx`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigSetToken,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration interactively",
	Long: `Interactive setup for MAU Intelligence CLI configuration.

This will guide you through:
  1. Setting the shared repository URL
  2. Choosing your default AI agent target
  3. Optional: Setting HTTPS token for private repos

Example:
  aisi config init`,
	RunE: runConfigInit,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetRepoCmd)
	configCmd.AddCommand(configSetTargetCmd)
	configCmd.AddCommand(configSetTokenCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
}

func runConfigSetRepo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.SetRepo(args[0], "")

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Printf("Repository URL set to: %s\n", args[0])
	return nil
}

func runConfigSetTarget(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	targetName := args[0]
	validTargets := map[string]bool{
		"cursor":   true,
		"kilo":     true,
		"junie":    true,
		"windsurf": true,
	}

	if !validTargets[targetName] {
		return fmt.Errorf("invalid target: %s. Valid targets: cursor, kilo, junie, windsurf", targetName)
	}

	cfg.SetActiveTarget(targetName)

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Printf("Default target set to: %s\n", targetName)
	return nil
}

func runConfigSetToken(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.SetHTTPSToken(args[0])

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Println("GitHub token saved.")
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Println("\nMAU Intelligence Configuration")
	fmt.Println("================================")

	if cfg.Repo.URL != "" {
		fmt.Printf("Repository URL:  %s\n", cfg.Repo.URL)
		fmt.Printf("Branch:          %s\n", cfg.Repo.Branch)
	} else {
		fmt.Println("Repository URL:  (not configured)")
	}

	fmt.Printf("Default Target:  %s\n", cfg.ActiveTarget)

	if cfg.HTTPSToken != "" {
		fmt.Println("HTTPS Token:     (configured)")
	} else {
		fmt.Println("HTTPS Token:     (not set)")
	}

	configDir, _ := config.ConfigDir()
	fmt.Printf("\nConfig location: %s\n", configDir)

	fmt.Println()
	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("🔧 MAU Intelligence - Initial Setup")
	fmt.Println("====================================")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Load existing config or create new
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Step 1: Repository URL
	fmt.Println("Step 1: Shared Repository")
	fmt.Println("---------------------------")
	fmt.Println("Enter your shared intelligence repository URL:")
	fmt.Println("  • SSH: git@github.com:your-company/shared-intelligence.git")
	fmt.Println("  • HTTPS: https://github.com/your-company/shared-intelligence")
	fmt.Println("  • Local: ./local-path (for development)")
	fmt.Println()

	currentRepo := cfg.Repo.URL
	if currentRepo != "" {
		fmt.Printf("Current: %s\n", currentRepo)
	}
	fmt.Print("Repository URL: ")

	repoURL, _ := reader.ReadString('\n')
	repoURL = strings.TrimSpace(repoURL)

	if repoURL != "" {
		cfg.SetRepo(repoURL, "")
		fmt.Println("✓ Repository configured")
	} else if currentRepo == "" {
		fmt.Println("⚠ No repository configured. You can set it later with: aisi config set-repo <url>")
	}
	fmt.Println()

	// Step 2: Default Target
	fmt.Println("Step 2: Default AI Agent Target")
	fmt.Println("--------------------------------")
	fmt.Println("Which AI agent will you use most often?")
	fmt.Println()
	fmt.Println("  [1] Cursor        (cursor, .cursor/)")
	fmt.Println("  [2] Kilo Code     (kilo, .kilocode/)")
	fmt.Println("  [3] Junie         (junie, .junie/)")
	fmt.Println("  [4] Windsurf      (windsurf, .windsurf/)")
	fmt.Println()

	currentTarget := cfg.ActiveTarget
	fmt.Printf("Current: %s\n", currentTarget)
	fmt.Print("Select [1-4 or press Enter to keep current]: ")

	targetInput, _ := reader.ReadString('\n')
	targetInput = strings.TrimSpace(targetInput)

	targetMap := map[string]string{
		"1": "cursor",
		"2": "kilo",
		"3": "junie",
		"4": "windsurf",
	}

	if targetInput != "" {
		if target, ok := targetMap[targetInput]; ok {
			cfg.SetActiveTarget(target)
		} else if targets.Names() != nil {
			// Check if they typed the name directly
			found := false
			for _, name := range targets.Names() {
				if targetInput == name {
					found = true
					cfg.SetActiveTarget(targetInput)
					break
				}
			}
			if !found {
				fmt.Printf("⚠ Unknown target '%s', using current: %s\n", targetInput, currentTarget)
			}
		}
	}

	target, _ := targets.Get(cfg.ActiveTarget)
	fmt.Printf("✓ Default target: %s\n\n", target.DisplayName)

	// Step 3: HTTPS Token (optional)
	fmt.Println("Step 3: HTTPS Token (Optional)")
	fmt.Println("--------------------------------")
	fmt.Println("If using a private repository or HTTPS instead of SSH,")
	fmt.Println("you may need a GitHub/GitLab personal access token.")
	fmt.Println()
	fmt.Println("Leave empty to skip (you can add it later with:")
	fmt.Println("  aisi config set-token <token>)")
	fmt.Println()

	if cfg.HTTPSToken != "" {
		fmt.Println("Current: (already configured)")
	}
	fmt.Print("HTTPS Token [press Enter to skip]: ")

	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token != "" {
		cfg.SetHTTPSToken(token)
		fmt.Println("✓ HTTPS token saved")
		fmt.Println()
	} else {
		fmt.Println("✓ Skipped (no token)")
		fmt.Println()
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Summary
	fmt.Println("✅ Configuration saved successfully!")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Printf("Repository:  %s\n", cfg.Repo.URL)
	fmt.Printf("Target:      %s\n", cfg.ActiveTarget)
	if cfg.HTTPSToken != "" {
		fmt.Println("Token:       (configured)")
	}
	fmt.Println()

	configDir, _ := config.ConfigDir()
	fmt.Printf("Config file: %s/config.yaml\n", configDir)
	fmt.Println()

	fmt.Println("Next steps:")
	fmt.Println("  • Run 'aisi list' to see available assets")
	fmt.Println("  • Run 'aisi' for interactive mode")

	return nil
}
