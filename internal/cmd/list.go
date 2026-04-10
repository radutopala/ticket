package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

var listFlags tk.FilterOptions
var sortFlags tk.SortOptions

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List tickets",
	Long: `List all tickets with optional filters for status, assignee, type, and tags.

Sort options: priority (default), created, status, title`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		filtered := tk.Filter(tickets, listFlags)
		tk.Sort(filtered, sortFlags)

		if jsonOutput {
			return outputJSON(cmd, filtered)
		}

		return runWithPager(func(w io.Writer) error {
			for _, t := range filtered {
				if _, err := fmt.Fprintln(w, formatTicketLine(t)); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List open/in_progress tickets with resolved deps",
	Long: `List open or in_progress tickets that have no unresolved dependencies.

Sort options: priority (default), created, status, title`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		result := tk.FilterByDependencyStatus(tickets, false, listFlags)
		tk.Sort(result, sortFlags)

		if jsonOutput {
			return outputJSON(cmd, result)
		}

		return runWithPager(func(w io.Writer) error {
			for _, t := range result {
				if _, err := fmt.Fprintln(w, formatTicketLine(t)); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "List open/in_progress tickets with unresolved deps",
	Long: `List open or in_progress tickets that have unresolved dependencies.

Sort options: priority (default), created, status, title`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		result := tk.FilterByDependencyStatus(tickets, true, listFlags)
		tk.Sort(result, sortFlags)

		if jsonOutput {
			return outputJSON(cmd, result)
		}

		return runWithPager(func(w io.Writer) error {
			for _, t := range result {
				if _, err := fmt.Fprintln(w, formatTicketLine(t)); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

var closedFlags struct {
	limit int
}

var closedCmd = &cobra.Command{
	Use:   "closed",
	Short: "List recently closed tickets",
	Long: `List recently closed tickets.

Sort options: priority, created (default, descending), status, title`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		var closed []*tk.Ticket
		for _, t := range tickets {
			if t.Status != tk.StatusClosed {
				continue
			}
			if listFlags.Matches(t) {
				closed = append(closed, t)
			}
		}

		// Use sortFlags if set, otherwise default to created descending
		opts := sortFlags
		if opts.SortBy == "" {
			opts.SortBy = "created"
			opts.Reverse = true
		}
		tk.Sort(closed, opts)

		// Limit results
		if closedFlags.limit > 0 && len(closed) > closedFlags.limit {
			closed = closed[:closedFlags.limit]
		}

		if jsonOutput {
			return outputJSON(cmd, closed)
		}

		return runWithPager(func(w io.Writer) error {
			for _, t := range closed {
				if _, err := fmt.Fprintln(w, formatTicketLine(t)); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func init() {
	listCmd.Flags().StringVar(&listFlags.Status, "status", "", "Filter by status (open|in_progress|closed)")
	listCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "Filter by assignee")
	listCmd.Flags().StringVarP(&listFlags.Tag, "tag", "T", "", "Filter by tag")
	listCmd.Flags().StringVarP(&listFlags.Type, "type", "t", "", "Filter by type (task|bug|feature|epic|chore)")
	listCmd.Flags().StringVarP(&sortFlags.SortBy, "sort", "s", "", "Sort by field (priority|created|status|title)")
	listCmd.Flags().BoolVarP(&sortFlags.Reverse, "reverse", "r", false, "Reverse sort order")

	readyCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "Filter by assignee")
	readyCmd.Flags().StringVarP(&listFlags.Tag, "tag", "T", "", "Filter by tag")
	readyCmd.Flags().StringVarP(&listFlags.Type, "type", "t", "", "Filter by type (task|bug|feature|epic|chore)")
	readyCmd.Flags().StringVarP(&sortFlags.SortBy, "sort", "s", "", "Sort by field (priority|created|status|title)")
	readyCmd.Flags().BoolVarP(&sortFlags.Reverse, "reverse", "r", false, "Reverse sort order")

	blockedCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "Filter by assignee")
	blockedCmd.Flags().StringVarP(&listFlags.Tag, "tag", "T", "", "Filter by tag")
	blockedCmd.Flags().StringVarP(&listFlags.Type, "type", "t", "", "Filter by type (task|bug|feature|epic|chore)")
	blockedCmd.Flags().StringVarP(&sortFlags.SortBy, "sort", "s", "", "Sort by field (priority|created|status|title)")
	blockedCmd.Flags().BoolVarP(&sortFlags.Reverse, "reverse", "r", false, "Reverse sort order")

	closedCmd.Flags().IntVar(&closedFlags.limit, "limit", 20, "Limit number of results")
	closedCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "Filter by assignee")
	closedCmd.Flags().StringVarP(&listFlags.Tag, "tag", "T", "", "Filter by tag")
	closedCmd.Flags().StringVarP(&listFlags.Type, "type", "t", "", "Filter by type (task|bug|feature|epic|chore)")
	closedCmd.Flags().StringVarP(&sortFlags.SortBy, "sort", "s", "", "Sort by field (priority|created|status|title)")
	closedCmd.Flags().BoolVarP(&sortFlags.Reverse, "reverse", "r", false, "Reverse sort order")
}
