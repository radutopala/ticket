package ticket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DepsSuite struct {
	suite.Suite
}

func TestDepsSuite(t *testing.T) {
	suite.Run(t, new(DepsSuite))
}

func (s *DepsSuite) TestCheckCycle_NoCycle() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b"},
		{ID: "c"},
	}
	require.NoError(s.T(), CheckCycle(tickets, "c", "a"))
}

func (s *DepsSuite) TestCheckCycle_DirectCycle() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b"},
	}
	require.Error(s.T(), CheckCycle(tickets, "b", "a"))
}

func (s *DepsSuite) TestCheckCycle_IndirectCycle() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b", Deps: []string{"c"}},
		{ID: "c"},
	}
	require.Error(s.T(), CheckCycle(tickets, "c", "a"))
}

func (s *DepsSuite) TestCheckCycle_SelfDep() {
	tickets := []*Ticket{
		{ID: "a"},
	}
	require.Error(s.T(), CheckCycle(tickets, "a", "a"))
}

func (s *DepsSuite) TestCheckCycle_DoesNotMutateOriginal() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b"},
		{ID: "c"},
	}
	origLen := len(tickets[0].Deps)
	_ = CheckCycle(tickets, "a", "c")
	require.Len(s.T(), tickets[0].Deps, origLen, "original Deps slice must not be mutated")
}

func (s *DepsSuite) TestTopologicalSort_NoDeps() {
	tickets := []*Ticket{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}
	sorted, err := TopologicalSort(tickets)
	require.NoError(s.T(), err)
	require.Len(s.T(), sorted, 3)
}

func (s *DepsSuite) TestTopologicalSort_LinearChain() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b", Deps: []string{"c"}},
		{ID: "c"},
	}
	sorted, err := TopologicalSort(tickets)
	require.NoError(s.T(), err)

	ids := make([]string, len(sorted))
	for i, t := range sorted {
		ids[i] = t.ID
	}
	require.Equal(s.T(), []string{"c", "b", "a"}, ids)
}

func (s *DepsSuite) TestTopologicalSort_Diamond() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b", "c"}},
		{ID: "b", Deps: []string{"d"}},
		{ID: "c", Deps: []string{"d"}},
		{ID: "d"},
	}
	sorted, err := TopologicalSort(tickets)
	require.NoError(s.T(), err)
	require.Len(s.T(), sorted, 4)

	idxMap := make(map[string]int)
	for i, t := range sorted {
		idxMap[t.ID] = i
	}
	require.Less(s.T(), idxMap["d"], idxMap["b"])
	require.Less(s.T(), idxMap["d"], idxMap["c"])
	require.Less(s.T(), idxMap["b"], idxMap["a"])
	require.Less(s.T(), idxMap["c"], idxMap["a"])
}

func (s *DepsSuite) TestTopologicalSort_Cycle() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b", Deps: []string{"c"}},
		{ID: "c", Deps: []string{"a"}},
	}
	_, err := TopologicalSort(tickets)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "cycle")
}

func (s *DepsSuite) TestDetectCycles_NoCycles() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b", Deps: []string{"c"}},
		{ID: "c"},
	}
	cycles := DetectCycles(tickets)
	require.Empty(s.T(), cycles)
}

func (s *DepsSuite) TestDetectCycles_SimpleCycle() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b", Deps: []string{"a"}},
	}
	cycles := DetectCycles(tickets)
	require.Len(s.T(), cycles, 1)
}

func (s *DepsSuite) TestDetectCycles_ThreeNodeCycle() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"b"}},
		{ID: "b", Deps: []string{"c"}},
		{ID: "c", Deps: []string{"a"}},
	}
	cycles := DetectCycles(tickets)
	require.Len(s.T(), cycles, 1)
}

func (s *DepsSuite) TestDetectCycles_SelfLoop() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"a"}},
	}
	cycles := DetectCycles(tickets)
	require.Len(s.T(), cycles, 1)
}

func (s *DepsSuite) TestDetectCycles_MissingDep() {
	tickets := []*Ticket{
		{ID: "a", Deps: []string{"nonexistent"}},
	}
	cycles := DetectCycles(tickets)
	require.Empty(s.T(), cycles)
}

func (s *DepsSuite) TestFindRootTickets() {
	now := time.Now()
	tickets := []*Ticket{
		{ID: "root1", Status: StatusOpen, Created: now},
		{ID: "root2", Status: StatusOpen, Created: now},
		{ID: "child", Status: StatusOpen, Deps: []string{"root1"}, Created: now},
		{ID: "closed", Status: StatusClosed, Created: now},
	}

	roots := FindRootTickets(tickets)
	rootIDs := make(map[string]bool)
	for _, r := range roots {
		rootIDs[r.ID] = true
	}

	require.True(s.T(), rootIDs["root2"])
	require.True(s.T(), rootIDs["child"])
	require.False(s.T(), rootIDs["root1"], "root1 is a dependency so not a root")
	require.False(s.T(), rootIDs["closed"], "closed tickets excluded")
}

func (s *DepsSuite) TestFindRootTickets_AllClosed() {
	tickets := []*Ticket{
		{ID: "a", Status: StatusClosed},
		{ID: "b", Status: StatusClosed},
	}
	roots := FindRootTickets(tickets)
	require.Empty(s.T(), roots)
}

func (s *DepsSuite) TestFindRootTickets_Empty() {
	roots := FindRootTickets(nil)
	require.Empty(s.T(), roots)
}

func (s *DepsSuite) TestBuildOpenIDSet() {
	tickets := []*Ticket{
		{ID: "open1", Status: StatusOpen},
		{ID: "wip", Status: StatusInProgress},
		{ID: "done", Status: StatusClosed},
	}
	set := BuildOpenIDSet(tickets)

	require.True(s.T(), set["open1"])
	require.True(s.T(), set["wip"])
	require.False(s.T(), set["done"])
}

func (s *DepsSuite) TestBuildOpenIDSet_Empty() {
	set := BuildOpenIDSet(nil)
	require.Empty(s.T(), set)
}

// --- AddDep tests ---

func (s *DepsSuite) TestAddDep_Success() {
	ticket := &Ticket{ID: "a"}
	all := []*Ticket{ticket, {ID: "b"}}

	err := AddDep(ticket, "b", all)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []string{"b"}, ticket.Deps)
}

func (s *DepsSuite) TestAddDep_SelfDep() {
	ticket := &Ticket{ID: "a"}
	all := []*Ticket{ticket}

	err := AddDep(ticket, "a", all)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "cannot depend on itself")
	require.Empty(s.T(), ticket.Deps)
}

func (s *DepsSuite) TestAddDep_Duplicate() {
	ticket := &Ticket{ID: "a", Deps: []string{"b"}}
	all := []*Ticket{ticket, {ID: "b"}}

	err := AddDep(ticket, "b", all)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "already exists")
	require.Equal(s.T(), []string{"b"}, ticket.Deps)
}

func (s *DepsSuite) TestAddDep_Cycle() {
	ticket := &Ticket{ID: "a"}
	all := []*Ticket{
		ticket,
		{ID: "b", Deps: []string{"a"}},
	}

	err := AddDep(ticket, "b", all)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "cycle")
	require.Empty(s.T(), ticket.Deps)
}

func (s *DepsSuite) TestAddDep_MultipleAdds() {
	ticket := &Ticket{ID: "a"}
	all := []*Ticket{ticket, {ID: "b"}, {ID: "c"}}

	require.NoError(s.T(), AddDep(ticket, "b", all))
	require.NoError(s.T(), AddDep(ticket, "c", all))
	require.Equal(s.T(), []string{"b", "c"}, ticket.Deps)
}

// --- RemoveDep tests ---

func (s *DepsSuite) TestRemoveDep_Success() {
	ticket := &Ticket{ID: "a", Deps: []string{"b", "c"}}

	err := RemoveDep(ticket, "b")
	require.NoError(s.T(), err)
	require.Equal(s.T(), []string{"c"}, ticket.Deps)
}

func (s *DepsSuite) TestRemoveDep_NotFound() {
	ticket := &Ticket{ID: "a", Deps: []string{"b"}}

	err := RemoveDep(ticket, "c")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
	require.Equal(s.T(), []string{"b"}, ticket.Deps)
}

func (s *DepsSuite) TestRemoveDep_Empty() {
	ticket := &Ticket{ID: "a"}

	err := RemoveDep(ticket, "b")
	require.Error(s.T(), err)
}

// --- RemoveFromSlice tests ---

func (s *DepsSuite) TestRemoveFromSlice_Found() {
	result, found := RemoveFromSlice([]string{"a", "b", "c"}, "b")
	require.True(s.T(), found)
	require.Equal(s.T(), []string{"a", "c"}, result)
}

func (s *DepsSuite) TestRemoveFromSlice_NotFound() {
	result, found := RemoveFromSlice([]string{"a", "b"}, "z")
	require.False(s.T(), found)
	require.Equal(s.T(), []string{"a", "b"}, result)
}

func (s *DepsSuite) TestRemoveFromSlice_Empty() {
	result, found := RemoveFromSlice([]string{}, "a")
	require.False(s.T(), found)
	require.Empty(s.T(), result)
}

func (s *DepsSuite) TestRemoveFromSlice_AllOccurrences() {
	result, found := RemoveFromSlice([]string{"a", "b", "a", "c"}, "a")
	require.True(s.T(), found)
	require.Equal(s.T(), []string{"b", "c"}, result)
}
