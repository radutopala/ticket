package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TicketSuite struct {
	suite.Suite
}

func TestTicketSuite(t *testing.T) {
	suite.Run(t, new(TicketSuite))
}

func (s *TicketSuite) TestParseStatus() {
	tests := []struct {
		name    string
		input   string
		want    Status
		wantErr bool
	}{
		{name: "open", input: "open", want: StatusOpen},
		{name: "in_progress", input: "in_progress", want: StatusInProgress},
		{name: "closed", input: "closed", want: StatusClosed},
		{name: "invalid", input: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := ParseStatus(tt.input)
			if tt.wantErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), tt.want, got)
		})
	}
}

func (s *TicketSuite) TestStatusString() {
	require.Equal(s.T(), "open", StatusOpen.String())
	require.Equal(s.T(), "in_progress", StatusInProgress.String())
	require.Equal(s.T(), "closed", StatusClosed.String())
}

func (s *TicketSuite) TestStatusIsValid() {
	require.True(s.T(), StatusOpen.IsValid())
	require.True(s.T(), StatusInProgress.IsValid())
	require.True(s.T(), StatusClosed.IsValid())
	require.False(s.T(), Status("invalid").IsValid())
}

func (s *TicketSuite) TestParseType() {
	tests := []struct {
		name    string
		input   string
		want    Type
		wantErr bool
	}{
		{name: "task", input: "task", want: TypeTask},
		{name: "bug", input: "bug", want: TypeBug},
		{name: "feature", input: "feature", want: TypeFeature},
		{name: "epic", input: "epic", want: TypeEpic},
		{name: "chore", input: "chore", want: TypeChore},
		{name: "invalid", input: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := ParseType(tt.input)
			if tt.wantErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), tt.want, got)
		})
	}
}

func (s *TicketSuite) TestTypeString() {
	require.Equal(s.T(), "task", TypeTask.String())
	require.Equal(s.T(), "bug", TypeBug.String())
	require.Equal(s.T(), "feature", TypeFeature.String())
	require.Equal(s.T(), "epic", TypeEpic.String())
	require.Equal(s.T(), "chore", TypeChore.String())
}

func (s *TicketSuite) TestTypeIsValid() {
	require.True(s.T(), TypeTask.IsValid())
	require.True(s.T(), TypeBug.IsValid())
	require.True(s.T(), TypeFeature.IsValid())
	require.True(s.T(), TypeEpic.IsValid())
	require.True(s.T(), TypeChore.IsValid())
	require.False(s.T(), Type("invalid").IsValid())
}

func (s *TicketSuite) TestParse() {
	content := `---
id: tic-abc1
status: open
type: task
priority: 2
assignee: John Doe
deps: [tic-xyz1]
links: []
created: 2026-01-31T10:00:00Z
---
# Test Ticket

This is the description.

## Design

Design notes here.

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Notes

### 2026-01-31T11:00:00Z

First note content.

### 2026-01-31T12:00:00Z

Second note content.
`

	ticket, err := Parse([]byte(content))
	require.NoError(s.T(), err)
	require.NotNil(s.T(), ticket)

	require.Equal(s.T(), "tic-abc1", ticket.ID)
	require.Equal(s.T(), StatusOpen, ticket.Status)
	require.Equal(s.T(), TypeTask, ticket.Type)
	require.Equal(s.T(), 2, ticket.Priority)
	require.Equal(s.T(), "John Doe", ticket.Assignee)
	require.Equal(s.T(), []string{"tic-xyz1"}, ticket.Deps)
	require.Equal(s.T(), "Test Ticket", ticket.Title)
	require.Contains(s.T(), ticket.Description, "This is the description")
	require.Contains(s.T(), ticket.Design, "Design notes here")
	require.Contains(s.T(), ticket.Acceptance, "Criterion 1")
	require.Len(s.T(), ticket.Notes, 2)
	require.Contains(s.T(), ticket.Notes[0].Content, "First note content")
	require.Contains(s.T(), ticket.Notes[1].Content, "Second note content")
}

func (s *TicketSuite) TestParseInvalidFrontmatter() {
	tests := []struct {
		name    string
		content string
	}{
		{name: "no frontmatter", content: "# Title\n\nContent"},
		{name: "unclosed frontmatter", content: "---\nid: test\n# Title"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := Parse([]byte(tt.content))
			require.Error(s.T(), err)
		})
	}
}

func (s *TicketSuite) TestRender() {
	ticket := &Ticket{
		ID:       "tic-test1",
		Status:   StatusInProgress,
		Type:     TypeFeature,
		Priority: 1,
		Assignee: "Jane Doe",
		Deps:     []string{"tic-dep1"},
		Created:  time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC),
		Title:    "Feature Title",
		Description: "Feature description.",
		Design:      "Design details.",
		Acceptance:  "- [ ] Accept this",
		Notes: []Note{
			{
				Timestamp: time.Date(2026, 1, 31, 11, 0, 0, 0, time.UTC),
				Content:   "Note content",
			},
		},
	}

	rendered, err := ticket.Render()
	require.NoError(s.T(), err)

	content := string(rendered)
	require.Contains(s.T(), content, "id: tic-test1")
	require.Contains(s.T(), content, "status: in_progress")
	require.Contains(s.T(), content, "type: feature")
	require.Contains(s.T(), content, "priority: 1")
	require.Contains(s.T(), content, "# Feature Title")
	require.Contains(s.T(), content, "Feature description")
	require.Contains(s.T(), content, "## Design")
	require.Contains(s.T(), content, "## Acceptance Criteria")
	require.Contains(s.T(), content, "## Notes")
}

func (s *TicketSuite) TestRoundTrip() {
	original := &Ticket{
		ID:       "tic-round1",
		Status:   StatusOpen,
		Type:     TypeBug,
		Priority: 3,
		Assignee: "Developer",
		Tags:     []string{"urgent", "backend"},
		Deps:     []string{"tic-dep1", "tic-dep2"},
		Links:    []string{"tic-link1"},
		Created:  time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC),
		Title:    "Bug Title",
		Description: "Bug description with details.",
		Design:      "Fix approach.",
		Acceptance:  "- [ ] Bug is fixed\n- [ ] Tests pass",
	}

	rendered, err := original.Render()
	require.NoError(s.T(), err)

	parsed, err := Parse(rendered)
	require.NoError(s.T(), err)

	require.Equal(s.T(), original.ID, parsed.ID)
	require.Equal(s.T(), original.Status, parsed.Status)
	require.Equal(s.T(), original.Type, parsed.Type)
	require.Equal(s.T(), original.Priority, parsed.Priority)
	require.Equal(s.T(), original.Assignee, parsed.Assignee)
	require.Equal(s.T(), original.Tags, parsed.Tags)
	require.Equal(s.T(), original.Deps, parsed.Deps)
	require.Equal(s.T(), original.Links, parsed.Links)
	require.Equal(s.T(), original.Title, parsed.Title)
}

func TestTitlePreservationAfterStatusChange(t *testing.T) {
	content := `---
id: test-1234
status: open
type: task
priority: 2
created: 2026-01-31T17:10:46.21915Z
---
# My Test Title

## Section
- item
`
	ticket, err := Parse([]byte(content))
	require.NoError(t, err)
	
	t.Logf("Title after parse: %q", ticket.Title)
	t.Logf("Description after parse: %q", ticket.Description)
	
	require.Equal(t, "My Test Title", ticket.Title, "Title should be preserved after parse")
	
	// Simulate status change
	ticket.Status = StatusInProgress
	
	rendered, err := ticket.Render()
	require.NoError(t, err)
	
	t.Logf("Rendered:\n%s", rendered)
	
	// Parse again
	ticket2, err := Parse(rendered)
	require.NoError(t, err)
	
	require.Equal(t, "My Test Title", ticket2.Title, "Title should be preserved after render and re-parse")
}
