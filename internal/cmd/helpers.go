package cmd

import (
	"fmt"

	"github.com/radutopala/ticket/internal/domain"
)

// updateTicketStatus updates a ticket's status and prints a confirmation message.
func updateTicketStatus(idArg string, newStatus domain.Status) error {
	id, err := store.ResolveID(idArg)
	if err != nil {
		return err
	}

	ticket, err := store.Read(id)
	if err != nil {
		return err
	}

	ticket.Status = newStatus

	if err := store.Write(ticket); err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}

	fmt.Printf("Updated %s -> %s\n", id, newStatus)
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
