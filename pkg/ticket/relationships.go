package ticket

import (
	"fmt"
	"slices"
)

// Relationships holds the discovered relationships for a ticket.
type Relationships struct {
	Blockers []string // tickets this one depends on (same as Deps)
	Blocking []string // tickets that depend on this one
	Children []string // tickets with this ticket as parent
	Links    []string // bidirectionally linked tickets
}

// GetRelationships discovers all relationships for a ticket by scanning all tickets.
func GetRelationships(id string, ticket *Ticket, allTickets []*Ticket) Relationships {
	rel := Relationships{
		Blockers: ticket.Deps,
		Links:    ticket.Links,
	}

	for _, t := range allTickets {
		if t.ID == id {
			continue
		}

		if slices.Contains(t.Deps, id) {
			rel.Blocking = append(rel.Blocking, t.ID)
		}

		if t.Parent == id {
			rel.Children = append(rel.Children, t.ID)
		}
	}

	return rel
}

// ValidateLinkIDs checks that a set of IDs to be linked has no duplicates.
func ValidateLinkIDs(ids []string) error {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return fmt.Errorf("duplicate ticket ID: %s", id)
		}
		seen[id] = true
	}
	return nil
}

// AddLink adds otherID to ticket's links if not already present.
// Returns true if the link was added, false if it already existed.
func AddLink(ticket *Ticket, otherID string) bool {
	if slices.Contains(ticket.Links, otherID) {
		return false
	}
	ticket.Links = append(ticket.Links, otherID)
	return true
}

// RemoveLink removes otherID from ticket's links.
// Returns the updated links slice and whether the link was found.
func RemoveLink(ticket *Ticket, otherID string) bool {
	newLinks, found := RemoveFromSlice(ticket.Links, otherID)
	ticket.Links = newLinks
	return found
}
