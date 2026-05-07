package cmd

import (
	"github.com/spf13/cobra"
)

var externalRefCmd = &cobra.Command{
	Use:   "external-ref <id> [ref]",
	Short: "Set or clear a ticket's external reference",
	Long:  `Set the external reference (e.g., gh-123, JIRA-456) on an existing ticket. Omit the ref or pass an empty string to clear it. Supports partial ID matching.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetFieldCommand(cmd, args, "external-ref", store.SetExternalRef)
	},
}
