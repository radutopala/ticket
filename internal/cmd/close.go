package cmd

import (
	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Set ticket status to closed",
	Long:  `Set the ticket status to closed. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateTicketStatus(args[0], domain.StatusClosed)
	},
}
