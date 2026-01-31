package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/radutopala/ticket/internal/domain"
)

type ListSuite struct {
	suite.Suite
}

func TestListSuite(t *testing.T) {
	suite.Run(t, new(ListSuite))
}

func (s *ListSuite) TestHasTag() {
	tests := []struct {
		name     string
		tags     []string
		tag      string
		expected bool
	}{
		{
			name:     "tag exists exact match",
			tags:     []string{"backend", "api", "urgent"},
			tag:      "api",
			expected: true,
		},
		{
			name:     "tag exists case insensitive",
			tags:     []string{"Backend", "API", "Urgent"},
			tag:      "api",
			expected: true,
		},
		{
			name:     "tag not found",
			tags:     []string{"backend", "api", "urgent"},
			tag:      "frontend",
			expected: false,
		},
		{
			name:     "empty tags",
			tags:     []string{},
			tag:      "api",
			expected: false,
		},
		{
			name:     "nil tags",
			tags:     nil,
			tag:      "api",
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := hasTag(tt.tags, tt.tag)
			require.Equal(s.T(), tt.expected, result)
		})
	}
}

func (s *ListSuite) TestFilterTickets() {
	now := time.Now()
	tickets := []*domain.Ticket{
		{ID: "t1", Status: domain.StatusOpen, Assignee: "alice", Tags: []string{"backend"}, Created: now},
		{ID: "t2", Status: domain.StatusInProgress, Assignee: "bob", Tags: []string{"frontend"}, Created: now},
		{ID: "t3", Status: domain.StatusClosed, Assignee: "alice", Tags: []string{"backend", "urgent"}, Created: now},
		{ID: "t4", Status: domain.StatusOpen, Assignee: "charlie", Tags: []string{"api"}, Created: now},
	}

	tests := []struct {
		name      string
		status    string
		assignee  string
		tag       string
		wantIDs   []string
	}{
		{
			name:    "no filters",
			wantIDs: []string{"t1", "t2", "t3", "t4"},
		},
		{
			name:    "filter by status open",
			status:  "open",
			wantIDs: []string{"t1", "t4"},
		},
		{
			name:    "filter by status in_progress",
			status:  "in_progress",
			wantIDs: []string{"t2"},
		},
		{
			name:     "filter by assignee",
			assignee: "alice",
			wantIDs:  []string{"t1", "t3"},
		},
		{
			name:    "filter by tag",
			tag:     "backend",
			wantIDs: []string{"t1", "t3"},
		},
		{
			name:     "filter by status and assignee",
			status:   "open",
			assignee: "alice",
			wantIDs:  []string{"t1"},
		},
		{
			name:     "filter by assignee and tag",
			assignee: "alice",
			tag:      "urgent",
			wantIDs:  []string{"t3"},
		},
		{
			name:    "no matches",
			status:  "open",
			tag:     "nonexistent",
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := filterTickets(tickets, tt.status, tt.assignee, tt.tag)

			var ids []string
			for _, t := range result {
				ids = append(ids, t.ID)
			}

			require.Equal(s.T(), tt.wantIDs, ids)
		})
	}
}

func (s *ListSuite) TestSortTicketsByPriority() {
	tests := []struct {
		name    string
		tickets []*domain.Ticket
		wantIDs []string
	}{
		{
			name: "sort by priority ascending",
			tickets: []*domain.Ticket{
				{ID: "t3", Priority: 3},
				{ID: "t1", Priority: 1},
				{ID: "t2", Priority: 2},
			},
			wantIDs: []string{"t1", "t2", "t3"},
		},
		{
			name: "same priority sort by ID",
			tickets: []*domain.Ticket{
				{ID: "c", Priority: 1},
				{ID: "a", Priority: 1},
				{ID: "b", Priority: 1},
			},
			wantIDs: []string{"a", "b", "c"},
		},
		{
			name: "mixed priority and ID",
			tickets: []*domain.Ticket{
				{ID: "t2", Priority: 2},
				{ID: "t3", Priority: 1},
				{ID: "t1", Priority: 2},
				{ID: "t4", Priority: 0},
			},
			wantIDs: []string{"t4", "t3", "t1", "t2"},
		},
		{
			name:    "empty list",
			tickets: []*domain.Ticket{},
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Make a copy to avoid mutating original
			tickets := make([]*domain.Ticket, len(tt.tickets))
			copy(tickets, tt.tickets)

			sortTicketsByPriority(tickets)

			var ids []string
			for _, t := range tickets {
				ids = append(ids, t.ID)
			}

			require.Equal(s.T(), tt.wantIDs, ids)
		})
	}
}
