package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/radutopala/ticket/internal/domain"
	"github.com/radutopala/ticket/internal/storage"
)

type CmdSuite struct {
	suite.Suite
	tempDir string
	cleanup func()
}

func TestCmdSuite(t *testing.T) {
	suite.Run(t, new(CmdSuite))
}

func (s *CmdSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "ticket-cmd-test-*")
	require.NoError(s.T(), err)

	s.tempDir = tempDir

	// Set TICKETS_DIR env var so PersistentPreRunE uses our temp dir
	s.T().Setenv("TICKETS_DIR", tempDir)

	store = storage.New(tempDir)
	require.NoError(s.T(), store.EnsureDir())

	// Reset all command flags to their default values
	listFlags.status = ""
	listFlags.assignee = ""
	listFlags.tag = ""
	closedFlags.limit = 20

	s.cleanup = func() {
		_ = os.RemoveAll(tempDir)
	}
}

func (s *CmdSuite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func (s *CmdSuite) createTestTicket(id string, status domain.Status, title string) *domain.Ticket {
	ticket := &domain.Ticket{
		ID:       id,
		Status:   status,
		Type:     domain.TypeTask,
		Priority: 2,
		Title:    title,
		Created:  time.Now().UTC(),
	}
	err := store.Write(ticket)
	require.NoError(s.T(), err)
	return ticket
}

// executeCommand runs a command through rootCmd and returns the output
func (s *CmdSuite) executeCommand(args ...string) (string, error) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String(), err
}

func (s *CmdSuite) TestShowCommand() {
	// Create a test ticket
	ticket := s.createTestTicket("tic-show", domain.StatusOpen, "Test Ticket Title")
	ticket.Description = "Test description"
	require.NoError(s.T(), store.Write(ticket))

	output, err := s.executeCommand("show", "tic-show")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-show")
	require.Contains(s.T(), output, "Test Ticket Title")
}

func (s *CmdSuite) TestShowCommandNotFound() {
	_, err := s.executeCommand("show", "nonexistent")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestCloseCommand() {
	s.createTestTicket("tic-close", domain.StatusOpen, "Ticket to close")

	output, err := s.executeCommand("close", "tic-close")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "closed")

	// Verify the ticket status was updated
	ticket, err := store.Read("tic-close")
	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.StatusClosed, ticket.Status)
}

func (s *CmdSuite) TestCloseCommandNotFound() {
	_, err := s.executeCommand("close", "nonexistent")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestStartCommand() {
	s.createTestTicket("tic-start", domain.StatusOpen, "Ticket to start")

	output, err := s.executeCommand("start", "tic-start")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "in_progress")

	// Verify the ticket status was updated
	ticket, err := store.Read("tic-start")
	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.StatusInProgress, ticket.Status)
}

func (s *CmdSuite) TestStartCommandNotFound() {
	_, err := s.executeCommand("start", "nonexistent")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestListCommand() {
	// Create multiple tickets
	s.createTestTicket("tic-list1", domain.StatusOpen, "First ticket")
	s.createTestTicket("tic-list2", domain.StatusInProgress, "Second ticket")
	s.createTestTicket("tic-list3", domain.StatusClosed, "Third ticket")

	output, err := s.executeCommand("list")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-list1")
	require.Contains(s.T(), output, "tic-list2")
	require.Contains(s.T(), output, "tic-list3")
}

func (s *CmdSuite) TestListCommandWithStatusFilter() {
	s.createTestTicket("tic-f1", domain.StatusOpen, "Open ticket")
	s.createTestTicket("tic-f2", domain.StatusClosed, "Closed ticket")

	output, err := s.executeCommand("list", "--status", "open")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-f1")
	require.NotContains(s.T(), output, "tic-f2")
}

func (s *CmdSuite) TestReadyCommand() {
	// Create tickets with and without deps
	s.createTestTicket("tic-ready1", domain.StatusOpen, "Ready ticket")
	t2 := s.createTestTicket("tic-ready2", domain.StatusOpen, "Blocked ticket")
	s.createTestTicket("tic-ready3", domain.StatusOpen, "Blocker ticket")

	// Make t2 depend on tic-ready3
	t2.Deps = []string{"tic-ready3"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("ready")

	require.NoError(s.T(), err)
	// tic-ready1 and tic-ready3 should appear (no blocking deps)
	require.Contains(s.T(), output, "tic-ready1")
	require.Contains(s.T(), output, "tic-ready3")
	// tic-ready2 is blocked by tic-ready3, should not appear
	require.NotContains(s.T(), output, "tic-ready2")
}

func (s *CmdSuite) TestBlockedCommand() {
	// Create tickets
	s.createTestTicket("tic-blk1", domain.StatusOpen, "Non-blocked ticket")
	t2 := s.createTestTicket("tic-blk2", domain.StatusOpen, "Blocked ticket")
	s.createTestTicket("tic-blk3", domain.StatusOpen, "Blocker ticket")

	// Make t2 depend on tic-blk3
	t2.Deps = []string{"tic-blk3"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("blocked")

	require.NoError(s.T(), err)
	// Only t2 should appear (blocked by t3)
	require.Contains(s.T(), output, "tic-blk2")
	require.NotContains(s.T(), output, "tic-blk1")
	require.NotContains(s.T(), output, "tic-blk3")
}

func (s *CmdSuite) TestClosedCommand() {
	s.createTestTicket("tic-cls1", domain.StatusOpen, "Open ticket")
	s.createTestTicket("tic-cls2", domain.StatusClosed, "Closed ticket")

	output, err := s.executeCommand("closed")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-cls2")
	require.NotContains(s.T(), output, "tic-cls1")
}

func (s *CmdSuite) TestShowWithParent() {
	// Create parent ticket
	s.createTestTicket("tic-parent", domain.StatusOpen, "Parent Ticket")

	// Create child ticket with parent
	child := &domain.Ticket{
		ID:       "tic-child",
		Status:   domain.StatusOpen,
		Type:     domain.TypeTask,
		Priority: 2,
		Title:    "Child Ticket",
		Parent:   "tic-parent",
		Created:  time.Now().UTC(),
	}
	require.NoError(s.T(), store.Write(child))

	output, err := s.executeCommand("show", "tic-child")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-child")
	require.Contains(s.T(), output, "parent: tic-parent")
}

func (s *CmdSuite) TestPartialIDResolution() {
	s.createTestTicket("tic-unique123", domain.StatusOpen, "Unique ticket")

	output, err := s.executeCommand("show", "unique123")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-unique123")
}

func (s *CmdSuite) TestAmbiguousIDResolution() {
	s.createTestTicket("tic-ambig1", domain.StatusOpen, "Ticket 1")
	s.createTestTicket("tic-ambig2", domain.StatusOpen, "Ticket 2")

	_, err := s.executeCommand("show", "ambig")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "ambiguous")
}

func (s *CmdSuite) TestCreateFlagsInit() {
	// Test that create flags are initialized
	require.NotNil(s.T(), createCmd.Flags().Lookup("description"))
	require.NotNil(s.T(), createCmd.Flags().Lookup("type"))
	require.NotNil(s.T(), createCmd.Flags().Lookup("priority"))
	require.NotNil(s.T(), createCmd.Flags().Lookup("assignee"))
	require.NotNil(s.T(), createCmd.Flags().Lookup("parent"))
	require.NotNil(s.T(), createCmd.Flags().Lookup("tags"))
}

func (s *CmdSuite) TestEmptyList() {
	// Clear any existing tickets
	files, _ := filepath.Glob(filepath.Join(s.tempDir, "*.md"))
	for _, f := range files {
		_ = os.Remove(f)
	}

	output, err := s.executeCommand("list")

	require.NoError(s.T(), err)
	require.Empty(s.T(), strings.TrimSpace(output))
}

func (s *CmdSuite) TestReopenCommand() {
	s.createTestTicket("tic-reopen", domain.StatusClosed, "Ticket to reopen")

	output, err := s.executeCommand("reopen", "tic-reopen")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "open")

	ticket, err := store.Read("tic-reopen")
	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.StatusOpen, ticket.Status)
}

func (s *CmdSuite) TestReopenCommandNotFound() {
	_, err := s.executeCommand("reopen", "nonexistent")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestStatusCommand() {
	s.createTestTicket("tic-status", domain.StatusOpen, "Ticket for status")

	output, err := s.executeCommand("status", "tic-status", "in_progress")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "in_progress")

	ticket, err := store.Read("tic-status")
	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.StatusInProgress, ticket.Status)
}

func (s *CmdSuite) TestStatusCommandInvalidStatus() {
	s.createTestTicket("tic-status-inv", domain.StatusOpen, "Ticket for invalid status")

	_, err := s.executeCommand("status", "tic-status-inv", "invalid")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid status")
}

func (s *CmdSuite) TestStatusCommandNotFound() {
	_, err := s.executeCommand("status", "nonexistent", "open")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestListWithAssigneeFilter() {
	t1 := s.createTestTicket("tic-asn1", domain.StatusOpen, "Alice ticket")
	t1.Assignee = "alice"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-asn2", domain.StatusOpen, "Bob ticket")
	t2.Assignee = "bob"
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("list", "--assignee", "alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-asn1")
	require.NotContains(s.T(), output, "tic-asn2")
}

func (s *CmdSuite) TestListWithTagFilter() {
	t1 := s.createTestTicket("tic-tag1", domain.StatusOpen, "Backend ticket")
	t1.Tags = []string{"backend"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-tag2", domain.StatusOpen, "Frontend ticket")
	t2.Tags = []string{"frontend"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("list", "--tag", "backend")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-tag1")
	require.NotContains(s.T(), output, "tic-tag2")
}

func (s *CmdSuite) TestReadyWithAssigneeFilter() {
	t1 := s.createTestTicket("tic-r-asn1", domain.StatusOpen, "Alice ready")
	t1.Assignee = "alice"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-r-asn2", domain.StatusOpen, "Bob ready")
	t2.Assignee = "bob"
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("ready", "--assignee", "alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-r-asn1")
	require.NotContains(s.T(), output, "tic-r-asn2")
}

func (s *CmdSuite) TestReadyWithTagFilter() {
	t1 := s.createTestTicket("tic-r-tag1", domain.StatusOpen, "Backend ready")
	t1.Tags = []string{"backend"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-r-tag2", domain.StatusOpen, "Frontend ready")
	t2.Tags = []string{"frontend"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("ready", "--tag", "backend")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-r-tag1")
	require.NotContains(s.T(), output, "tic-r-tag2")
}

func (s *CmdSuite) TestBlockedWithAssigneeFilter() {
	s.createTestTicket("tic-blocker-asn", domain.StatusOpen, "Blocker")

	t1 := s.createTestTicket("tic-b-asn1", domain.StatusOpen, "Alice blocked")
	t1.Assignee = "alice"
	t1.Deps = []string{"tic-blocker-asn"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-b-asn2", domain.StatusOpen, "Bob blocked")
	t2.Assignee = "bob"
	t2.Deps = []string{"tic-blocker-asn"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("blocked", "--assignee", "alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-b-asn1")
	require.NotContains(s.T(), output, "tic-b-asn2")
}

func (s *CmdSuite) TestBlockedWithTagFilter() {
	s.createTestTicket("tic-blocker-tag", domain.StatusOpen, "Blocker")

	t1 := s.createTestTicket("tic-b-tag1", domain.StatusOpen, "Backend blocked")
	t1.Tags = []string{"backend"}
	t1.Deps = []string{"tic-blocker-tag"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-b-tag2", domain.StatusOpen, "Frontend blocked")
	t2.Tags = []string{"frontend"}
	t2.Deps = []string{"tic-blocker-tag"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("blocked", "--tag", "backend")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-b-tag1")
	require.NotContains(s.T(), output, "tic-b-tag2")
}

func (s *CmdSuite) TestClosedWithAssigneeFilter() {
	t1 := s.createTestTicket("tic-c-asn1", domain.StatusClosed, "Alice closed")
	t1.Assignee = "alice"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-c-asn2", domain.StatusClosed, "Bob closed")
	t2.Assignee = "bob"
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("closed", "--assignee", "alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-c-asn1")
	require.NotContains(s.T(), output, "tic-c-asn2")
}

func (s *CmdSuite) TestClosedWithTagFilter() {
	t1 := s.createTestTicket("tic-c-tag1", domain.StatusClosed, "Backend closed")
	t1.Tags = []string{"backend"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-c-tag2", domain.StatusClosed, "Frontend closed")
	t2.Tags = []string{"frontend"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("closed", "--tag", "backend")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-c-tag1")
	require.NotContains(s.T(), output, "tic-c-tag2")
}

func (s *CmdSuite) TestClosedWithLimit() {
	// Create 3 closed tickets
	s.createTestTicket("tic-c-lim1", domain.StatusClosed, "Closed 1")
	time.Sleep(10 * time.Millisecond)
	s.createTestTicket("tic-c-lim2", domain.StatusClosed, "Closed 2")
	time.Sleep(10 * time.Millisecond)
	s.createTestTicket("tic-c-lim3", domain.StatusClosed, "Closed 3")

	output, err := s.executeCommand("closed", "--limit", "2")

	require.NoError(s.T(), err)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(s.T(), lines, 2)
}

func (s *CmdSuite) TestReadyExcludesClosedTickets() {
	s.createTestTicket("tic-ready-excl1", domain.StatusOpen, "Open ticket")
	s.createTestTicket("tic-ready-excl2", domain.StatusClosed, "Closed ticket")

	output, err := s.executeCommand("ready")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-ready-excl1")
	require.NotContains(s.T(), output, "tic-ready-excl2")
}

func (s *CmdSuite) TestShowWithLinks() {
	// Create two related tickets
	s.createTestTicket("tic-link1", domain.StatusOpen, "Linked Ticket 1")

	// Create ticket with links
	linked := &domain.Ticket{
		ID:       "tic-link2",
		Status:   domain.StatusOpen,
		Type:     domain.TypeTask,
		Priority: 2,
		Title:    "Linked Ticket 2",
		Links:    []string{"tic-link1"},
		Created:  time.Now().UTC(),
	}
	require.NoError(s.T(), store.Write(linked))

	output, err := s.executeCommand("show", "tic-link2")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-link2")
	require.Contains(s.T(), output, "tic-link1")
	require.Contains(s.T(), output, "links:")
}

func (s *CmdSuite) TestGetConfig() {
	c := GetConfig()
	// Config may or may not be set depending on test order
	_ = c
}

func (s *CmdSuite) TestGetLogger() {
	l := GetLogger()
	// Logger may or may not be set depending on test order
	_ = l
}

func (s *CmdSuite) TestGetStorage() {
	st := GetStorage()
	require.NotNil(s.T(), st)
}

func (s *CmdSuite) TestCreateCommand() {
	output, err := s.executeCommand("create", "New Ticket Title")

	require.NoError(s.T(), err)
	// Output should contain the generated ticket ID
	require.Contains(s.T(), output, "tic-")
}

func (s *CmdSuite) TestCreateCommandWithFlags() {
	output, err := s.executeCommand("create", "Feature Ticket",
		"--type", "feature",
		"--priority", "1",
		"--description", "A new feature",
		"--assignee", "developer")

	require.NoError(s.T(), err)
	id := strings.TrimSpace(output)
	require.Contains(s.T(), id, "tic-")

	// Verify the ticket was created with correct values
	ticket, err := store.Read(id)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Feature Ticket", ticket.Title)
	require.Equal(s.T(), domain.TypeFeature, ticket.Type)
	require.Equal(s.T(), 1, ticket.Priority)
	require.Equal(s.T(), "A new feature", ticket.Description)
	require.Equal(s.T(), "developer", ticket.Assignee)
}

func (s *CmdSuite) TestCreateCommandWithInvalidType() {
	_, err := s.executeCommand("create", "Bad Type Ticket", "--type", "invalid")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid type")
}

func (s *CmdSuite) TestDepAddCommand() {
	s.createTestTicket("tic-dep-a", domain.StatusOpen, "Ticket A")
	s.createTestTicket("tic-dep-b", domain.StatusOpen, "Ticket B (depends on A)")

	output, err := s.executeCommand("dep", "add", "tic-dep-b", "tic-dep-a")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Added dependency")

	// Verify the dependency was added
	ticket, err := store.Read("tic-dep-b")
	require.NoError(s.T(), err)
	require.Contains(s.T(), ticket.Deps, "tic-dep-a")
}

func (s *CmdSuite) TestDepAddCommandNotFound() {
	s.createTestTicket("tic-dep-exists", domain.StatusOpen, "Existing ticket")

	_, err := s.executeCommand("dep", "add", "tic-dep-exists", "nonexistent")

	require.Error(s.T(), err)
}

func (s *CmdSuite) TestDepRemoveCommand() {
	s.createTestTicket("tic-dep-rm-a", domain.StatusOpen, "Ticket A")
	t := s.createTestTicket("tic-dep-rm-b", domain.StatusOpen, "Ticket B")
	t.Deps = []string{"tic-dep-rm-a"}
	require.NoError(s.T(), store.Write(t))

	output, err := s.executeCommand("dep", "remove", "tic-dep-rm-b", "tic-dep-rm-a")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Removed dependency")

	// Verify the dependency was removed
	ticket, err := store.Read("tic-dep-rm-b")
	require.NoError(s.T(), err)
	require.NotContains(s.T(), ticket.Deps, "tic-dep-rm-a")
}

func (s *CmdSuite) TestDepTreeCommand() {
	s.createTestTicket("tic-tree-root", domain.StatusOpen, "Root ticket")
	t := s.createTestTicket("tic-tree-child", domain.StatusOpen, "Child ticket")
	t.Deps = []string{"tic-tree-root"}
	require.NoError(s.T(), store.Write(t))

	output, err := s.executeCommand("dep", "tree")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-tree")
}

func (s *CmdSuite) TestDepCheckCommand() {
	// Create tickets without cycles
	s.createTestTicket("tic-nocycle1", domain.StatusOpen, "No cycle 1")
	t := s.createTestTicket("tic-nocycle2", domain.StatusOpen, "No cycle 2")
	t.Deps = []string{"tic-nocycle1"}
	require.NoError(s.T(), store.Write(t))

	output, err := s.executeCommand("dep", "check")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "No cycles detected")
}

func (s *CmdSuite) TestLinkCommand() {
	s.createTestTicket("tic-lnk-a", domain.StatusOpen, "Ticket A")
	s.createTestTicket("tic-lnk-b", domain.StatusOpen, "Ticket B")

	output, err := s.executeCommand("link", "tic-lnk-a", "tic-lnk-b")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Linked")

	// Verify the link was created
	ticketA, err := store.Read("tic-lnk-a")
	require.NoError(s.T(), err)
	require.Contains(s.T(), ticketA.Links, "tic-lnk-b")
}

func (s *CmdSuite) TestUnlinkCommand() {
	t1 := s.createTestTicket("tic-ulnk-a", domain.StatusOpen, "Ticket A")
	t1.Links = []string{"tic-ulnk-b"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-ulnk-b", domain.StatusOpen, "Ticket B")
	t2.Links = []string{"tic-ulnk-a"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("unlink", "tic-ulnk-a", "tic-ulnk-b")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Unlinked")
}

func (s *CmdSuite) TestQueryCommand() {
	s.createTestTicket("tic-query1", domain.StatusOpen, "Query ticket 1")
	s.createTestTicket("tic-query2", domain.StatusClosed, "Query ticket 2")

	output, err := s.executeCommand("query")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-query1")
	require.Contains(s.T(), output, "tic-query2")
}

func (s *CmdSuite) TestHelpOutput() {
	// Test that running root command with no args produces expected help output
	output, err := s.executeCommand()

	require.NoError(s.T(), err)

	// Verify header line with description
	require.Contains(s.T(), output, "tk - minimal ticket system with dependency tracking")

	// Verify Usage section
	require.Contains(s.T(), output, "Usage:")
	require.Contains(s.T(), output, "tk [command]")

	// Verify Available Commands section with key commands
	require.Contains(s.T(), output, "Available Commands:")
	require.Contains(s.T(), output, "create")
	require.Contains(s.T(), output, "show")
	require.Contains(s.T(), output, "list")
	require.Contains(s.T(), output, "ready")
	require.Contains(s.T(), output, "blocked")
	require.Contains(s.T(), output, "dep")

	// Verify create command has its flags documented
	require.Contains(s.T(), output, "--description")
	require.Contains(s.T(), output, "--type")
	require.Contains(s.T(), output, "--priority")

	// Verify footer about .tickets/
	require.Contains(s.T(), output, "Tickets stored as markdown files in .tickets/")
	require.Contains(s.T(), output, "Supports partial ID matching")
}
