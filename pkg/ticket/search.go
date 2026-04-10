package ticket

import (
	"sort"
	"strings"
)

// SearchMatch represents a search result with optional context.
type SearchMatch struct {
	Ticket  *Ticket
	Context string
}

// Search searches tickets by text in title and description.
// Results are sorted by priority (ascending).
func Search(tickets []*Ticket, query string, caseSensitive bool, statusFilter string) []SearchMatch {
	var matches []SearchMatch

	searchQuery := query
	if !caseSensitive {
		searchQuery = strings.ToLower(query)
	}

	for _, t := range tickets {
		if statusFilter != "" && string(t.Status) != statusFilter {
			continue
		}

		title := t.Title
		description := t.Description

		if !caseSensitive {
			title = strings.ToLower(title)
			description = strings.ToLower(description)
		}

		// Check title
		if strings.Contains(title, searchQuery) {
			matches = append(matches, SearchMatch{Ticket: t, Context: ""})
			continue
		}

		// Check description
		if idx := strings.Index(description, searchQuery); idx != -1 {
			context := ExtractContext(t.Description, idx, len(query), 40)
			matches = append(matches, SearchMatch{Ticket: t, Context: context})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Ticket.Priority != matches[j].Ticket.Priority {
			return matches[i].Ticket.Priority < matches[j].Ticket.Priority
		}
		return matches[i].Ticket.ID < matches[j].Ticket.ID
	})

	return matches
}

// ExtractContext extracts a word-boundary-aware context string around a match position.
func ExtractContext(text string, matchIdx, matchLen, contextLen int) string {
	start := max(matchIdx-contextLen, 0)
	end := min(matchIdx+matchLen+contextLen, len(text))

	// Find word boundaries
	for start > 0 && text[start] != ' ' && text[start] != '\n' {
		start--
	}
	for end < len(text) && text[end] != ' ' && text[end] != '\n' {
		end++
	}

	context := strings.TrimSpace(text[start:end])
	context = strings.ReplaceAll(context, "\n", " ")

	return context
}
