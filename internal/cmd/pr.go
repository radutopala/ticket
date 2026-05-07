package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr <id> [ref]",
	Short: "Set or clear a ticket's PR/MR reference",
	Long:  `Set the pull/merge request reference (e.g., gh-pr-42, !123, https://...) on an existing ticket. Omit the ref or pass an empty string to clear it. Supports partial ID matching.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := store.ResolveID(args[0])
		if err != nil {
			return err
		}

		ref := ""
		if len(args) == 2 {
			ref = args[1]
		}

		ticket, err := store.SetPR(id, ref)
		if err != nil {
			return fmt.Errorf("failed to update ticket: %w", err)
		}

		if jsonOutput {
			return outputJSON(cmd, ticket)
		}

		if ref == "" {
			fmt.Printf("Cleared pr on %s\n", ticket.ID)
		} else {
			fmt.Printf("Updated %s pr -> %s\n", ticket.ID, ref)
		}
		return nil
	},
}
