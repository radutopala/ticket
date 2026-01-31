package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Set ticket status to closed",
	Long:  `Set the ticket status to closed. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := store.ResolveID(args[0])
		if err != nil {
			return err
		}

		ticket, err := store.Read(id)
		if err != nil {
			return err
		}

		ticket.Status = domain.StatusClosed

		if err := store.Write(ticket); err != nil {
			return fmt.Errorf("failed to update ticket: %w", err)
		}

		fmt.Printf("Updated %s -> closed\n", id)
		return nil
	},
}
