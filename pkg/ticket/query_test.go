package ticket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type QuerySuite struct {
	suite.Suite
}

func TestQuerySuite(t *testing.T) {
	suite.Run(t, new(QuerySuite))
}

func (s *QuerySuite) TestFilterOptions_Matches() {
	ticket := &Ticket{
		ID:       "t1",
		Status:   StatusOpen,
		Assignee: "alice",
		Tags:     []string{"backend", "urgent"},
		Type:     TypeBug,
	}

	tests := []struct {
		name   string
		filter FilterOptions
		want   bool
	}{
		{name: "empty filter matches all", filter: FilterOptions{}, want: true},
		{name: "status match", filter: FilterOptions{Status: "open"}, want: true},
		{name: "status mismatch", filter: FilterOptions{Status: "closed"}, want: false},
		{name: "assignee match", filter: FilterOptions{Assignee: "alice"}, want: true},
		{name: "assignee mismatch", filter: FilterOptions{Assignee: "bob"}, want: false},
		{name: "tag match", filter: FilterOptions{Tag: "backend"}, want: true},
		{name: "tag mismatch", filter: FilterOptions{Tag: "frontend"}, want: false},
		{name: "type match", filter: FilterOptions{Type: "bug"}, want: true},
		{name: "type mismatch", filter: FilterOptions{Type: "task"}, want: false},
		{name: "combined match", filter: FilterOptions{Status: "open", Assignee: "alice", Tag: "urgent"}, want: true},
		{name: "combined partial mismatch", filter: FilterOptions{Status: "open", Assignee: "bob"}, want: false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			require.Equal(s.T(), tt.want, tt.filter.Matches(ticket))
		})
	}
}

func (s *QuerySuite) TestFilter() {
	now := time.Now()
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Assignee: "alice", Tags: []string{"backend"}, Created: now},
		{ID: "t2", Status: StatusInProgress, Assignee: "bob", Tags: []string{"frontend"}, Created: now},
		{ID: "t3", Status: StatusClosed, Assignee: "alice", Tags: []string{"backend", "urgent"}, Created: now},
		{ID: "t4", Status: StatusOpen, Assignee: "charlie", Tags: []string{"api"}, Created: now},
	}

	tests := []struct {
		name    string
		filter  FilterOptions
		wantIDs []string
	}{
		{name: "no filter", filter: FilterOptions{}, wantIDs: []string{"t1", "t2", "t3", "t4"}},
		{name: "by status", filter: FilterOptions{Status: "open"}, wantIDs: []string{"t1", "t4"}},
		{name: "by assignee", filter: FilterOptions{Assignee: "alice"}, wantIDs: []string{"t1", "t3"}},
		{name: "by tag", filter: FilterOptions{Tag: "backend"}, wantIDs: []string{"t1", "t3"}},
		{name: "no match", filter: FilterOptions{Tag: "nonexistent"}, wantIDs: nil},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := Filter(tickets, tt.filter)
			var ids []string
			for _, t := range result {
				ids = append(ids, t.ID)
			}
			require.Equal(s.T(), tt.wantIDs, ids)
		})
	}
}

func (s *QuerySuite) TestSort_Priority() {
	tickets := []*Ticket{
		{ID: "t3", Priority: 3},
		{ID: "t1", Priority: 1},
		{ID: "t2", Priority: 2},
	}
	Sort(tickets, SortOptions{})
	require.Equal(s.T(), "t1", tickets[0].ID)
	require.Equal(s.T(), "t2", tickets[1].ID)
	require.Equal(s.T(), "t3", tickets[2].ID)
}

func (s *QuerySuite) TestSort_PrioritySameByID() {
	tickets := []*Ticket{
		{ID: "c", Priority: 1},
		{ID: "a", Priority: 1},
		{ID: "b", Priority: 1},
	}
	Sort(tickets, SortOptions{})
	require.Equal(s.T(), "a", tickets[0].ID)
	require.Equal(s.T(), "b", tickets[1].ID)
	require.Equal(s.T(), "c", tickets[2].ID)
}

func (s *QuerySuite) TestSort_Created() {
	now := time.Now()
	tickets := []*Ticket{
		{ID: "t2", Created: now.Add(-1 * time.Hour)},
		{ID: "t1", Created: now.Add(-3 * time.Hour)},
		{ID: "t3", Created: now},
	}
	Sort(tickets, SortOptions{SortBy: "created"})
	require.Equal(s.T(), "t1", tickets[0].ID)
	require.Equal(s.T(), "t2", tickets[1].ID)
	require.Equal(s.T(), "t3", tickets[2].ID)
}

func (s *QuerySuite) TestSort_Status() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen},
		{ID: "t2", Status: StatusClosed},
		{ID: "t3", Status: StatusInProgress},
	}
	Sort(tickets, SortOptions{SortBy: "status"})
	require.Equal(s.T(), "t2", tickets[0].ID) // closed
	require.Equal(s.T(), "t3", tickets[1].ID) // in_progress
	require.Equal(s.T(), "t1", tickets[2].ID) // open
}

func (s *QuerySuite) TestSort_Title() {
	tickets := []*Ticket{
		{ID: "t1", Title: "Beta"},
		{ID: "t2", Title: "Alpha"},
		{ID: "t3", Title: "Gamma"},
	}
	Sort(tickets, SortOptions{SortBy: "title"})
	require.Equal(s.T(), "t2", tickets[0].ID)
	require.Equal(s.T(), "t1", tickets[1].ID)
	require.Equal(s.T(), "t3", tickets[2].ID)
}

func (s *QuerySuite) TestSort_Reverse() {
	tickets := []*Ticket{
		{ID: "t1", Priority: 1},
		{ID: "t2", Priority: 2},
		{ID: "t3", Priority: 3},
	}
	Sort(tickets, SortOptions{Reverse: true})
	require.Equal(s.T(), "t3", tickets[0].ID)
	require.Equal(s.T(), "t2", tickets[1].ID)
	require.Equal(s.T(), "t1", tickets[2].ID)
}

func (s *QuerySuite) TestFilterByDependencyStatus_Ready() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen},
		{ID: "t2", Status: StatusOpen, Deps: []string{"t1"}},
		{ID: "t3", Status: StatusOpen, Deps: []string{"t4"}},
		{ID: "t4", Status: StatusClosed},
	}

	ready := FilterByDependencyStatus(tickets, false, FilterOptions{})
	var ids []string
	for _, t := range ready {
		ids = append(ids, t.ID)
	}
	// t1 has no deps (ready), t3 dep is closed (ready), t2 dep is open (blocked)
	require.Contains(s.T(), ids, "t1")
	require.Contains(s.T(), ids, "t3")
	require.NotContains(s.T(), ids, "t2")
	require.NotContains(s.T(), ids, "t4") // closed tickets excluded
}

func (s *QuerySuite) TestFilterByDependencyStatus_Blocked() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen},
		{ID: "t2", Status: StatusOpen, Deps: []string{"t1"}},
		{ID: "t3", Status: StatusClosed},
	}

	blocked := FilterByDependencyStatus(tickets, true, FilterOptions{})
	require.Len(s.T(), blocked, 1)
	require.Equal(s.T(), "t2", blocked[0].ID)
}

func (s *QuerySuite) TestFilterByDependencyStatus_WithFilter() {
	tickets := []*Ticket{
		{ID: "t1", Status: StatusOpen, Assignee: "alice"},
		{ID: "t2", Status: StatusOpen, Assignee: "bob"},
	}

	result := FilterByDependencyStatus(tickets, false, FilterOptions{Assignee: "alice"})
	require.Len(s.T(), result, 1)
	require.Equal(s.T(), "t1", result[0].ID)
}

func (s *QuerySuite) TestHasTag() {
	require.True(s.T(), HasTag([]string{"backend", "api"}, "api"))
	require.True(s.T(), HasTag([]string{"Backend", "API"}, "api"))
	require.False(s.T(), HasTag([]string{"backend"}, "frontend"))
	require.False(s.T(), HasTag(nil, "api"))
	require.False(s.T(), HasTag([]string{}, "api"))
}
