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
	Type     string
}

// SortOptions holds sorting options for list commands.
type SortOptions struct {
	SortBy  string
	Reverse bool
}

// validSortFields lists valid sort field names.
var validSortFields = []string{"priority", "created", "status", "title"}

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
	if f.Type != "" && string(t.Type) != f.Type {
		return false
	}
	return true
}

var listFlags FilterOptions
var sortFlags SortOptions

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

		filtered := filterTickets(tickets, listFlags)
		sortTickets(filtered, sortFlags)

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
		return listByDependencyStatus(false)
	},
}

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "List open/in_progress tickets with unresolved deps",
	Long: `List open or in_progress tickets that have unresolved dependencies.

Sort options: priority (default), created, status, title`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listByDependencyStatus(true)
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

		var closed []*domain.Ticket
		for _, t := range tickets {
			if t.Status != domain.StatusClosed {
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
		sortTickets(closed, opts)

		// Limit results
		if closedFlags.limit > 0 && len(closed) > closedFlags.limit {
			closed = closed[:closedFlags.limit]
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

func sortTickets(tickets []*domain.Ticket, opts SortOptions) {
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "priority"
	}

	sort.Slice(tickets, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "created":
			less = tickets[i].Created.Before(tickets[j].Created)
		case "status":
			less = string(tickets[i].Status) < string(tickets[j].Status)
		case "title":
			less = strings.ToLower(tickets[i].Title) < strings.ToLower(tickets[j].Title)
		default: // priority
			if tickets[i].Priority != tickets[j].Priority {
				less = tickets[i].Priority < tickets[j].Priority
			} else {
				less = tickets[i].ID < tickets[j].ID
			}
		}

		if opts.Reverse {
			return !less
		}
		return less
	})
}

// listByDependencyStatus lists tickets filtered by their dependency status.
// If wantBlocked is true, it lists tickets with unresolved dependencies (blocked).
// If wantBlocked is false, it lists tickets with no unresolved dependencies (ready).
func listByDependencyStatus(wantBlocked bool) error {
	tickets, err := store.List()
	if err != nil {
		return err
	}

	openIDs := buildOpenIDSet(tickets)

	var result []*domain.Ticket
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

		if hasBlockingDeps == wantBlocked && listFlags.Matches(t) {
			result = append(result, t)
		}
	}

	sortTickets(result, sortFlags)

	return runWithPager(func(w io.Writer) error {
		for _, t := range result {
			if _, err := fmt.Fprintln(w, formatTicketLine(t)); err != nil {
				return err
			}
		}
		return nil
	})
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
