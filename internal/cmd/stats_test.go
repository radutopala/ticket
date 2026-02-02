package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/radutopala/ticket/internal/domain"
)

type StatsSuite struct {
	suite.Suite
}

func TestStatsSuite(t *testing.T) {
	suite.Run(t, new(StatsSuite))
}

func (s *StatsSuite) TestComputeStats() {
	now := time.Now()
	tests := []struct {
		name    string
		tickets []*domain.Ticket
		want    Stats
	}{
		{
			name:    "empty tickets",
			tickets: []*domain.Ticket{},
			want: Stats{
				Total:      0,
				ByStatus:   map[string]int{},
				ByType:     map[string]int{},
				ByAssignee: map[string]int{},
			},
		},
		{
			name: "single ticket",
			tickets: []*domain.Ticket{
				{ID: "t1", Status: domain.StatusOpen, Type: domain.TypeTask, Assignee: "alice", Created: now},
			},
			want: Stats{
				Total:      1,
				ByStatus:   map[string]int{"open": 1},
				ByType:     map[string]int{"task": 1},
				ByAssignee: map[string]int{"alice": 1},
			},
		},
		{
			name: "multiple tickets with various attributes",
			tickets: []*domain.Ticket{
				{ID: "t1", Status: domain.StatusOpen, Type: domain.TypeTask, Assignee: "alice", Created: now},
				{ID: "t2", Status: domain.StatusOpen, Type: domain.TypeBug, Assignee: "bob", Created: now},
				{ID: "t3", Status: domain.StatusInProgress, Type: domain.TypeFeature, Assignee: "alice", Created: now},
				{ID: "t4", Status: domain.StatusClosed, Type: domain.TypeTask, Assignee: "charlie", Created: now},
				{ID: "t5", Status: domain.StatusClosed, Type: domain.TypeBug, Created: now}, // unassigned
			},
			want: Stats{
				Total: 5,
				ByStatus: map[string]int{
					"open":        2,
					"in_progress": 1,
					"closed":      2,
				},
				ByType: map[string]int{
					"task":    2,
					"bug":     2,
					"feature": 1,
				},
				ByAssignee: map[string]int{
					"alice":      2,
					"bob":        1,
					"charlie":    1,
					"unassigned": 1,
				},
			},
		},
		{
			name: "tickets without type",
			tickets: []*domain.Ticket{
				{ID: "t1", Status: domain.StatusOpen, Assignee: "alice", Created: now},
				{ID: "t2", Status: domain.StatusOpen, Type: domain.TypeTask, Assignee: "alice", Created: now},
			},
			want: Stats{
				Total:      2,
				ByStatus:   map[string]int{"open": 2},
				ByType:     map[string]int{"task": 1},
				ByAssignee: map[string]int{"alice": 2},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := computeStats(tt.tickets)
			require.Equal(s.T(), tt.want.Total, got.Total)
			require.Equal(s.T(), tt.want.ByStatus, got.ByStatus)
			require.Equal(s.T(), tt.want.ByType, got.ByType)
			require.Equal(s.T(), tt.want.ByAssignee, got.ByAssignee)
		})
	}
}

func (s *StatsSuite) TestOutputStatsJSON() {
	stats := Stats{
		Total: 3,
		ByStatus: map[string]int{
			"open":   2,
			"closed": 1,
		},
		ByType: map[string]int{
			"task": 2,
			"bug":  1,
		},
		ByAssignee: map[string]int{
			"alice":      2,
			"unassigned": 1,
		},
	}

	var buf bytes.Buffer
	err := outputStatsJSON(&buf, stats)
	require.NoError(s.T(), err)

	var got Stats
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(s.T(), err)

	require.Equal(s.T(), stats.Total, got.Total)
	require.Equal(s.T(), stats.ByStatus, got.ByStatus)
	require.Equal(s.T(), stats.ByType, got.ByType)
	require.Equal(s.T(), stats.ByAssignee, got.ByAssignee)
}

func (s *StatsSuite) TestOutputStatsText() {
	stats := Stats{
		Total: 5,
		ByStatus: map[string]int{
			"open":        2,
			"in_progress": 1,
			"closed":      2,
		},
		ByType: map[string]int{
			"task":    2,
			"bug":     1,
			"feature": 2,
		},
		ByAssignee: map[string]int{
			"alice":      2,
			"bob":        1,
			"unassigned": 2,
		},
	}

	var buf bytes.Buffer
	err := outputStatsText(&buf, stats)
	require.NoError(s.T(), err)

	output := buf.String()

	// Check total
	require.Contains(s.T(), output, "Total: 5 tickets")

	// Check status section
	require.Contains(s.T(), output, "By Status:")
	require.Contains(s.T(), output, "open:")
	require.Contains(s.T(), output, "in_progress:")
	require.Contains(s.T(), output, "closed:")

	// Check type section
	require.Contains(s.T(), output, "By Type:")
	require.Contains(s.T(), output, "task:")
	require.Contains(s.T(), output, "bug:")
	require.Contains(s.T(), output, "feature:")

	// Check assignee section
	require.Contains(s.T(), output, "By Assignee:")
	require.Contains(s.T(), output, "alice:")
	require.Contains(s.T(), output, "bob:")
	require.Contains(s.T(), output, "unassigned:")
}

func (s *StatsSuite) TestSortedKeys() {
	tests := []struct {
		name string
		m    map[string]int
		want []string
	}{
		{
			name: "sorts alphabetically",
			m:    map[string]int{"charlie": 1, "alice": 2, "bob": 3},
			want: []string{"alice", "bob", "charlie"},
		},
		{
			name: "empty map",
			m:    map[string]int{},
			want: []string{},
		},
		{
			name: "single key",
			m:    map[string]int{"alice": 1},
			want: []string{"alice"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := sortedKeys(tt.m)
			require.Equal(s.T(), tt.want, got)
		})
	}
}

func (s *StatsSuite) TestMaxKeyLen() {
	tests := []struct {
		name string
		keys []string
		want int
	}{
		{
			name: "various lengths",
			keys: []string{"a", "abc", "ab"},
			want: 3,
		},
		{
			name: "empty slice",
			keys: []string{},
			want: 0,
		},
		{
			name: "single key",
			keys: []string{"hello"},
			want: 5,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := maxKeyLen(tt.keys)
			require.Equal(s.T(), tt.want, got)
		})
	}
}
