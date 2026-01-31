package cmd

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/radutopala/ticket/internal/domain"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Display a ticket",
	Long:  `Display the full contents of a ticket by ID. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := store.ResolveID(args[0])
		if err != nil {
			return err
		}

		ticket, err := store.Read(id)
		if err != nil {
			return err
		}

		// Render the ticket content
		content, err := ticket.Render()
		if err != nil {
			return fmt.Errorf("failed to render ticket: %w", err)
		}

		// Add parent comment if present
		output := string(content)
		if ticket.Parent != "" {
			// Find where to insert parent comment (after links line in frontmatter)
			lines := strings.Split(output, "\n")
			var result []string
			for i, line := range lines {
				result = append(result, line)
				if strings.HasPrefix(line, "links:") && i > 0 {
					// Try to get parent title
					parentTitle := ""
					if parentTicket, err := store.Read(ticket.Parent); err == nil {
						parentTitle = parentTicket.Title
					}
					if parentTitle != "" {
						result = append(result, fmt.Sprintf("parent: %s  # %s", ticket.Parent, parentTitle))
					} else {
						result = append(result, fmt.Sprintf("parent: %s", ticket.Parent))
					}
				}
			}
			output = strings.Join(result, "\n")
		}

		// Get relationships
		relationships, err := getTicketRelationships(id, ticket)
		if err != nil {
			return fmt.Errorf("failed to get relationships: %w", err)
		}

		return runWithPager(func(w io.Writer) error {
			if _, err := fmt.Fprint(w, output); err != nil {
				return err
			}
			if relationships != "" {
				if _, err := fmt.Fprintln(w, "---"); err != nil {
					return err
				}
				if _, err := fmt.Fprint(w, relationships); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

// getTicketRelationships returns a string with the ticket's relationships.
func getTicketRelationships(id string, ticket *domain.Ticket) (string, error) {
	allTickets, err := store.List()
	if err != nil {
		return "", err
	}

	var blocking []string
	var children []string

	for _, t := range allTickets {
		if t.ID == id {
			continue
		}

		// Check if this ticket depends on us (we are blocking it)
		if slices.Contains(t.Deps, id) {
			blocking = append(blocking, t.ID)
		}

		// Check if this ticket is a child of us
		if t.Parent == id {
			children = append(children, t.ID)
		}
	}

	var lines []string

	// Blockers (tickets this one depends on)
	if len(ticket.Deps) > 0 {
		lines = append(lines, fmt.Sprintf("Blockers: %s", strings.Join(ticket.Deps, ", ")))
	}

	// Blocking (tickets that depend on this one)
	if len(blocking) > 0 {
		lines = append(lines, fmt.Sprintf("Blocking: %s", strings.Join(blocking, ", ")))
	}

	// Children (tickets with this ticket as parent)
	if len(children) > 0 {
		lines = append(lines, fmt.Sprintf("Children: %s", strings.Join(children, ", ")))
	}

	// Links (bidirectionally linked tickets)
	if len(ticket.Links) > 0 {
		lines = append(lines, fmt.Sprintf("Links: %s", strings.Join(ticket.Links, ", ")))
	}

	if len(lines) == 0 {
		return "", nil
	}

	return strings.Join(lines, "\n") + "\n", nil
}
