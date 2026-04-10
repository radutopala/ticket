package cmd

import (
	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

var statusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "Update ticket status",
	Long:  `Update the ticket status. Valid statuses: open, in_progress, closed. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		newStatus, err := tk.ParseStatus(args[1])
		if err != nil {
			return err
		}
		return updateTicketStatus(cmd, args[0], newStatus)
	},
}
