package commands

import (
	"fmt"
	"os"

	"github.com/rosseca/aisi/internal/config"
	"github.com/rosseca/aisi/internal/targets"
	"github.com/rosseca/aisi/internal/tui"
	"github.com/spf13/cobra"
)

var (
	targetFlag string
	repoFlag   string
	rootCmd    = &cobra.Command{
		Use:   "aisi",
		Short: "AI Shared Intelligence - AI Agent Assets Manager",
		Long: `AI Shared Intelligence (aisi) is a CLI tool to manage and install AI agent assets
(rules, skills, subagents, hooks, MCP configs) across different AI coding assistants.

Supports: Cursor, Kilo Code, Junie, and custom targets.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveMode()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&targetFlag, "target", "", "Target AI agent (cursor, kilo, junie)")
	rootCmd.PersistentFlags().StringVar(&repoFlag, "repo", "", "Repository URL or local path (overrides config)")
}

func Execute() error {
	return rootCmd.Execute()
}

func runInteractiveMode() error {
	cfg, configExists, err := getConfigWithExists()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	targetName := targetFlag
	if targetName == "" {
		targetName = cfg.ActiveTarget
	}
	if targetName == "" {
		targetName = "cursor"
	}

	target, err := targets.Get(targetName)
	if err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Show repo info if using --repo flag
	if repoFlag != "" {
		fmt.Printf("📦 Using repository: %s\n\n", repoFlag)
	}

	return tui.Run(cfg, target, projectRoot, configExists)
}

func getTarget() (*targets.Target, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	targetName := targetFlag
	if targetName == "" {
		targetName = cfg.ActiveTarget
	}
	if targetName == "" {
		targetName = "cursor"
	}

	return targets.Get(targetName)
}

func getConfig() (*config.Config, error) {
	cfg, _, err := getConfigWithExists()
	return cfg, err
}

func getConfigWithExists() (*config.Config, bool, error) {
	cfg, exists, err := config.LoadWithExists()
	if err != nil {
		return nil, false, err
	}

	// Override repo URL if --repo flag is provided
	if repoFlag != "" {
		cfg.SetRepo(repoFlag, "")
	}

	return cfg, exists, nil
}
