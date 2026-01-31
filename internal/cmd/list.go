package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

var listFlags struct {
	status   string
	assignee string
	tag      string
}

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

		filtered := filterTickets(tickets, listFlags.status, listFlags.assignee, listFlags.tag)
		sortTicketsByPriority(filtered)

		for _, t := range filtered {
			fmt.Printf("%s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title)
		}

		return nil
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

			if !hasBlockingDeps {
				if listFlags.assignee == "" || t.Assignee == listFlags.assignee {
					if listFlags.tag == "" || hasTag(t.Tags, listFlags.tag) {
						ready = append(ready, t)
					}
				}
			}
		}

		sortTicketsByPriority(ready)

		for _, t := range ready {
			fmt.Printf("%s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title)
		}

		return nil
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

			if hasBlockingDeps {
				if listFlags.assignee == "" || t.Assignee == listFlags.assignee {
					if listFlags.tag == "" || hasTag(t.Tags, listFlags.tag) {
						blocked = append(blocked, t)
					}
				}
			}
		}

		sortTicketsByPriority(blocked)

		for _, t := range blocked {
			fmt.Printf("%s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title)
		}

		return nil
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
			if listFlags.assignee == "" || t.Assignee == listFlags.assignee {
				if listFlags.tag == "" || hasTag(t.Tags, listFlags.tag) {
					closed = append(closed, t)
				}
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

		for _, t := range closed {
			fmt.Printf("%s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title)
		}

		return nil
	},
}

func filterTickets(tickets []*domain.Ticket, status, assignee, tag string) []*domain.Ticket {
	var result []*domain.Ticket
	for _, t := range tickets {
		if status != "" && string(t.Status) != status {
			continue
		}
		if assignee != "" && t.Assignee != assignee {
			continue
		}
		if tag != "" && !hasTag(t.Tags, tag) {
			continue
		}
		result = append(result, t)
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
	listCmd.Flags().StringVar(&listFlags.status, "status", "", "Filter by status (open|in_progress|closed)")
	listCmd.Flags().StringVarP(&listFlags.assignee, "assignee", "a", "", "Filter by assignee")
	listCmd.Flags().StringVarP(&listFlags.tag, "tag", "T", "", "Filter by tag")

	readyCmd.Flags().StringVarP(&listFlags.assignee, "assignee", "a", "", "Filter by assignee")
	readyCmd.Flags().StringVarP(&listFlags.tag, "tag", "T", "", "Filter by tag")

	blockedCmd.Flags().StringVarP(&listFlags.assignee, "assignee", "a", "", "Filter by assignee")
	blockedCmd.Flags().StringVarP(&listFlags.tag, "tag", "T", "", "Filter by tag")

	closedCmd.Flags().IntVar(&closedFlags.limit, "limit", 20, "Limit number of results")
	closedCmd.Flags().StringVarP(&listFlags.assignee, "assignee", "a", "", "Filter by assignee")
	closedCmd.Flags().StringVarP(&listFlags.tag, "tag", "T", "", "Filter by tag")
}
