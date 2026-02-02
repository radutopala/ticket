package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

var bulkFlags struct {
	tag      string
	status   string
	assignee string
	dryRun   bool
}

var bulkCmd = &cobra.Command{
	Use:   "bulk <action>",
	Short: "Perform bulk operations on multiple tickets",
	Long: `Perform bulk status updates on multiple tickets at once.
Filter tickets by tag, status, or assignee and apply an action to all matching tickets.

Actions:
  close   - Set matching tickets to closed status
  reopen  - Set matching tickets to open status
  start   - Set matching tickets to in_progress status

Examples:
  tk bulk close --tag=sprint-1           # Close all tickets with tag sprint-1
  tk bulk start --assignee=alice         # Start all tickets assigned to alice
  tk bulk reopen --status=closed         # Reopen all closed tickets
  tk bulk close --tag=bug --dry-run      # Preview what would be closed`,
}

var bulkCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Close multiple tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBulkAction(domain.StatusClosed, "closed")
	},
}

var bulkReopenCmd = &cobra.Command{
	Use:   "reopen",
	Short: "Reopen multiple tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBulkAction(domain.StatusOpen, "reopened")
	},
}

var bulkStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start multiple tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBulkAction(domain.StatusInProgress, "started")
	},
}

func runBulkAction(newStatus domain.Status, actionVerb string) error {
	tickets, err := store.List()
	if err != nil {
		return err
	}

	// Build filter options
	filterOpts := FilterOptions{
		Status:   bulkFlags.status,
		Assignee: bulkFlags.assignee,
		Tag:      bulkFlags.tag,
	}

	// Filter tickets
	filtered := filterTickets(tickets, filterOpts)

	if len(filtered) == 0 {
		fmt.Println("No tickets match the specified filters")
		return nil
	}

	if bulkFlags.dryRun {
		fmt.Printf("Dry run: would %s %d ticket(s):\n", actionVerb, len(filtered))
		for _, t := range filtered {
			fmt.Printf("  %s [%s] - %s\n", t.ID, t.Status, t.Title)
		}
		return nil
	}

	var updated int
	for _, t := range filtered {
		if t.Status == newStatus {
			continue // Skip tickets already in target status
		}
		t.Status = newStatus
		if err := store.Write(t); err != nil {
			return fmt.Errorf("failed to update %s: %w", t.ID, err)
		}
		updated++
		fmt.Printf("%s %s\n", actionVerb, t.ID)
	}

	if updated == 0 {
		fmt.Printf("No tickets needed updating (all already %s)\n", newStatus)
	} else {
		fmt.Printf("Successfully %s %d ticket(s)\n", actionVerb, updated)
	}

	return nil
}

func init() {
	// Add flags to parent bulk command (inherited by subcommands)
	bulkCmd.PersistentFlags().StringVarP(&bulkFlags.tag, "tag", "T", "", "Filter by tag")
	bulkCmd.PersistentFlags().StringVar(&bulkFlags.status, "status", "", "Filter by status (open|in_progress|closed)")
	bulkCmd.PersistentFlags().StringVarP(&bulkFlags.assignee, "assignee", "a", "", "Filter by assignee")
	bulkCmd.PersistentFlags().BoolVar(&bulkFlags.dryRun, "dry-run", false, "Preview changes without applying them")

	// Add subcommands
	bulkCmd.AddCommand(bulkCloseCmd)
	bulkCmd.AddCommand(bulkReopenCmd)
	bulkCmd.AddCommand(bulkStartCmd)
}
