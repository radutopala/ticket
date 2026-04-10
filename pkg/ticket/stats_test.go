package ticket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type StatsSuite struct {
	suite.Suite
}

func TestStatsSuite(t *testing.T) {
	suite.Run(t, new(StatsSuite))
}

func (s *StatsSuite) TestComputeStats_Empty() {
	stats := ComputeStats([]*Ticket{})
	require.Equal(s.T(), 0, stats.Total)
	require.Empty(s.T(), stats.ByStatus)
	require.Empty(s.T(), stats.ByType)
	require.Empty(s.T(), stats.ByAssignee)
}

func (s *StatsSuite) TestComputeStats_Single() {
	now := time.Now()
	stats := ComputeStats([]*Ticket{
		{ID: "t1", Status: StatusOpen, Type: TypeTask, Assignee: "alice", Created: now},
	})
	require.Equal(s.T(), 1, stats.Total)
	require.Equal(s.T(), map[string]int{"open": 1}, stats.ByStatus)
	require.Equal(s.T(), map[string]int{"task": 1}, stats.ByType)
	require.Equal(s.T(), map[string]int{"alice": 1}, stats.ByAssignee)
}

func (s *StatsSuite) TestComputeStats_Multiple() {
	now := time.Now()
	stats := ComputeStats([]*Ticket{
		{ID: "t1", Status: StatusOpen, Type: TypeTask, Assignee: "alice", Created: now},
		{ID: "t2", Status: StatusOpen, Type: TypeBug, Assignee: "bob", Created: now},
		{ID: "t3", Status: StatusInProgress, Type: TypeFeature, Assignee: "alice", Created: now},
		{ID: "t4", Status: StatusClosed, Type: TypeTask, Assignee: "charlie", Created: now},
		{ID: "t5", Status: StatusClosed, Type: TypeBug, Created: now},
	})

	require.Equal(s.T(), 5, stats.Total)
	require.Equal(s.T(), 2, stats.ByStatus["open"])
	require.Equal(s.T(), 1, stats.ByStatus["in_progress"])
	require.Equal(s.T(), 2, stats.ByStatus["closed"])
	require.Equal(s.T(), 2, stats.ByType["task"])
	require.Equal(s.T(), 2, stats.ByType["bug"])
	require.Equal(s.T(), 1, stats.ByType["feature"])
	require.Equal(s.T(), 2, stats.ByAssignee["alice"])
	require.Equal(s.T(), 1, stats.ByAssignee["bob"])
	require.Equal(s.T(), 1, stats.ByAssignee["charlie"])
	require.Equal(s.T(), 1, stats.ByAssignee["unassigned"])
}

func (s *StatsSuite) TestComputeStats_NoType() {
	now := time.Now()
	stats := ComputeStats([]*Ticket{
		{ID: "t1", Status: StatusOpen, Assignee: "alice", Created: now},
	})
	require.Empty(s.T(), stats.ByType)
}

func (s *StatsSuite) TestComputeStats_Unassigned() {
	now := time.Now()
	stats := ComputeStats([]*Ticket{
		{ID: "t1", Status: StatusOpen, Created: now},
	})
	require.Equal(s.T(), 1, stats.ByAssignee["unassigned"])
}
