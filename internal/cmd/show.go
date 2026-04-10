package cmd

import (
	"fmt"
	"io"
	"strings"

	tk "github.com/radutopala/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Display a ticket",
	Long:  `Display the full contents of a ticket by ID. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticket, err := resolveAndReadTicket(args[0])
		if err != nil {
			return err
		}

		// Load all tickets once for parent lookup and relationships
		allTickets, err := store.List()
		if err != nil {
			return fmt.Errorf("failed to list tickets: %w", err)
		}

		// Build ticket map for O(1) lookups
		ticketMap := make(map[string]*tk.Ticket)
		for _, t := range allTickets {
			ticketMap[t.ID] = t
		}

		if jsonOutput {
			return outputJSON(cmd, ticket)
		}

		// Render the ticket content
		content, err := ticket.Render()
		if err != nil {
			return fmt.Errorf("failed to render ticket: %w", err)
		}

		// Add parent comment if present
		output := string(content)
		if ticket.Parent != "" {
			lines := strings.Split(output, "\n")
			var result []string
			for i, line := range lines {
				result = append(result, line)
				if strings.HasPrefix(line, "links:") && i > 0 {
					parentTitle := ""
					if parentTicket, ok := ticketMap[ticket.Parent]; ok {
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
		rel := tk.GetRelationships(ticket.ID, ticket, allTickets)
		relationships := formatRelationships(rel)

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

// formatRelationships formats relationships as a display string.
func formatRelationships(rel tk.Relationships) string {
	var lines []string

	if len(rel.Blockers) > 0 {
		lines = append(lines, fmt.Sprintf("Blockers: %s", strings.Join(rel.Blockers, ", ")))
	}
	if len(rel.Blocking) > 0 {
		lines = append(lines, fmt.Sprintf("Blocking: %s", strings.Join(rel.Blocking, ", ")))
	}
	if len(rel.Children) > 0 {
		lines = append(lines, fmt.Sprintf("Children: %s", strings.Join(rel.Children, ", ")))
	}
	if len(rel.Links) > 0 {
		lines = append(lines, fmt.Sprintf("Links: %s", strings.Join(rel.Links, ", ")))
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n"
}
