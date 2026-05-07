package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

// outputJSON writes v as indented JSON to the command's output writer.
func outputJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// resolveAndReadTicket resolves a partial ID and reads the ticket.
// This is a common pattern used throughout the commands.
func resolveAndReadTicket(idArg string) (*tk.Ticket, error) {
	id, err := store.ResolveID(idArg)
	if err != nil {
		return nil, err
	}
	return store.Read(id)
}

// updateTicketStatus updates a ticket's status and prints a confirmation message.
func updateTicketStatus(cmd *cobra.Command, idArg string, newStatus tk.Status) error {
	ticket, err := resolveAndReadTicket(idArg)
	if err != nil {
		return err
	}

	ticket.Status = newStatus

	if err := store.Write(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	if jsonOutput {
		return outputJSON(cmd, ticket)
	}

	fmt.Printf("Updated %s -> %s\n", ticket.ID, newStatus)
	return nil
}

// formatTicketLine formats a ticket as a single-line summary.
func formatTicketLine(t *tk.Ticket) string {
	return fmt.Sprintf("%s [P%d][%s] - %s", t.ID, t.Priority, t.Status, t.Title)
}

// runSetFieldCommand resolves the ID, invokes setter, and prints the result.
// Used by simple `<cmd> <id> [ref]` commands that update a single string field.
func runSetFieldCommand(cmd *cobra.Command, args []string, label string, setter func(id, ref string) (*tk.Ticket, error)) error {
	id, err := store.ResolveID(args[0])
	if err != nil {
		return err
	}

	ref := ""
	if len(args) == 2 {
		ref = args[1]
	}

	ticket, err := setter(id, ref)
	if err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	if jsonOutput {
		return outputJSON(cmd, ticket)
	}

	if ref == "" {
		fmt.Printf("Cleared %s on %s\n", label, ticket.ID)
	} else {
		fmt.Printf("Updated %s %s -> %s\n", ticket.ID, label, ref)
	}
	return nil
}
