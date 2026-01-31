package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

// FilterOptions holds common filtering options for list commands.
type FilterOptions struct {
	Status   string
	Assignee string
	Tag      string
}

// Matches checks if a ticket matches the filter options.
func (f FilterOptions) Matches(t *domain.Ticket) bool {
	if f.Status != "" && string(t.Status) != f.Status {
		return false
	}
	if f.Assignee != "" && t.Assignee != f.Assignee {
		return false
	}
	if f.Tag != "" && !hasTag(t.Tags, f.Tag) {
		return false
	}
	return true
}

var listFlags FilterOptions

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List tickets",
	Long:    `List all tickets with optional filters for status, assignee, and tags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		filtered := filterTickets(tickets, listFlags)
		sortTicketsByPriority(filtered)

		return runWithPager(func(w io.Writer) error {
			for _, t := range filtered {
				if _, err := fmt.Fprintf(w, "%s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title); err != nil {
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
	Long:  `List open or in_progress tickets that have no unresolved dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		// Build a set of open ticket IDs
		openIDs := make(map[string]bool)
		for _, t := range tickets {
			if t.Status != domain.StatusClosed {
				openIDs[t.ID] = true
			}
		}

		var ready []*domain.Ticket
		for _, t := range tickets {
			if t.Status == domain.StatusClosed {
				continue
			}

			// Check if all dependencies are resolved (closed)
			hasBlockingDeps := false
			for _, dep := range t.Deps {
				if openIDs[dep] {
					hasBlockingDeps = true
					break
				}
			}

			if !hasBlockingDeps && listFlags.Matches(t) {
				ready = append(ready, t)
			}
		}

		sortTicketsByPriority(ready)

		return runWithPager(func(w io.Writer) error {
			for _, t := range ready {
				if _, err := fmt.Fprintf(w, "%s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title); err != nil {
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
	Long:  `List open or in_progress tickets that have unresolved dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		// Build a set of open ticket IDs
		openIDs := make(map[string]bool)
		for _, t := range tickets {
			if t.Status != domain.StatusClosed {
				openIDs[t.ID] = true
			}
		}

		var blocked []*domain.Ticket
		for _, t := range tickets {
			if t.Status == domain.StatusClosed {
				continue
			}

			// Check if any dependency is unresolved (open)
			hasBlockingDeps := false
			for _, dep := range t.Deps {
				if openIDs[dep] {
					hasBlockingDeps = true
					break
				}
			}

			if hasBlockingDeps && listFlags.Matches(t) {
				blocked = append(blocked, t)
			}
		}

		sortTicketsByPriority(blocked)

		return runWithPager(func(w io.Writer) error {
			for _, t := range blocked {
				if _, err := fmt.Fprintf(w, "%s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title); err != nil {
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
	Long:  `List recently closed tickets, sorted by modification time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		var closed []*domain.Ticket
		for _, t := range tickets {
			if t.Status != domain.StatusClosed {
				continue
			}
			if listFlags.Matches(t) {
				closed = append(closed, t)
			}
		}

		// Sort by created date (descending - newest first)
		sort.Slice(closed, func(i, j int) bool {
			return closed[i].Created.After(closed[j].Created)
		})

		// Limit results
		if closedFlags.limit > 0 && len(closed) > closedFlags.limit {
			closed = closed[:closedFlags.limit]
		}

		return runWithPager(func(w io.Writer) error {
			for _, t := range closed {
				if _, err := fmt.Fprintf(w, "%s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func filterTickets(tickets []*domain.Ticket, opts FilterOptions) []*domain.Ticket {
	var result []*domain.Ticket
	for _, t := range tickets {
		if opts.Matches(t) {
			result = append(result, t)
		}
	}
	return result
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func sortTicketsByPriority(tickets []*domain.Ticket) {
	sort.Slice(tickets, func(i, j int) bool {
		if tickets[i].Priority != tickets[j].Priority {
			return tickets[i].Priority < tickets[j].Priority
		}
		return tickets[i].ID < tickets[j].ID
	})
}

func init() {
	listCmd.Flags().StringVar(&listFlags.Status, "status", "", "Filter by status (open|in_progress|closed)")
	listCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "Filter by assignee")
	listCmd.Flags().StringVarP(&listFlags.Tag, "tag", "T", "", "Filter by tag")

	readyCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "Filter by assignee")
	readyCmd.Flags().StringVarP(&listFlags.Tag, "tag", "T", "", "Filter by tag")

	blockedCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "Filter by assignee")
	blockedCmd.Flags().StringVarP(&listFlags.Tag, "tag", "T", "", "Filter by tag")

	closedCmd.Flags().IntVar(&closedFlags.limit, "limit", 20, "Limit number of results")
	closedCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "Filter by assignee")
	closedCmd.Flags().StringVarP(&listFlags.Tag, "tag", "T", "", "Filter by tag")
}
