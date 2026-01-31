package cmd

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <id> <id> [id...]",
	Short: "Link tickets together (symmetric)",
	Long:  `Link two or more tickets together. Links are bidirectional and will be added to all specified tickets.`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve all IDs first
		ids := make([]string, len(args))
		for i, arg := range args {
			id, err := store.ResolveID(arg)
			if err != nil {
				return fmt.Errorf("failed to resolve %s: %w", arg, err)
			}
			ids[i] = id
		}

		// Check for duplicates
		seen := make(map[string]bool)
		for _, id := range ids {
			if seen[id] {
				return fmt.Errorf("duplicate ticket ID: %s", id)
			}
			seen[id] = true
		}

		// Add links to all tickets
		for _, id := range ids {
			ticket, err := store.Read(id)
			if err != nil {
				return err
			}

			// Add all other IDs as links
			for _, otherID := range ids {
				if otherID == id {
					continue
				}
				if !slices.Contains(ticket.Links, otherID) {
					ticket.Links = append(ticket.Links, otherID)
				}
			}

			if err := store.Write(ticket); err != nil {
				return err
			}
		}

		fmt.Printf("Linked: %v\n", ids)
		return nil
	},
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <id> <target-id>",
	Short: "Remove link between tickets",
	Long:  `Remove a bidirectional link between two tickets.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id1, err := store.ResolveID(args[0])
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", args[0], err)
		}

		id2, err := store.ResolveID(args[1])
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", args[1], err)
		}

		// Remove link from first ticket
		ticket1, err := store.Read(id1)
		if err != nil {
			return err
		}

		found1 := false
		newLinks1 := make([]string, 0, len(ticket1.Links))
		for _, link := range ticket1.Links {
			if link == id2 {
				found1 = true
			} else {
				newLinks1 = append(newLinks1, link)
			}
		}
		ticket1.Links = newLinks1

		// Remove link from second ticket
		ticket2, err := store.Read(id2)
		if err != nil {
			return err
		}

		found2 := false
		newLinks2 := make([]string, 0, len(ticket2.Links))
		for _, link := range ticket2.Links {
			if link == id1 {
				found2 = true
			} else {
				newLinks2 = append(newLinks2, link)
			}
		}
		ticket2.Links = newLinks2

		if !found1 && !found2 {
			return fmt.Errorf("no link found between %s and %s", id1, id2)
		}

		if err := store.Write(ticket1); err != nil {
			return err
		}
		if err := store.Write(ticket2); err != nil {
			return err
		}

		fmt.Printf("Unlinked: %s and %s\n", id1, id2)
		return nil
	},
}
