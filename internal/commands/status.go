package commands

import (
	"fmt"
	"os"

	"github.com/rosseca/aisi/internal/tracker"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installed assets status",
	Long:  `Display the status of installed assets in the current project.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	target, err := getTarget()
	if err != nil {
		return err
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	track := tracker.New(projectRoot, target)
	installed, err := track.GetInstalled()
	if err != nil {
		return err
	}

	commit, _ := track.GetRepoCommit()

	fmt.Printf("\nInstalled Assets (Target: %s)\n", target.DisplayName)
	if commit != "" {
		fmt.Printf("Repository commit: %s\n", commit[:7])
	}
	fmt.Println()

	hasAny := false

	if len(installed.Rules) > 0 {
		hasAny = true
		fmt.Println("RULES")
		for _, r := range installed.Rules {
			fmt.Printf("  ✓ %s\n", r)
		}
	}

	if len(installed.Skills) > 0 {
		hasAny = true
		fmt.Println("SKILLS")
		for _, s := range installed.Skills {
			fmt.Printf("  ✓ %s\n", s)
		}
	}

	if len(installed.Agents) > 0 {
		hasAny = true
		fmt.Println("AGENTS")
		for _, a := range installed.Agents {
			fmt.Printf("  ✓ %s\n", a)
		}
	}

	if len(installed.Hooks) > 0 {
		hasAny = true
		fmt.Println("HOOKS")
		for _, h := range installed.Hooks {
			fmt.Printf("  ✓ %s\n", h)
		}
	}

	if len(installed.MCP) > 0 {
		hasAny = true
		fmt.Println("MCP")
		for _, m := range installed.MCP {
			fmt.Printf("  ✓ %s\n", m)
		}
	}

	if len(installed.AgentsMD) > 0 {
		hasAny = true
		fmt.Println("AGENTS.MD")
		for _, am := range installed.AgentsMD {
			fmt.Printf("  ✓ %s\n", am)
		}
	}

	if len(installed.External) > 0 {
		hasAny = true
		fmt.Println("EXTERNAL")
		for _, e := range installed.External {
			fmt.Printf("  ✓ %s\n", e)
		}
	}

	if !hasAny {
		fmt.Println("No assets installed yet.")
		fmt.Println("Run 'aisi list' to see available assets.")
	}

	fmt.Println()
	return nil
}
