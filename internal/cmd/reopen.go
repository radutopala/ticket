package cmd

import (
	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

var reopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Set ticket status to open",
	Long:  `Set the ticket status back to open. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateTicketStatus(cmd, args[0], tk.StatusOpen)
	},
}
