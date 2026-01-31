package cmd

import (
	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

var statusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "Update ticket status",
	Long:  `Update the ticket status. Valid statuses: open, in_progress, closed. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		newStatus, err := domain.ParseStatus(args[1])
		if err != nil {
			return err
		}
		return updateTicketStatus(args[0], newStatus)
	},
}
