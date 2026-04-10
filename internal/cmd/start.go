package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Set ticket status to in_progress",
	Long:  `Set the ticket status to in_progress. Supports partial ID matching. Uses file locking to prevent race conditions.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := store.ResolveID(args[0])
		if err != nil {
			return err
		}

		ticket, err := store.AtomicClaim(id)
		if err != nil {
			if errors.Is(err, tk.ErrAlreadyClaimed) {
				return fmt.Errorf("cannot claim %s: %w", id, err)
			}
			return fmt.Errorf("failed to claim ticket: %w", err)
		}

		if jsonOutput {
			return outputJSON(cmd, ticket)
		}

		fmt.Printf("Claimed %s -> in_progress\n", ticket.ID)
		return nil
	},
}
