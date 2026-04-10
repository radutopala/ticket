package ticket

import (
	"sort"
	"strings"
)

// FilterOptions holds common filtering options for ticket queries.
type FilterOptions struct {
	Status   string
	Assignee string
	Tag      string
	Type     string
}

// SortOptions holds sorting options for ticket queries.
type SortOptions struct {
	SortBy  string
	Reverse bool
}

// ValidSortFields lists valid sort field names.
var ValidSortFields = []string{"priority", "created", "status", "title"}

// Matches checks if a ticket matches the filter options.
func (f FilterOptions) Matches(t *Ticket) bool {
	if f.Status != "" && string(t.Status) != f.Status {
		return false
	}
	if f.Assignee != "" && t.Assignee != f.Assignee {
		return false
	}
	if f.Tag != "" && !HasTag(t.Tags, f.Tag) {
		return false
	}
	if f.Type != "" && string(t.Type) != f.Type {
		return false
	}
	return true
}

// Filter returns tickets matching the given filter options.
func Filter(tickets []*Ticket, opts FilterOptions) []*Ticket {
	var result []*Ticket
	for _, t := range tickets {
		if opts.Matches(t) {
			result = append(result, t)
		}
	}
	return result
}

// Sort sorts tickets in place by the given options.
func Sort(tickets []*Ticket, opts SortOptions) {
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "priority"
	}

	sort.Slice(tickets, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "created":
			less = tickets[i].Created.Before(tickets[j].Created)
		case "status":
			less = string(tickets[i].Status) < string(tickets[j].Status)
		case "title":
			less = strings.ToLower(tickets[i].Title) < strings.ToLower(tickets[j].Title)
		default: // priority
			if tickets[i].Priority != tickets[j].Priority {
				less = tickets[i].Priority < tickets[j].Priority
			} else {
				less = tickets[i].ID < tickets[j].ID
			}
		}

		if opts.Reverse {
			return !less
		}
		return less
	})
}

// FilterByDependencyStatus filters non-closed tickets by their dependency status.
// If wantBlocked is true, returns tickets with unresolved (non-closed) dependencies.
// If wantBlocked is false, returns tickets with no unresolved dependencies (ready).
func FilterByDependencyStatus(tickets []*Ticket, wantBlocked bool, filter FilterOptions) []*Ticket {
	openIDs := BuildOpenIDSet(tickets)

	var result []*Ticket
	for _, t := range tickets {
		if t.Status == StatusClosed {
			continue
		}

		hasBlockingDeps := false
		for _, dep := range t.Deps {
			if openIDs[dep] {
				hasBlockingDeps = true
				break
			}
		}

		if hasBlockingDeps == wantBlocked && filter.Matches(t) {
			result = append(result, t)
		}
	}

	return result
}

// HasTag checks if a tag list contains a tag (case-insensitive).
func HasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}
