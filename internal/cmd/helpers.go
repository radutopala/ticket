package cmd

import (
	"fmt"

	"github.com/radutopala/ticket/internal/domain"
)

// resolveAndReadTicket resolves a partial ID and reads the ticket.
// This is a common pattern used throughout the commands.
func resolveAndReadTicket(idArg string) (*domain.Ticket, error) {
	id, err := store.ResolveID(idArg)
	if err != nil {
		return nil, err
	}
	return store.Read(id)
}

// updateTicketStatus updates a ticket's status and prints a confirmation message.
func updateTicketStatus(idArg string, newStatus domain.Status) error {
	ticket, err := resolveAndReadTicket(idArg)
	if err != nil {
		return err
	}

	ticket.Status = newStatus

	if err := store.Write(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	fmt.Printf("Updated %s -> %s\n", ticket.ID, newStatus)
	return nil
}

// removeFromSlice removes the first occurrence of value from slice.
// Returns the new slice and a boolean indicating if the value was found.
func removeFromSlice(slice []string, value string) ([]string, bool) {
	found := false
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item == value {
			found = true
		} else {
			result = append(result, item)
		}
	}
	return result, found
}

// buildOpenIDSet builds a set of IDs for all non-closed tickets.
func buildOpenIDSet(tickets []*domain.Ticket) map[string]bool {
	openIDs := make(map[string]bool)
	for _, t := range tickets {
		if t.Status != domain.StatusClosed {
			openIDs[t.ID] = true
		}
	}
	return openIDs
}

// formatTicketLine formats a ticket as a single-line summary.
func formatTicketLine(t *domain.Ticket) string {
	return fmt.Sprintf("%s [P%d][%s] - %s", t.ID, t.Priority, t.Status, t.Title)
}
