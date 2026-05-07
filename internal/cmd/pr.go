package cmd

import (
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr <id> [ref]",
	Short: "Set or clear a ticket's PR/MR reference",
	Long:  `Set the pull/merge request reference (e.g., gh-pr-42, !123, https://...) on an existing ticket. Omit the ref or pass an empty string to clear it. Supports partial ID matching.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetFieldCommand(cmd, args, "pr", store.SetPR)
	},
}
