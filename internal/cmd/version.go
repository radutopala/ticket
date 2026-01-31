package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// SetVersion sets the version info from main package
func SetVersion(v, c, d string) {
	version = v
	commit = c
	date = d
}

// Version returns the current version
func Version() string {
	return version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tk %s\n", version)
		if commit != "none" {
			fmt.Printf("  commit: %s\n", commit)
		}
		if date != "unknown" {
			fmt.Printf("  built:  %s\n", date)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
