package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/radutopala/ticket/internal/domain"
)

type SearchSuite struct {
	suite.Suite
}

func TestSearchSuite(t *testing.T) {
	suite.Run(t, new(SearchSuite))
}

func (s *SearchSuite) TestSearchTickets() {
	now := time.Now()
	tickets := []*domain.Ticket{
		{ID: "t1", Status: domain.StatusOpen, Title: "Fix authentication bug", Description: "Users cannot login", Created: now},
		{ID: "t2", Status: domain.StatusInProgress, Title: "Add new feature", Description: "Implement authentication flow", Created: now},
		{ID: "t3", Status: domain.StatusClosed, Title: "Update documentation", Description: "Add API docs", Created: now},
		{ID: "t4", Status: domain.StatusOpen, Title: "Refactor code", Description: "Clean up the auth module", Created: now},
	}

	tests := []struct {
		name          string
		query         string
		caseSensitive bool
		statusFilter  string
		wantIDs       []string
	}{
		{
			name:    "search in title case insensitive",
			query:   "authentication",
			wantIDs: []string{"t1", "t2"},
		},
		{
			name:    "search in description",
			query:   "API",
			wantIDs: []string{"t3"},
		},
		{
			name:          "case sensitive no match",
			query:         "Authentication",
			caseSensitive: true,
			wantIDs:       nil,
		},
		{
			name:          "case sensitive match",
			query:         "Fix",
			caseSensitive: true,
			wantIDs:       []string{"t1"},
		},
		{
			name:         "filter by status open",
			query:        "auth",
			statusFilter: "open",
			wantIDs:      []string{"t1", "t4"},
		},
		{
			name:         "filter by status with no matches",
			query:        "documentation",
			statusFilter: "open",
			wantIDs:      nil,
		},
		{
			name:    "no matches",
			query:   "nonexistent",
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			matches := searchTickets(tickets, tt.query, tt.caseSensitive, tt.statusFilter)

			var ids []string
			for _, m := range matches {
				ids = append(ids, m.ticket.ID)
			}

			require.Equal(s.T(), tt.wantIDs, ids)
		})
	}
}

func (s *SearchSuite) TestSearchTicketsContextExtraction() {
	now := time.Now()
	tickets := []*domain.Ticket{
		{
			ID:          "t1",
			Status:      domain.StatusOpen,
			Title:       "Simple title",
			Description: "This is a longer description that contains the search term somewhere in the middle of the text",
			Created:     now,
		},
	}

	matches := searchTickets(tickets, "search term", false, "")
	require.Len(s.T(), matches, 1)
	require.NotEmpty(s.T(), matches[0].context)
	require.Contains(s.T(), matches[0].context, "search term")
}

func (s *SearchSuite) TestExtractContext() {
	tests := []struct {
		name       string
		text       string
		matchIdx   int
		matchLen   int
		contextLen int
		wantLen    int // minimum expected length
	}{
		{
			name:       "middle of text",
			text:       "This is a test string with some words in it",
			matchIdx:   10,
			matchLen:   4,
			contextLen: 10,
			wantLen:    10,
		},
		{
			name:       "start of text",
			text:       "Test at start of text",
			matchIdx:   0,
			matchLen:   4,
			contextLen: 10,
			wantLen:    4,
		},
		{
			name:       "end of text",
			text:       "Some text at the end",
			matchIdx:   16,
			matchLen:   3,
			contextLen: 10,
			wantLen:    3,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := extractContext(tt.text, tt.matchIdx, tt.matchLen, tt.contextLen)
			require.GreaterOrEqual(s.T(), len(result), tt.wantLen)
		})
	}
}

func (s *SearchSuite) TestSortSearchMatchesByPriority() {
	matches := []searchMatch{
		{ticket: &domain.Ticket{ID: "t3", Priority: 3}},
		{ticket: &domain.Ticket{ID: "t1", Priority: 1}},
		{ticket: &domain.Ticket{ID: "t2", Priority: 2}},
		{ticket: &domain.Ticket{ID: "t0", Priority: 1}},
	}

	sortSearchMatchesByPriority(matches)

	var ids []string
	for _, m := range matches {
		ids = append(ids, m.ticket.ID)
	}

	require.Equal(s.T(), []string{"t0", "t1", "t2", "t3"}, ids)
}
