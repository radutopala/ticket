package cmd

import (
	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Set ticket status to closed",
	Long:  `Set the ticket status to closed. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateTicketStatus(cmd, args[0], tk.StatusClosed)
	},
}
