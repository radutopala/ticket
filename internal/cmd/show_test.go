package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

type ShowSuite struct {
	suite.Suite
}

func TestShowSuite(t *testing.T) {
	suite.Run(t, new(ShowSuite))
}

func (s *ShowSuite) TestGetTicketRelationships_NoRelationships() {
	ticket := &tk.Ticket{
		ID:      "tic-main",
		Status:  tk.StatusOpen,
		Created: time.Now().UTC(),
	}
	allTickets := []*tk.Ticket{
		ticket,
		{ID: "tic-other", Status: tk.StatusOpen, Created: time.Now().UTC()},
	}

	rel := tk.GetRelationships("tic-main", ticket, allTickets)
	result := formatRelationships(rel)
	require.Equal(s.T(), "", result)
}

func (s *ShowSuite) TestGetTicketRelationships_WithBlockers() {
	ticket := &tk.Ticket{
		ID:      "tic-main",
		Status:  tk.StatusOpen,
		Deps:    []string{"tic-blocker1", "tic-blocker2"},
		Created: time.Now().UTC(),
	}
	allTickets := []*tk.Ticket{
		ticket,
		{ID: "tic-blocker1", Status: tk.StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-blocker2", Status: tk.StatusOpen, Created: time.Now().UTC()},
	}

	rel := tk.GetRelationships("tic-main", ticket, allTickets)
	result := formatRelationships(rel)
	require.Contains(s.T(), result, "Blockers: tic-blocker1, tic-blocker2")
}

func (s *ShowSuite) TestGetTicketRelationships_WithBlocking() {
	ticket := &tk.Ticket{
		ID:      "tic-main",
		Status:  tk.StatusOpen,
		Created: time.Now().UTC(),
	}
	allTickets := []*tk.Ticket{
		ticket,
		{ID: "tic-dependent1", Status: tk.StatusOpen, Deps: []string{"tic-main"}, Created: time.Now().UTC()},
		{ID: "tic-dependent2", Status: tk.StatusOpen, Deps: []string{"tic-main"}, Created: time.Now().UTC()},
	}

	rel := tk.GetRelationships("tic-main", ticket, allTickets)
	result := formatRelationships(rel)
	require.Contains(s.T(), result, "Blocking:")
	require.Contains(s.T(), result, "tic-dependent1")
	require.Contains(s.T(), result, "tic-dependent2")
}

func (s *ShowSuite) TestGetTicketRelationships_WithChildren() {
	ticket := &tk.Ticket{
		ID:      "tic-parent",
		Status:  tk.StatusOpen,
		Created: time.Now().UTC(),
	}
	allTickets := []*tk.Ticket{
		ticket,
		{ID: "tic-child1", Status: tk.StatusOpen, Parent: "tic-parent", Created: time.Now().UTC()},
		{ID: "tic-child2", Status: tk.StatusOpen, Parent: "tic-parent", Created: time.Now().UTC()},
	}

	rel := tk.GetRelationships("tic-parent", ticket, allTickets)
	result := formatRelationships(rel)
	require.Contains(s.T(), result, "Children:")
	require.Contains(s.T(), result, "tic-child1")
	require.Contains(s.T(), result, "tic-child2")
}

func (s *ShowSuite) TestGetTicketRelationships_WithLinks() {
	ticket := &tk.Ticket{
		ID:      "tic-main",
		Status:  tk.StatusOpen,
		Links:   []string{"tic-linked1", "tic-linked2"},
		Created: time.Now().UTC(),
	}
	allTickets := []*tk.Ticket{
		ticket,
		{ID: "tic-linked1", Status: tk.StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-linked2", Status: tk.StatusOpen, Created: time.Now().UTC()},
	}

	rel := tk.GetRelationships("tic-main", ticket, allTickets)
	result := formatRelationships(rel)
	require.Contains(s.T(), result, "Links: tic-linked1, tic-linked2")
}

func (s *ShowSuite) TestGetTicketRelationships_AllRelationshipTypes() {
	ticket := &tk.Ticket{
		ID:      "tic-main",
		Status:  tk.StatusOpen,
		Deps:    []string{"tic-blocker"},
		Links:   []string{"tic-linked"},
		Created: time.Now().UTC(),
	}
	allTickets := []*tk.Ticket{
		ticket,
		{ID: "tic-blocker", Status: tk.StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-dependent", Status: tk.StatusOpen, Deps: []string{"tic-main"}, Created: time.Now().UTC()},
		{ID: "tic-child", Status: tk.StatusOpen, Parent: "tic-main", Created: time.Now().UTC()},
		{ID: "tic-linked", Status: tk.StatusOpen, Created: time.Now().UTC()},
	}

	rel := tk.GetRelationships("tic-main", ticket, allTickets)
	result := formatRelationships(rel)
	require.Contains(s.T(), result, "Blockers: tic-blocker")
	require.Contains(s.T(), result, "Blocking: tic-dependent")
	require.Contains(s.T(), result, "Children: tic-child")
	require.Contains(s.T(), result, "Links: tic-linked")
}

func (s *ShowSuite) TestGetTicketRelationships_SkipsSelf() {
	ticket := &tk.Ticket{
		ID:      "tic-main",
		Status:  tk.StatusOpen,
		Deps:    []string{"tic-main"}, // Self-dependency (shouldn't happen but test edge case)
		Created: time.Now().UTC(),
	}
	allTickets := []*tk.Ticket{
		ticket,
	}

	rel := tk.GetRelationships("tic-main", ticket, allTickets)
	result := formatRelationships(rel)
	// Should only show blockers (self), not self in blocking
	require.Contains(s.T(), result, "Blockers: tic-main")
	require.NotContains(s.T(), result, "Blocking:")
}
