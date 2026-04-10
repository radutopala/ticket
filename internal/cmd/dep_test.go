package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

// captureOutput captures stdout during function execution.
func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

type DepSuite struct {
	suite.Suite
}

func TestDepSuite(t *testing.T) {
	suite.Run(t, new(DepSuite))
}

func (s *DepSuite) TestTopologicalSort() {
	tests := []struct {
		name      string
		tickets   []*tk.Ticket
		wantOrder []string
		wantErr   bool
	}{
		{
			name: "no dependencies",
			tickets: []*tk.Ticket{
				{ID: "a", Status: tk.StatusOpen},
				{ID: "b", Status: tk.StatusOpen},
				{ID: "c", Status: tk.StatusOpen},
			},
			wantOrder: nil, // Any order is valid when there are no dependencies
			wantErr:   false,
		},
		{
			name: "linear chain",
			tickets: []*tk.Ticket{
				{ID: "a", Status: tk.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: tk.StatusOpen, Deps: []string{"c"}},
				{ID: "c", Status: tk.StatusOpen},
			},
			wantOrder: []string{"c", "b", "a"},
			wantErr:   false,
		},
		{
			name: "diamond dependency",
			tickets: []*tk.Ticket{
				{ID: "a", Status: tk.StatusOpen, Deps: []string{"b", "c"}},
				{ID: "b", Status: tk.StatusOpen, Deps: []string{"d"}},
				{ID: "c", Status: tk.StatusOpen, Deps: []string{"d"}},
				{ID: "d", Status: tk.StatusOpen},
			},
			wantErr: false,
		},
		{
			name: "cycle",
			tickets: []*tk.Ticket{
				{ID: "a", Status: tk.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: tk.StatusOpen, Deps: []string{"c"}},
				{ID: "c", Status: tk.StatusOpen, Deps: []string{"a"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			sorted, err := tk.TopologicalSort(tt.tickets)
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
		tickets    []*tk.Ticket
		wantCycles int
	}{
		{
			name: "no cycles",
			tickets: []*tk.Ticket{
				{ID: "a", Status: tk.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: tk.StatusOpen, Deps: []string{"c"}},
				{ID: "c", Status: tk.StatusOpen},
			},
			wantCycles: 0,
		},
		{
			name: "simple cycle",
			tickets: []*tk.Ticket{
				{ID: "a", Status: tk.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: tk.StatusOpen, Deps: []string{"a"}},
			},
			wantCycles: 1,
		},
		{
			name: "three node cycle",
			tickets: []*tk.Ticket{
				{ID: "a", Status: tk.StatusOpen, Deps: []string{"b"}},
				{ID: "b", Status: tk.StatusOpen, Deps: []string{"c"}},
				{ID: "c", Status: tk.StatusOpen, Deps: []string{"a"}},
			},
			wantCycles: 1,
		},
		{
			name: "self loop",
			tickets: []*tk.Ticket{
				{ID: "a", Status: tk.StatusOpen, Deps: []string{"a"}},
			},
			wantCycles: 1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cycles := tk.DetectCycles(tt.tickets)
			require.Len(s.T(), cycles, tt.wantCycles)
		})
	}
}

func (s *DepSuite) TestStatusIndicator() {
	require.Equal(s.T(), "[ ]", statusIndicator(tk.StatusOpen))
	require.Equal(s.T(), "[~]", statusIndicator(tk.StatusInProgress))
	require.Equal(s.T(), "[x]", statusIndicator(tk.StatusClosed))
	require.Equal(s.T(), "[?]", statusIndicator(tk.Status("unknown")))
}

func (s *DepSuite) TestFindRootTickets() {
	now := time.Now()
	tickets := []*tk.Ticket{
		{ID: "root1", Status: tk.StatusOpen, Created: now},
		{ID: "root2", Status: tk.StatusOpen, Created: now},
		{ID: "child", Status: tk.StatusOpen, Deps: []string{"root1"}, Created: now},
		{ID: "closed", Status: tk.StatusClosed, Created: now},
	}

	roots := tk.FindRootTickets(tickets)

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

func (s *DepSuite) TestPrintDepTree() {
	tests := []struct {
		name      string
		ticket    *tk.Ticket
		ticketMap map[string]*tk.Ticket
		prefix    string
		isLast    bool
		expected  string
	}{
		{
			name: "root node with no dependencies",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Root ticket",
				Status: tk.StatusOpen,
			},
			ticketMap: map[string]*tk.Ticket{},
			prefix:    "",
			isLast:    true,
			expected:  "[ ] tic-001 - Root ticket\n",
		},
		{
			name: "root node in progress",
			ticket: &tk.Ticket{
				ID:     "tic-002",
				Title:  "In progress ticket",
				Status: tk.StatusInProgress,
			},
			ticketMap: map[string]*tk.Ticket{},
			prefix:    "",
			isLast:    true,
			expected:  "[~] tic-002 - In progress ticket\n",
		},
		{
			name: "root node closed",
			ticket: &tk.Ticket{
				ID:     "tic-003",
				Title:  "Closed ticket",
				Status: tk.StatusClosed,
			},
			ticketMap: map[string]*tk.Ticket{},
			prefix:    "",
			isLast:    true,
			expected:  "[x] tic-003 - Closed ticket\n",
		},
		{
			name: "non-root with isLast true uses └── connector",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Last child",
				Status: tk.StatusOpen,
			},
			ticketMap: map[string]*tk.Ticket{},
			prefix:    "    ",
			isLast:    true,
			expected:  "    └── [ ] tic-001 - Last child\n",
		},
		{
			name: "non-root with isLast false uses ├── connector",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Middle child",
				Status: tk.StatusOpen,
			},
			ticketMap: map[string]*tk.Ticket{},
			prefix:    "    ",
			isLast:    false,
			expected:  "    ├── [ ] tic-001 - Middle child\n",
		},
		{
			name: "ticket with single dependency from root",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Parent ticket",
				Status: tk.StatusOpen,
				Deps:   []string{"tic-002"},
			},
			ticketMap: map[string]*tk.Ticket{
				"tic-002": {
					ID:     "tic-002",
					Title:  "Dependency ticket",
					Status: tk.StatusOpen,
				},
			},
			prefix:   "",
			isLast:   true,
			expected: "[ ] tic-001 - Parent ticket\n[ ] tic-002 - Dependency ticket\n",
		},
		{
			name: "ticket with multiple dependencies from root",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Parent",
				Status: tk.StatusOpen,
				Deps:   []string{"tic-002", "tic-003"},
			},
			ticketMap: map[string]*tk.Ticket{
				"tic-002": {
					ID:     "tic-002",
					Title:  "First dep",
					Status: tk.StatusOpen,
				},
				"tic-003": {
					ID:     "tic-003",
					Title:  "Second dep",
					Status: tk.StatusClosed,
				},
			},
			prefix:   "",
			isLast:   true,
			expected: "[ ] tic-001 - Parent\n[ ] tic-002 - First dep\n[x] tic-003 - Second dep\n",
		},
		{
			name: "missing dependency shows not found",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Parent",
				Status: tk.StatusOpen,
				Deps:   []string{"missing-id"},
			},
			ticketMap: map[string]*tk.Ticket{},
			prefix:    "",
			isLast:    true,
			expected:  "[ ] tic-001 - Parent\n└── [?] missing-id - (not found)\n",
		},
		{
			name: "mixed found and missing dependencies from root",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Parent",
				Status: tk.StatusOpen,
				Deps:   []string{"tic-002", "missing-id"},
			},
			ticketMap: map[string]*tk.Ticket{
				"tic-002": {
					ID:     "tic-002",
					Title:  "Found dep",
					Status: tk.StatusOpen,
				},
			},
			prefix:   "",
			isLast:   true,
			expected: "[ ] tic-001 - Parent\n[ ] tic-002 - Found dep\n└── [?] missing-id - (not found)\n",
		},
		{
			name: "deep nesting from root prints all as roots",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Root",
				Status: tk.StatusOpen,
				Deps:   []string{"tic-002"},
			},
			ticketMap: map[string]*tk.Ticket{
				"tic-002": {
					ID:     "tic-002",
					Title:  "Level 1",
					Status: tk.StatusOpen,
					Deps:   []string{"tic-003"},
				},
				"tic-003": {
					ID:     "tic-003",
					Title:  "Level 2",
					Status: tk.StatusOpen,
				},
			},
			prefix:   "",
			isLast:   true,
			expected: "[ ] tic-001 - Root\n[ ] tic-002 - Level 1\n[ ] tic-003 - Level 2\n",
		},
		{
			name: "non-root with prefix adds vertical bar for non-last",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Middle",
				Status: tk.StatusOpen,
				Deps:   []string{"tic-002"},
			},
			ticketMap: map[string]*tk.Ticket{
				"tic-002": {
					ID:     "tic-002",
					Title:  "Child",
					Status: tk.StatusOpen,
				},
			},
			prefix:   "│   ",
			isLast:   false,
			expected: "│   ├── [ ] tic-001 - Middle\n│   │   └── [ ] tic-002 - Child\n",
		},
		{
			name: "non-root with prefix and isLast adds spaces",
			ticket: &tk.Ticket{
				ID:     "tic-001",
				Title:  "Last",
				Status: tk.StatusOpen,
				Deps:   []string{"tic-002"},
			},
			ticketMap: map[string]*tk.Ticket{
				"tic-002": {
					ID:     "tic-002",
					Title:  "Child",
					Status: tk.StatusOpen,
				},
			},
			prefix:   "│   ",
			isLast:   true,
			expected: "│   └── [ ] tic-001 - Last\n│       └── [ ] tic-002 - Child\n",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			output := captureOutput(func() {
				printDepTree(tt.ticket, tt.ticketMap, tt.prefix, tt.isLast)
			})
			require.Equal(s.T(), tt.expected, output)
		})
	}
}
