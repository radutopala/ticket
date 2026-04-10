package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

var searchFlags struct {
	caseSensitive bool
	status        string
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tickets by text",
	Long: `Search for tickets containing the specified text in title or description.

The search is case-insensitive by default.

Examples:
  tk search 'authentication'           # Search for "authentication"
  tk search 'bug fix' --case-sensitive # Case-sensitive search
  tk search 'TODO' --status=open       # Search only open tickets`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		tickets, err := store.List()
		if err != nil {
			return err
		}

		matches := tk.Search(tickets, query, searchFlags.caseSensitive, searchFlags.status)

		if jsonOutput {
			return outputJSON(cmd, matches)
		}

		return runWithPager(func(w io.Writer) error {
			for _, m := range matches {
				if _, err := fmt.Fprintln(w, formatTicketLine(m.Ticket)); err != nil {
					return err
				}
				if m.Context != "" {
					if _, err := fmt.Fprintf(w, "  ...%s...\n", m.Context); err != nil {
						return err
					}
				}
			}
			return nil
		})
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchFlags.caseSensitive, "case-sensitive", false, "Perform case-sensitive search")
	searchCmd.Flags().StringVar(&searchFlags.status, "status", "", "Filter by status (open|in_progress|closed)")
}
