package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
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

		matches := searchTickets(tickets, query, searchFlags.caseSensitive, searchFlags.status)

		sortSearchMatchesByPriority(matches)

		return runWithPager(func(w io.Writer) error {
			for _, m := range matches {
				if _, err := fmt.Fprintln(w, formatTicketLine(m.ticket)); err != nil {
					return err
				}
				if m.context != "" {
					if _, err := fmt.Fprintf(w, "  ...%s...\n", m.context); err != nil {
						return err
					}
				}
			}
			return nil
		})
	},
}

type searchMatch struct {
	ticket  *domain.Ticket
	context string
}

func searchTickets(tickets []*domain.Ticket, query string, caseSensitive bool, statusFilter string) []searchMatch {
	var matches []searchMatch

	searchQuery := query
	if !caseSensitive {
		searchQuery = strings.ToLower(query)
	}

	for _, t := range tickets {
		// Apply status filter if specified
		if statusFilter != "" && string(t.Status) != statusFilter {
			continue
		}

		title := t.Title
		description := t.Description

		if !caseSensitive {
			title = strings.ToLower(title)
			description = strings.ToLower(description)
		}

		var context string

		// Check title
		if strings.Contains(title, searchQuery) {
			matches = append(matches, searchMatch{ticket: t, context: ""})
			continue
		}

		// Check description
		if idx := strings.Index(description, searchQuery); idx != -1 {
			context = extractContext(t.Description, idx, len(query), 40)
			matches = append(matches, searchMatch{ticket: t, context: context})
		}
	}

	return matches
}

func extractContext(text string, matchIdx, matchLen, contextLen int) string {
	start := matchIdx - contextLen
	if start < 0 {
		start = 0
	}

	end := matchIdx + matchLen + contextLen
	if end > len(text) {
		end = len(text)
	}

	// Find word boundaries
	for start > 0 && text[start] != ' ' && text[start] != '\n' {
		start--
	}
	for end < len(text) && text[end] != ' ' && text[end] != '\n' {
		end++
	}

	context := strings.TrimSpace(text[start:end])
	// Replace newlines with spaces for clean output
	context = strings.ReplaceAll(context, "\n", " ")

	return context
}

func sortSearchMatchesByPriority(matches []searchMatch) {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].ticket.Priority != matches[j].ticket.Priority {
			return matches[i].ticket.Priority < matches[j].ticket.Priority
		}
		return matches[i].ticket.ID < matches[j].ticket.ID
	})
}

func init() {
	searchCmd.Flags().BoolVar(&searchFlags.caseSensitive, "case-sensitive", false, "Perform case-sensitive search")
	searchCmd.Flags().StringVar(&searchFlags.status, "status", "", "Filter by status (open|in_progress|closed)")
}
