package ticket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RelationshipsSuite struct {
	suite.Suite
}

func TestRelationshipsSuite(t *testing.T) {
	suite.Run(t, new(RelationshipsSuite))
}

func (s *RelationshipsSuite) TestGetRelationships_NoRelationships() {
	ticket := &Ticket{ID: "t1", Status: StatusOpen, Created: time.Now()}
	all := []*Ticket{
		ticket,
		{ID: "t2", Status: StatusOpen, Created: time.Now()},
	}

	rel := GetRelationships("t1", ticket, all)
	require.Empty(s.T(), rel.Blockers)
	require.Empty(s.T(), rel.Blocking)
	require.Empty(s.T(), rel.Children)
	require.Empty(s.T(), rel.Links)
}

func (s *RelationshipsSuite) TestGetRelationships_Blockers() {
	ticket := &Ticket{
		ID:   "t1",
		Deps: []string{"t2", "t3"},
	}
	all := []*Ticket{
		ticket,
		{ID: "t2"},
		{ID: "t3"},
	}

	rel := GetRelationships("t1", ticket, all)
	require.Equal(s.T(), []string{"t2", "t3"}, rel.Blockers)
}

func (s *RelationshipsSuite) TestGetRelationships_Blocking() {
	ticket := &Ticket{ID: "t1"}
	all := []*Ticket{
		ticket,
		{ID: "t2", Deps: []string{"t1"}},
		{ID: "t3", Deps: []string{"t1"}},
		{ID: "t4", Deps: []string{"t5"}},
	}

	rel := GetRelationships("t1", ticket, all)
	require.Equal(s.T(), []string{"t2", "t3"}, rel.Blocking)
}

func (s *RelationshipsSuite) TestGetRelationships_Children() {
	ticket := &Ticket{ID: "parent"}
	all := []*Ticket{
		ticket,
		{ID: "child1", Parent: "parent"},
		{ID: "child2", Parent: "parent"},
		{ID: "other", Parent: "someone-else"},
	}

	rel := GetRelationships("parent", ticket, all)
	require.Equal(s.T(), []string{"child1", "child2"}, rel.Children)
}

func (s *RelationshipsSuite) TestGetRelationships_Links() {
	ticket := &Ticket{
		ID:    "t1",
		Links: []string{"t2", "t3"},
	}
	all := []*Ticket{ticket, {ID: "t2"}, {ID: "t3"}}

	rel := GetRelationships("t1", ticket, all)
	require.Equal(s.T(), []string{"t2", "t3"}, rel.Links)
}

func (s *RelationshipsSuite) TestGetRelationships_AllTypes() {
	ticket := &Ticket{
		ID:    "main",
		Deps:  []string{"blocker"},
		Links: []string{"linked"},
	}
	all := []*Ticket{
		ticket,
		{ID: "blocker"},
		{ID: "dependent", Deps: []string{"main"}},
		{ID: "child", Parent: "main"},
		{ID: "linked"},
	}

	rel := GetRelationships("main", ticket, all)
	require.Equal(s.T(), []string{"blocker"}, rel.Blockers)
	require.Equal(s.T(), []string{"dependent"}, rel.Blocking)
	require.Equal(s.T(), []string{"child"}, rel.Children)
	require.Equal(s.T(), []string{"linked"}, rel.Links)
}

func (s *RelationshipsSuite) TestGetRelationships_SkipsSelf() {
	ticket := &Ticket{
		ID:   "t1",
		Deps: []string{"t1"},
	}
	all := []*Ticket{ticket}

	rel := GetRelationships("t1", ticket, all)
	require.Equal(s.T(), []string{"t1"}, rel.Blockers)
	require.Empty(s.T(), rel.Blocking, "self should be skipped in blocking scan")
}

// --- ValidateLinkIDs tests ---

func (s *RelationshipsSuite) TestValidateLinkIDs_NoDuplicates() {
	require.NoError(s.T(), ValidateLinkIDs([]string{"a", "b", "c"}))
}

func (s *RelationshipsSuite) TestValidateLinkIDs_WithDuplicate() {
	err := ValidateLinkIDs([]string{"a", "b", "a"})
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "duplicate ticket ID: a")
}

func (s *RelationshipsSuite) TestValidateLinkIDs_Empty() {
	require.NoError(s.T(), ValidateLinkIDs([]string{}))
}

// --- AddLink tests ---

func (s *RelationshipsSuite) TestAddLink_New() {
	ticket := &Ticket{ID: "t1"}
	added := AddLink(ticket, "t2")
	require.True(s.T(), added)
	require.Equal(s.T(), []string{"t2"}, ticket.Links)
}

func (s *RelationshipsSuite) TestAddLink_AlreadyExists() {
	ticket := &Ticket{ID: "t1", Links: []string{"t2"}}
	added := AddLink(ticket, "t2")
	require.False(s.T(), added)
	require.Equal(s.T(), []string{"t2"}, ticket.Links)
}

func (s *RelationshipsSuite) TestAddLink_Multiple() {
	ticket := &Ticket{ID: "t1"}
	require.True(s.T(), AddLink(ticket, "t2"))
	require.True(s.T(), AddLink(ticket, "t3"))
	require.False(s.T(), AddLink(ticket, "t2"))
	require.Equal(s.T(), []string{"t2", "t3"}, ticket.Links)
}

// --- RemoveLink tests ---

func (s *RelationshipsSuite) TestRemoveLink_Found() {
	ticket := &Ticket{ID: "t1", Links: []string{"t2", "t3"}}
	found := RemoveLink(ticket, "t2")
	require.True(s.T(), found)
	require.Equal(s.T(), []string{"t3"}, ticket.Links)
}

func (s *RelationshipsSuite) TestRemoveLink_NotFound() {
	ticket := &Ticket{ID: "t1", Links: []string{"t2"}}
	found := RemoveLink(ticket, "t3")
	require.False(s.T(), found)
	require.Equal(s.T(), []string{"t2"}, ticket.Links)
}

func (s *RelationshipsSuite) TestRemoveLink_Empty() {
	ticket := &Ticket{ID: "t1"}
	found := RemoveLink(ticket, "t2")
	require.False(s.T(), found)
	require.Empty(s.T(), ticket.Links)
}
