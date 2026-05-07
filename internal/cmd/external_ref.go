package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var externalRefCmd = &cobra.Command{
	Use:   "external-ref <id> [ref]",
	Short: "Set or clear a ticket's external reference",
	Long:  `Set the external reference (e.g., gh-123, JIRA-456) on an existing ticket. Omit the ref or pass an empty string to clear it. Supports partial ID matching.`,
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

		ticket, err := store.SetExternalRef(id, ref)
		if err != nil {
			return fmt.Errorf("failed to update ticket: %w", err)
		}

		if jsonOutput {
			return outputJSON(cmd, ticket)
		}

		if ref == "" {
			fmt.Printf("Cleared external-ref on %s\n", ticket.ID)
		} else {
			fmt.Printf("Updated %s external-ref -> %s\n", ticket.ID, ref)
		}
		return nil
	},
}
