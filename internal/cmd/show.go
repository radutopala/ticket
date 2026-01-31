package cmd

import (
	"fmt"
	"strings"

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

		fmt.Print(output)
		return nil
	},
}
