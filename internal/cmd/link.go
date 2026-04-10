package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
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

		if err := tk.ValidateLinkIDs(ids); err != nil {
			return err
		}

		// Add links to all tickets
		for _, id := range ids {
			ticket, err := store.Read(id)
			if err != nil {
				return err
			}

			for _, otherID := range ids {
				if otherID == id {
					continue
				}
				tk.AddLink(ticket, otherID)
			}

			if err := store.Write(ticket); err != nil {
				return err
			}
		}

		if jsonOutput {
			// Re-read all linked tickets for output
			linked := make([]*tk.Ticket, 0, len(ids))
			for _, id := range ids {
				t, err := store.Read(id)
				if err != nil {
					return err
				}
				linked = append(linked, t)
			}
			return outputJSON(cmd, linked)
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

		ticket1, err := store.Read(id1)
		if err != nil {
			return err
		}

		found1 := tk.RemoveLink(ticket1, id2)

		ticket2, err := store.Read(id2)
		if err != nil {
			return err
		}

		found2 := tk.RemoveLink(ticket2, id1)

		if !found1 && !found2 {
			return fmt.Errorf("no link found between %s and %s", id1, id2)
		}

		if err := store.Write(ticket1); err != nil {
			return err
		}
		if err := store.Write(ticket2); err != nil {
			return err
		}

		if jsonOutput {
			// Re-read both tickets for output
			t1, err := store.Read(id1)
			if err != nil {
				return err
			}
			t2, err := store.Read(id2)
			if err != nil {
				return err
			}
			return outputJSON(cmd, []*tk.Ticket{t1, t2})
		}

		fmt.Printf("Unlinked: %s and %s\n", id1, id2)
		return nil
	},
}
