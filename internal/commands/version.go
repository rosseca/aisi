package commands

import (
	"fmt"
	"runtime"

	"github.com/rosseca/aisi/internal/version"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("aisi %s\n", version.Version)
		fmt.Printf("  commit:  %s\n", version.Commit)
		fmt.Printf("  built:   %s\n", version.Date)
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}
