package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/radutopala/ticket/internal/domain"
)

type DepSuite struct {
	suite.Suite
}

func TestDepSuite(t *testing.T) {
	suite.Run(t, new(DepSuite))
}

func (s *DepSuite) TestTopologicalSort() {
	tests := []struct {
		name      string
		tickets   []*domain.Ticket
		wantOrder []string
		wantErr   bool
	}{
		{
			name: "no dependencies",
			tickets: []*domain.Ticket{
				{ID: "a", Status: domain.StatusOpen},
				{ID: "b", Status: domain.StatusOpen},
				{ID: "c", Status: domain.StatusOpen},
			},
			wantOrder: nil, // Any order is valid when there are no dependencies
			wantErr:   false,
		},
		{
			name: "linear chain",
			tickets: []*domain.Ticket{
				{ID: "a", Status: domain.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: domain.StatusOpen, Deps: []string{"c"}},
				{ID: "c", Status: domain.StatusOpen},
			},
			wantOrder: []string{"c", "b", "a"},
			wantErr:   false,
		},
		{
			name: "diamond dependency",
			tickets: []*domain.Ticket{
				{ID: "a", Status: domain.StatusOpen, Deps: []string{"b", "c"}},
				{ID: "b", Status: domain.StatusOpen, Deps: []string{"d"}},
				{ID: "c", Status: domain.StatusOpen, Deps: []string{"d"}},
				{ID: "d", Status: domain.StatusOpen},
			},
			wantErr: false,
		},
		{
			name: "cycle",
			tickets: []*domain.Ticket{
				{ID: "a", Status: domain.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: domain.StatusOpen, Deps: []string{"c"}},
				{ID: "c", Status: domain.StatusOpen, Deps: []string{"a"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			sorted, err := TopologicalSort(tt.tickets)
			if tt.wantErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Len(s.T(), sorted, len(tt.tickets))

			if tt.wantOrder != nil {
				ids := make([]string, len(sorted))
				for i, t := range sorted {
					ids[i] = t.ID
				}
				require.Equal(s.T(), tt.wantOrder, ids)
			}

			// Verify topological property: for each ticket, all its deps appear before it
			idxMap := make(map[string]int)
			for i, t := range sorted {
				idxMap[t.ID] = i
			}

			for _, t := range sorted {
				for _, dep := range t.Deps {
					depIdx, ok := idxMap[dep]
					if ok {
						require.Less(s.T(), depIdx, idxMap[t.ID],
							"dependency %s should come before %s", dep, t.ID)
					}
				}
			}
		})
	}
}

func (s *DepSuite) TestDetectCycles() {
	tests := []struct {
		name       string
		tickets    []*domain.Ticket
		wantCycles int
	}{
		{
			name: "no cycles",
			tickets: []*domain.Ticket{
				{ID: "a", Status: domain.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: domain.StatusOpen, Deps: []string{"c"}},
				{ID: "c", Status: domain.StatusOpen},
			},
			wantCycles: 0,
		},
		{
			name: "simple cycle",
			tickets: []*domain.Ticket{
				{ID: "a", Status: domain.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: domain.StatusOpen, Deps: []string{"a"}},
			},
			wantCycles: 1,
		},
		{
			name: "three node cycle",
			tickets: []*domain.Ticket{
				{ID: "a", Status: domain.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: domain.StatusOpen, Deps: []string{"c"}},
				{ID: "c", Status: domain.StatusOpen, Deps: []string{"a"}},
			},
			wantCycles: 1,
		},
		{
			name: "self loop",
			tickets: []*domain.Ticket{
				{ID: "a", Status: domain.StatusOpen, Deps: []string{"a"}},
			},
			wantCycles: 1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cycles := DetectCycles(tt.tickets)
			require.Len(s.T(), cycles, tt.wantCycles)
		})
	}
}

func (s *DepSuite) TestStatusIndicator() {
	require.Equal(s.T(), "[ ]", statusIndicator(domain.StatusOpen))
	require.Equal(s.T(), "[~]", statusIndicator(domain.StatusInProgress))
	require.Equal(s.T(), "[x]", statusIndicator(domain.StatusClosed))
	require.Equal(s.T(), "[?]", statusIndicator(domain.Status("unknown")))
}

func (s *DepSuite) TestFindRootTickets() {
	now := time.Now()
	tickets := []*domain.Ticket{
		{ID: "root1", Status: domain.StatusOpen, Created: now},
		{ID: "root2", Status: domain.StatusOpen, Created: now},
		{ID: "child", Status: domain.StatusOpen, Deps: []string{"root1"}, Created: now},
		{ID: "closed", Status: domain.StatusClosed, Created: now},
	}

	ticketMap := make(map[string]*domain.Ticket)
	for _, t := range tickets {
		ticketMap[t.ID] = t
	}

	roots := findRootTickets(tickets, ticketMap)

	// root2 and child are roots (child depends on root1, making root1 not a root)
	// closed tickets are excluded
	rootIDs := make(map[string]bool)
	for _, r := range roots {
		rootIDs[r.ID] = true
	}

	require.True(s.T(), rootIDs["root2"], "root2 should be a root")
	require.True(s.T(), rootIDs["child"], "child should be a root (it's not a dependency)")
	require.False(s.T(), rootIDs["root1"], "root1 should not be a root (it's a dependency)")
	require.False(s.T(), rootIDs["closed"], "closed ticket should not be a root")
}
