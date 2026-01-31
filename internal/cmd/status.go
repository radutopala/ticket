package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

var statusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "Update ticket status",
	Long:  `Update the ticket status. Valid statuses: open, in_progress, closed. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := store.ResolveID(args[0])
		if err != nil {
			return err
		}

		newStatus, err := domain.ParseStatus(args[1])
		if err != nil {
			return err
		}

		ticket, err := store.Read(id)
		if err != nil {
			return err
		}

		ticket.Status = newStatus

		if err := store.Write(ticket); err != nil {
			return fmt.Errorf("failed to update ticket: %w", err)
		}

		fmt.Printf("Updated %s -> %s\n", id, newStatus)
		return nil
	},
}
