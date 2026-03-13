package commands

import (
	"fmt"

	"github.com/rosseca/aisi/internal/manifest"
	"github.com/rosseca/aisi/internal/repo"
	"github.com/rosseca/aisi/internal/version"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [type]",
	Short: "List available assets from the shared repository",
	Long: `List all available assets or filter by type.

Examples:
  aisi list         # List all assets
  aisi list rules   # List only rules
  aisi list skills  # List only skills`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	if !cfg.IsConfigured() {
		return fmt.Errorf("repository not configured. Run: aisi config set-repo <url>")
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

	filterType := ""
	if len(args) > 0 {
		filterType = args[0]
	}

	if filterType == "" || filterType == "rules" {
		if len(m.Rules) > 0 {
			fmt.Println("\nRULES")
			for _, r := range m.Rules {
				badge := "[local]   "
				fmt.Printf("  %s %s - %s\n", badge, r.Name, r.Description)
			}
		}
	}

	if filterType == "" || filterType == "skills" {
		if len(m.Skills) > 0 || len(m.AllExternalSkills()) > 0 {
			fmt.Println("\nSKILLS")
			for _, s := range m.Skills {
				badge := "[local]   "
				fmt.Printf("  %s %s - %s\n", badge, s.Name, s.Description)
			}
			for _, e := range m.AllExternalSkills() {
				badge := "[external]"
				fmt.Printf("  %s %s - %s (%s)\n", badge, e.Name, e.Description, e.Repo)
				if len(e.Requirements) > 0 {
					fmt.Printf("             ⚠️  Requires: %s\n", e.Requirements[0])
				}
			}
		}
	}

	if filterType == "" || filterType == "agents" {
		if len(m.Agents) > 0 || len(m.AllExternalAgents()) > 0 {
			fmt.Println("\nAGENTS")
			for _, a := range m.Agents {
				badge := "[local]   "
				fmt.Printf("  %s %s - %s\n", badge, a.Name, a.Description)
			}
			for _, e := range m.AllExternalAgents() {
				badge := "[external]"
				fmt.Printf("  %s %s - %s (%s)\n", badge, e.Name, e.Description, e.Repo)
			}
		}
	}

	if filterType == "" || filterType == "hooks" {
		if len(m.Hooks) > 0 {
			fmt.Println("\nHOOKS")
			for _, h := range m.Hooks {
				badge := "[local]   "
				fmt.Printf("  %s %s - %s\n", badge, h.Name, h.Description)
			}
		}
	}

	if filterType == "" || filterType == "mcp" {
		if len(m.MCP) > 0 {
			fmt.Println("\nMCP")
			for _, mc := range m.MCP {
				badge := "[local]   "
				fmt.Printf("  %s %s - %s\n", badge, mc.Name, mc.Description)
			}
		}
	}

	if filterType == "" || filterType == "agents-md" || filterType == "agentsmd" {
		if len(m.AgentsMD) > 0 {
			fmt.Println("\nAGENTS.MD")
			for _, am := range m.AgentsMD {
				badge := "[local]   "
				fmt.Printf("  %s %s - %s\n", badge, am.Name, am.Description)
			}
		}
	}

	fmt.Println()
	return nil
}
