package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	tk "github.com/radutopala/ticket/pkg/ticket"
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

	store = tk.OpenDir(tempDir)
	require.NoError(s.T(), store.EnsureDir())

	// Reset all command flags to their default values
	jsonOutput = false
	listFlags.Status = ""
	listFlags.Assignee = ""
	listFlags.Tag = ""
	closedFlags.limit = 20
	createFlags.description = ""
	createFlags.design = ""
	createFlags.acceptance = ""
	createFlags.ticketType = ""
	createFlags.priority = 2
	createFlags.assignee = ""
	createFlags.externalRef = ""
	createFlags.parent = ""
	createFlags.tags = nil
	exportFlags.format = "json"
	exportFlags.output = ""
	importFlags.skipExisting = false
	bulkFlags.tag = ""
	bulkFlags.status = ""
	bulkFlags.assignee = ""
	bulkFlags.dryRun = false

	s.cleanup = func() {
		_ = os.RemoveAll(tempDir)
	}
}

func (s *CmdSuite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func (s *CmdSuite) createTestTicket(id string, status tk.Status, title string) *tk.Ticket {
	ticket := &tk.Ticket{
		ID:       id,
		Status:   status,
		Type:     tk.TypeTask,
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
	ticket := s.createTestTicket("tic-show", tk.StatusOpen, "Test Ticket Title")
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
	s.createTestTicket("tic-close", tk.StatusOpen, "Ticket to close")

	output, err := s.executeCommand("close", "tic-close")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "closed")

	// Verify the ticket status was updated
	ticket, err := store.Read("tic-close")
	require.NoError(s.T(), err)
	require.Equal(s.T(), tk.StatusClosed, ticket.Status)
}

func (s *CmdSuite) TestCloseCommandNotFound() {
	_, err := s.executeCommand("close", "nonexistent")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestStartCommand() {
	s.createTestTicket("tic-start", tk.StatusOpen, "Ticket to start")

	output, err := s.executeCommand("start", "tic-start")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "in_progress")

	// Verify the ticket status was updated
	ticket, err := store.Read("tic-start")
	require.NoError(s.T(), err)
	require.Equal(s.T(), tk.StatusInProgress, ticket.Status)
}

func (s *CmdSuite) TestStartCommandNotFound() {
	_, err := s.executeCommand("start", "nonexistent")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestListCommand() {
	// Create multiple tickets
	s.createTestTicket("tic-list1", tk.StatusOpen, "First ticket")
	s.createTestTicket("tic-list2", tk.StatusInProgress, "Second ticket")
	s.createTestTicket("tic-list3", tk.StatusClosed, "Third ticket")

	output, err := s.executeCommand("list")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-list1")
	require.Contains(s.T(), output, "tic-list2")
	require.Contains(s.T(), output, "tic-list3")
}

func (s *CmdSuite) TestListCommandWithStatusFilter() {
	s.createTestTicket("tic-f1", tk.StatusOpen, "Open ticket")
	s.createTestTicket("tic-f2", tk.StatusClosed, "Closed ticket")

	output, err := s.executeCommand("list", "--status", "open")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-f1")
	require.NotContains(s.T(), output, "tic-f2")
}

func (s *CmdSuite) TestReadyCommand() {
	// Create tickets with and without deps
	s.createTestTicket("tic-ready1", tk.StatusOpen, "Ready ticket")
	t2 := s.createTestTicket("tic-ready2", tk.StatusOpen, "Blocked ticket")
	s.createTestTicket("tic-ready3", tk.StatusOpen, "Blocker ticket")

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
	s.createTestTicket("tic-blk1", tk.StatusOpen, "Non-blocked ticket")
	t2 := s.createTestTicket("tic-blk2", tk.StatusOpen, "Blocked ticket")
	s.createTestTicket("tic-blk3", tk.StatusOpen, "Blocker ticket")

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
	s.createTestTicket("tic-cls1", tk.StatusOpen, "Open ticket")
	s.createTestTicket("tic-cls2", tk.StatusClosed, "Closed ticket")

	output, err := s.executeCommand("closed")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-cls2")
	require.NotContains(s.T(), output, "tic-cls1")
}

func (s *CmdSuite) TestShowWithParent() {
	// Create parent ticket
	s.createTestTicket("tic-parent", tk.StatusOpen, "Parent Ticket")

	// Create child ticket with parent
	child := &tk.Ticket{
		ID:       "tic-child",
		Status:   tk.StatusOpen,
		Type:     tk.TypeTask,
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
	s.createTestTicket("tic-unique123", tk.StatusOpen, "Unique ticket")

	output, err := s.executeCommand("show", "unique123")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-unique123")
}

func (s *CmdSuite) TestAmbiguousIDResolution() {
	s.createTestTicket("tic-ambig1", tk.StatusOpen, "Ticket 1")
	s.createTestTicket("tic-ambig2", tk.StatusOpen, "Ticket 2")

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
	s.createTestTicket("tic-reopen", tk.StatusClosed, "Ticket to reopen")

	output, err := s.executeCommand("reopen", "tic-reopen")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "open")

	ticket, err := store.Read("tic-reopen")
	require.NoError(s.T(), err)
	require.Equal(s.T(), tk.StatusOpen, ticket.Status)
}

func (s *CmdSuite) TestReopenCommandNotFound() {
	_, err := s.executeCommand("reopen", "nonexistent")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestStatusCommand() {
	s.createTestTicket("tic-status", tk.StatusOpen, "Ticket for status")

	output, err := s.executeCommand("status", "tic-status", "in_progress")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "in_progress")

	ticket, err := store.Read("tic-status")
	require.NoError(s.T(), err)
	require.Equal(s.T(), tk.StatusInProgress, ticket.Status)
}

func (s *CmdSuite) TestStatusCommandInvalidStatus() {
	s.createTestTicket("tic-status-inv", tk.StatusOpen, "Ticket for invalid status")

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
	t1 := s.createTestTicket("tic-asn1", tk.StatusOpen, "Alice ticket")
	t1.Assignee = "alice"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-asn2", tk.StatusOpen, "Bob ticket")
	t2.Assignee = "bob"
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("list", "--assignee", "alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-asn1")
	require.NotContains(s.T(), output, "tic-asn2")
}

func (s *CmdSuite) TestListWithTagFilter() {
	t1 := s.createTestTicket("tic-tag1", tk.StatusOpen, "Backend ticket")
	t1.Tags = []string{"backend"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-tag2", tk.StatusOpen, "Frontend ticket")
	t2.Tags = []string{"frontend"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("list", "--tag", "backend")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-tag1")
	require.NotContains(s.T(), output, "tic-tag2")
}

func (s *CmdSuite) TestReadyWithAssigneeFilter() {
	t1 := s.createTestTicket("tic-r-asn1", tk.StatusOpen, "Alice ready")
	t1.Assignee = "alice"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-r-asn2", tk.StatusOpen, "Bob ready")
	t2.Assignee = "bob"
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("ready", "--assignee", "alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-r-asn1")
	require.NotContains(s.T(), output, "tic-r-asn2")
}

func (s *CmdSuite) TestReadyWithTagFilter() {
	t1 := s.createTestTicket("tic-r-tag1", tk.StatusOpen, "Backend ready")
	t1.Tags = []string{"backend"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-r-tag2", tk.StatusOpen, "Frontend ready")
	t2.Tags = []string{"frontend"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("ready", "--tag", "backend")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-r-tag1")
	require.NotContains(s.T(), output, "tic-r-tag2")
}

func (s *CmdSuite) TestBlockedWithAssigneeFilter() {
	s.createTestTicket("tic-blocker-asn", tk.StatusOpen, "Blocker")

	t1 := s.createTestTicket("tic-b-asn1", tk.StatusOpen, "Alice blocked")
	t1.Assignee = "alice"
	t1.Deps = []string{"tic-blocker-asn"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-b-asn2", tk.StatusOpen, "Bob blocked")
	t2.Assignee = "bob"
	t2.Deps = []string{"tic-blocker-asn"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("blocked", "--assignee", "alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-b-asn1")
	require.NotContains(s.T(), output, "tic-b-asn2")
}

func (s *CmdSuite) TestBlockedWithTagFilter() {
	s.createTestTicket("tic-blocker-tag", tk.StatusOpen, "Blocker")

	t1 := s.createTestTicket("tic-b-tag1", tk.StatusOpen, "Backend blocked")
	t1.Tags = []string{"backend"}
	t1.Deps = []string{"tic-blocker-tag"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-b-tag2", tk.StatusOpen, "Frontend blocked")
	t2.Tags = []string{"frontend"}
	t2.Deps = []string{"tic-blocker-tag"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("blocked", "--tag", "backend")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-b-tag1")
	require.NotContains(s.T(), output, "tic-b-tag2")
}

func (s *CmdSuite) TestClosedWithAssigneeFilter() {
	t1 := s.createTestTicket("tic-c-asn1", tk.StatusClosed, "Alice closed")
	t1.Assignee = "alice"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-c-asn2", tk.StatusClosed, "Bob closed")
	t2.Assignee = "bob"
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("closed", "--assignee", "alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-c-asn1")
	require.NotContains(s.T(), output, "tic-c-asn2")
}

func (s *CmdSuite) TestClosedWithTagFilter() {
	t1 := s.createTestTicket("tic-c-tag1", tk.StatusClosed, "Backend closed")
	t1.Tags = []string{"backend"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-c-tag2", tk.StatusClosed, "Frontend closed")
	t2.Tags = []string{"frontend"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("closed", "--tag", "backend")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-c-tag1")
	require.NotContains(s.T(), output, "tic-c-tag2")
}

func (s *CmdSuite) TestClosedWithLimit() {
	// Create 3 closed tickets
	s.createTestTicket("tic-c-lim1", tk.StatusClosed, "Closed 1")
	time.Sleep(10 * time.Millisecond)
	s.createTestTicket("tic-c-lim2", tk.StatusClosed, "Closed 2")
	time.Sleep(10 * time.Millisecond)
	s.createTestTicket("tic-c-lim3", tk.StatusClosed, "Closed 3")

	output, err := s.executeCommand("closed", "--limit", "2")

	require.NoError(s.T(), err)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(s.T(), lines, 2)
}

func (s *CmdSuite) TestReadyExcludesClosedTickets() {
	s.createTestTicket("tic-ready-excl1", tk.StatusOpen, "Open ticket")
	s.createTestTicket("tic-ready-excl2", tk.StatusClosed, "Closed ticket")

	output, err := s.executeCommand("ready")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-ready-excl1")
	require.NotContains(s.T(), output, "tic-ready-excl2")
}

func (s *CmdSuite) TestShowWithLinks() {
	// Create two related tickets
	s.createTestTicket("tic-link1", tk.StatusOpen, "Linked Ticket 1")

	// Create ticket with links
	linked := &tk.Ticket{
		ID:       "tic-link2",
		Status:   tk.StatusOpen,
		Type:     tk.TypeTask,
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
	require.Equal(s.T(), tk.TypeFeature, ticket.Type)
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
	s.createTestTicket("tic-dep-a", tk.StatusOpen, "Ticket A")
	s.createTestTicket("tic-dep-b", tk.StatusOpen, "Ticket B (depends on A)")

	output, err := s.executeCommand("dep", "add", "tic-dep-b", "tic-dep-a")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Added dependency")

	// Verify the dependency was added
	ticket, err := store.Read("tic-dep-b")
	require.NoError(s.T(), err)
	require.Contains(s.T(), ticket.Deps, "tic-dep-a")
}

func (s *CmdSuite) TestDepAddCommandNotFound() {
	s.createTestTicket("tic-dep-exists", tk.StatusOpen, "Existing ticket")

	_, err := s.executeCommand("dep", "add", "tic-dep-exists", "nonexistent")

	require.Error(s.T(), err)
}

func (s *CmdSuite) TestDepRemoveCommand() {
	s.createTestTicket("tic-dep-rm-a", tk.StatusOpen, "Ticket A")
	t := s.createTestTicket("tic-dep-rm-b", tk.StatusOpen, "Ticket B")
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
	s.createTestTicket("tic-tree-root", tk.StatusOpen, "Root ticket")
	t := s.createTestTicket("tic-tree-child", tk.StatusOpen, "Child ticket")
	t.Deps = []string{"tic-tree-root"}
	require.NoError(s.T(), store.Write(t))

	output, err := s.executeCommand("dep", "tree")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-tree")
}

func (s *CmdSuite) TestDepCheckCommand() {
	// Create tickets without cycles
	s.createTestTicket("tic-nocycle1", tk.StatusOpen, "No cycle 1")
	t := s.createTestTicket("tic-nocycle2", tk.StatusOpen, "No cycle 2")
	t.Deps = []string{"tic-nocycle1"}
	require.NoError(s.T(), store.Write(t))

	output, err := s.executeCommand("dep", "check")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "No cycles detected")
}

func (s *CmdSuite) TestLinkCommand() {
	s.createTestTicket("tic-lnk-a", tk.StatusOpen, "Ticket A")
	s.createTestTicket("tic-lnk-b", tk.StatusOpen, "Ticket B")

	output, err := s.executeCommand("link", "tic-lnk-a", "tic-lnk-b")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Linked")

	// Verify the link was created
	ticketA, err := store.Read("tic-lnk-a")
	require.NoError(s.T(), err)
	require.Contains(s.T(), ticketA.Links, "tic-lnk-b")
}

func (s *CmdSuite) TestUnlinkCommand() {
	t1 := s.createTestTicket("tic-ulnk-a", tk.StatusOpen, "Ticket A")
	t1.Links = []string{"tic-ulnk-b"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-ulnk-b", tk.StatusOpen, "Ticket B")
	t2.Links = []string{"tic-ulnk-a"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("unlink", "tic-ulnk-a", "tic-ulnk-b")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Unlinked")
}

func (s *CmdSuite) TestQueryCommand() {
	s.createTestTicket("tic-query1", tk.StatusOpen, "Query ticket 1")
	s.createTestTicket("tic-query2", tk.StatusClosed, "Query ticket 2")

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

func (s *CmdSuite) TestGetGitUserName() {
	// Test that getGitUserName returns a value (assuming git is configured in test env)
	name := getGitUserName()
	// In a typical dev environment, git user.name should be set
	// We just check that the function doesn't panic and returns something reasonable
	_ = name
}

func (s *CmdSuite) TestCreateWithoutAssigneeUsesGitUserName() {
	// Create a ticket without specifying assignee
	output, err := s.executeCommand("create", "Test Ticket No Assignee")

	require.NoError(s.T(), err)
	id := strings.TrimSpace(output)
	require.Contains(s.T(), id, "tic-")

	// Read the ticket and check if assignee was set to git user.name
	ticket, err := store.Read(id)
	require.NoError(s.T(), err)

	// The assignee should be set to the git user.name (or empty if git is not configured)
	gitUserName := getGitUserName()
	require.Equal(s.T(), gitUserName, ticket.Assignee)
}

func (s *CmdSuite) TestCreateWithExplicitAssigneeOverridesGitUserName() {
	// Create a ticket with explicit assignee
	output, err := s.executeCommand("create", "Test Ticket Explicit Assignee",
		"--assignee", "explicit-user")

	require.NoError(s.T(), err)
	id := strings.TrimSpace(output)
	require.Contains(s.T(), id, "tic-")

	// Read the ticket and verify explicit assignee takes priority
	ticket, err := store.Read(id)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "explicit-user", ticket.Assignee)
}

func (s *CmdSuite) TestCreateWithInvalidPriorityTooLow() {
	_, err := s.executeCommand("create", "Test Ticket", "--priority", "-1")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid priority")
	require.Contains(s.T(), err.Error(), "must be between 0 and 4")
}

func (s *CmdSuite) TestCreateWithInvalidPriorityTooHigh() {
	_, err := s.executeCommand("create", "Test Ticket", "--priority", "5")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid priority")
	require.Contains(s.T(), err.Error(), "must be between 0 and 4")
}

func (s *CmdSuite) TestCreateWithValidPriorityBoundaries() {
	// Test priority 0 (lowest valid value, highest priority)
	output, err := s.executeCommand("create", "Priority 0 Ticket", "--priority", "0")
	require.NoError(s.T(), err)
	id := strings.TrimSpace(output)
	ticket, err := store.Read(id)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 0, ticket.Priority)

	// Test priority 4 (highest valid value, lowest priority)
	output, err = s.executeCommand("create", "Priority 4 Ticket", "--priority", "4")
	require.NoError(s.T(), err)
	id = strings.TrimSpace(output)
	ticket, err = store.Read(id)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 4, ticket.Priority)
}

func (s *CmdSuite) TestCreateWithNonExistentParent() {
	_, err := s.executeCommand("create", "Child Ticket", "--parent", "nonexistent-parent")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "parent ticket not found")
}

func (s *CmdSuite) TestCreateWithValidParent() {
	// Create a parent ticket first
	s.createTestTicket("tic-parent-create", tk.StatusOpen, "Parent Ticket")

	// Create a child ticket with valid parent
	output, err := s.executeCommand("create", "Child Ticket", "--parent", "tic-parent-create")

	require.NoError(s.T(), err)
	id := strings.TrimSpace(output)
	ticket, err := store.Read(id)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "tic-parent-create", ticket.Parent)
}

func (s *CmdSuite) TestCreateWithValidParentPartialID() {
	// Create a parent ticket first
	s.createTestTicket("tic-parent-partial", tk.StatusOpen, "Parent Ticket")

	// Create a child ticket using partial parent ID
	output, err := s.executeCommand("create", "Child Ticket", "--parent", "parent-partial")

	require.NoError(s.T(), err)
	id := strings.TrimSpace(output)
	ticket, err := store.Read(id)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "tic-parent-partial", ticket.Parent)
}

func (s *CmdSuite) TestCreateWithExternalRef() {
	output, err := s.executeCommand("create", "External Ref Ticket", "--external-ref", "gh-123")

	require.NoError(s.T(), err)
	id := strings.TrimSpace(output)
	ticket, err := store.Read(id)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "gh-123", ticket.ExternalRef)
}

func (s *CmdSuite) TestCreateWithTags() {
	output, err := s.executeCommand("create", "Tagged Ticket", "--tags", "backend,urgent,api")

	require.NoError(s.T(), err)
	id := strings.TrimSpace(output)
	ticket, err := store.Read(id)
	require.NoError(s.T(), err)
	require.Equal(s.T(), []string{"backend", "urgent", "api"}, ticket.Tags)
}

func (s *CmdSuite) TestDepTreeFullFlag() {
	// Create a chain of dependencies
	s.createTestTicket("tic-tree-full-a", tk.StatusOpen, "Tree A")
	t := s.createTestTicket("tic-tree-full-b", tk.StatusOpen, "Tree B")
	t.Deps = []string{"tic-tree-full-a"}
	require.NoError(s.T(), store.Write(t))

	output, err := s.executeCommand("dep", "tree", "--full")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-tree-full")
}

func (s *CmdSuite) TestDepTreeForSpecificTicket() {
	s.createTestTicket("tic-tree-spec-a", tk.StatusOpen, "Spec A")
	t := s.createTestTicket("tic-tree-spec-b", tk.StatusOpen, "Spec B")
	t.Deps = []string{"tic-tree-spec-a"}
	require.NoError(s.T(), store.Write(t))

	output, err := s.executeCommand("dep", "tree", "tic-tree-spec-b")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-tree-spec-b")
	require.Contains(s.T(), output, "tic-tree-spec-a")
}

func (s *CmdSuite) TestUndepCommand() {
	s.createTestTicket("tic-undep-a", tk.StatusOpen, "Undep A")
	t := s.createTestTicket("tic-undep-b", tk.StatusOpen, "Undep B")
	t.Deps = []string{"tic-undep-a"}
	require.NoError(s.T(), store.Write(t))

	output, err := s.executeCommand("undep", "tic-undep-b", "tic-undep-a")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Removed dependency")

	// Verify the dependency was removed
	ticket, err := store.Read("tic-undep-b")
	require.NoError(s.T(), err)
	require.NotContains(s.T(), ticket.Deps, "tic-undep-a")
}

func (s *CmdSuite) TestAddNoteCommand() {
	s.createTestTicket("tic-note1", tk.StatusOpen, "Note Test Ticket")

	output, err := s.executeCommand("add-note", "tic-note1", "This is a test note")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Added note")

	// Verify the note was added
	ticket, err := store.Read("tic-note1")
	require.NoError(s.T(), err)
	require.Len(s.T(), ticket.Notes, 1)
	require.Contains(s.T(), ticket.Notes[0].Content, "This is a test note")
}

func (s *CmdSuite) TestAddNoteCommandNotFound() {
	_, err := s.executeCommand("add-note", "nonexistent", "Note text")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *CmdSuite) TestAddNoteCommandMultipleNotes() {
	s.createTestTicket("tic-note2", tk.StatusOpen, "Multiple Notes Ticket")

	_, err := s.executeCommand("add-note", "tic-note2", "First note")
	require.NoError(s.T(), err)

	_, err = s.executeCommand("add-note", "tic-note2", "Second note")
	require.NoError(s.T(), err)

	ticket, err := store.Read("tic-note2")
	require.NoError(s.T(), err)
	require.Len(s.T(), ticket.Notes, 2)
	require.Contains(s.T(), ticket.Notes[0].Content, "First note")
	require.Contains(s.T(), ticket.Notes[1].Content, "Second note")
}

func (s *CmdSuite) TestDepCheckWithCycle() {
	// Create tickets with a cycle
	t1 := s.createTestTicket("tic-cycle1", tk.StatusOpen, "Cycle 1")
	t2 := s.createTestTicket("tic-cycle2", tk.StatusOpen, "Cycle 2")

	t1.Deps = []string{"tic-cycle2"}
	require.NoError(s.T(), store.Write(t1))
	t2.Deps = []string{"tic-cycle1"}
	require.NoError(s.T(), store.Write(t2))

	_, err := s.executeCommand("dep", "check")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "cycle")
}

func (s *CmdSuite) TestLinkMultipleTickets() {
	s.createTestTicket("tic-mlink1", tk.StatusOpen, "Multi Link 1")
	s.createTestTicket("tic-mlink2", tk.StatusOpen, "Multi Link 2")
	s.createTestTicket("tic-mlink3", tk.StatusOpen, "Multi Link 3")

	output, err := s.executeCommand("link", "tic-mlink1", "tic-mlink2", "tic-mlink3")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Linked")

	// Verify all tickets are linked to each other
	t1, err := store.Read("tic-mlink1")
	require.NoError(s.T(), err)
	require.Contains(s.T(), t1.Links, "tic-mlink2")
	require.Contains(s.T(), t1.Links, "tic-mlink3")

	t2, err := store.Read("tic-mlink2")
	require.NoError(s.T(), err)
	require.Contains(s.T(), t2.Links, "tic-mlink1")
	require.Contains(s.T(), t2.Links, "tic-mlink3")
}

func (s *CmdSuite) TestExportCommandJSON() {
	s.createTestTicket("tic-exp1", tk.StatusOpen, "Export Test 1")
	s.createTestTicket("tic-exp2", tk.StatusClosed, "Export Test 2")

	output, err := s.executeCommand("export")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "tic-exp1")
	require.Contains(s.T(), output, "tic-exp2")
	require.Contains(s.T(), output, "Export Test 1")
	require.Contains(s.T(), output, "Export Test 2")
	// Should be valid JSON (contains brackets)
	require.Contains(s.T(), output, "[")
	require.Contains(s.T(), output, "]")
}

func (s *CmdSuite) TestExportCommandCSV() {
	t1 := s.createTestTicket("tic-expcsv1", tk.StatusOpen, "CSV Export 1")
	t1.Tags = []string{"tag1", "tag2"}
	require.NoError(s.T(), store.Write(t1))

	output, err := s.executeCommand("export", "--format=csv")

	require.NoError(s.T(), err)
	// Check CSV header
	require.Contains(s.T(), output, "ID,Status,Type")
	// Check data row
	require.Contains(s.T(), output, "tic-expcsv1")
	require.Contains(s.T(), output, "CSV Export 1")
	// Tags should be semicolon-separated
	require.Contains(s.T(), output, "tag1;tag2")
}

func (s *CmdSuite) TestExportCommandToFile() {
	s.createTestTicket("tic-expfile", tk.StatusOpen, "File Export Test")

	outputFile := filepath.Join(s.tempDir, "export.json")
	_, err := s.executeCommand("export", "--output="+outputFile)

	require.NoError(s.T(), err)

	// Verify file was created
	data, err := os.ReadFile(outputFile)
	require.NoError(s.T(), err)
	require.Contains(s.T(), string(data), "tic-expfile")
	require.Contains(s.T(), string(data), "File Export Test")
}

func (s *CmdSuite) TestExportCommandInvalidFormat() {
	_, err := s.executeCommand("export", "--format=xml")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "unsupported format")
}

func (s *CmdSuite) TestImportCommandJSON() {
	// Create a JSON file with tickets to import
	importData := `[
		{
			"ID": "tic-imp1",
			"Status": "open",
			"Type": "task",
			"Priority": 1,
			"Title": "Imported Ticket 1",
			"Description": "Test import"
		},
		{
			"ID": "tic-imp2",
			"Status": "closed",
			"Type": "bug",
			"Priority": 0,
			"Title": "Imported Ticket 2"
		}
	]`

	importFile := filepath.Join(s.tempDir, "import.json")
	err := os.WriteFile(importFile, []byte(importData), 0644)
	require.NoError(s.T(), err)

	output, err := s.executeCommand("import", importFile)

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Imported 2 ticket(s)")

	// Verify tickets were created
	t1, err := store.Read("tic-imp1")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Imported Ticket 1", t1.Title)
	require.Equal(s.T(), tk.StatusOpen, t1.Status)
	require.Equal(s.T(), 1, t1.Priority)

	t2, err := store.Read("tic-imp2")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Imported Ticket 2", t2.Title)
	require.Equal(s.T(), tk.StatusClosed, t2.Status)
	require.Equal(s.T(), tk.TypeBug, t2.Type)
}

func (s *CmdSuite) TestImportCommandSkipExisting() {
	// Create an existing ticket
	s.createTestTicket("tic-impskip", tk.StatusOpen, "Existing Ticket")

	// Try to import a ticket with the same ID
	importData := `[
		{
			"ID": "tic-impskip",
			"Status": "closed",
			"Title": "Should Be Skipped"
		},
		{
			"ID": "tic-impnew",
			"Status": "open",
			"Title": "New Ticket"
		}
	]`

	importFile := filepath.Join(s.tempDir, "import-skip.json")
	err := os.WriteFile(importFile, []byte(importData), 0644)
	require.NoError(s.T(), err)

	output, err := s.executeCommand("import", importFile, "--skip-existing")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Imported 1 ticket(s)")
	require.Contains(s.T(), output, "skipped 1 existing")

	// Verify existing ticket was not modified
	existing, err := store.Read("tic-impskip")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Existing Ticket", existing.Title)
	require.Equal(s.T(), tk.StatusOpen, existing.Status)

	// Verify new ticket was created
	newTicket, err := store.Read("tic-impnew")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "New Ticket", newTicket.Title)
}

func (s *CmdSuite) TestImportCommandConflictError() {
	// Create an existing ticket
	s.createTestTicket("tic-impconflict", tk.StatusOpen, "Existing Ticket")

	// Try to import a ticket with the same ID without --skip-existing
	importData := `[{"ID": "tic-impconflict", "Title": "Conflict"}]`

	importFile := filepath.Join(s.tempDir, "import-conflict.json")
	err := os.WriteFile(importFile, []byte(importData), 0644)
	require.NoError(s.T(), err)

	_, err = s.executeCommand("import", importFile)

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "already exists")
}

func (s *CmdSuite) TestImportCommandGeneratesID() {
	// Import ticket without ID
	importData := `[{"Title": "No ID Ticket", "Status": "open"}]`

	importFile := filepath.Join(s.tempDir, "import-noid.json")
	err := os.WriteFile(importFile, []byte(importData), 0644)
	require.NoError(s.T(), err)

	output, err := s.executeCommand("import", importFile)

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Imported 1 ticket(s)")
	require.Contains(s.T(), output, "generated 1 ID(s)")
}

func (s *CmdSuite) TestImportCommandInvalidJSON() {
	importFile := filepath.Join(s.tempDir, "invalid.json")
	err := os.WriteFile(importFile, []byte("not valid json"), 0644)
	require.NoError(s.T(), err)

	_, err = s.executeCommand("import", importFile)

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "failed to parse JSON")
}

func (s *CmdSuite) TestImportCommandFileNotFound() {
	_, err := s.executeCommand("import", "/nonexistent/file.json")

	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "failed to read input")
}

func (s *CmdSuite) TestExportImportRoundTrip() {
	// Create tickets with various fields
	t1 := s.createTestTicket("tic-rt1", tk.StatusOpen, "Round Trip 1")
	t1.Description = "Test description"
	t1.Tags = []string{"backend", "api"}
	t1.Priority = 1
	t1.Assignee = "developer"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-rt2", tk.StatusInProgress, "Round Trip 2")
	t2.Type = tk.TypeBug
	t2.Deps = []string{"tic-rt1"}
	require.NoError(s.T(), store.Write(t2))

	// Export to file
	exportFile := filepath.Join(s.tempDir, "roundtrip.json")
	_, err := s.executeCommand("export", "--output="+exportFile)
	require.NoError(s.T(), err)

	// Delete tickets
	require.NoError(s.T(), store.Delete("tic-rt1"))
	require.NoError(s.T(), store.Delete("tic-rt2"))

	// Verify deleted
	require.False(s.T(), store.Exists("tic-rt1"))
	require.False(s.T(), store.Exists("tic-rt2"))

	// Import back
	output, err := s.executeCommand("import", exportFile)
	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Imported")

	// Verify tickets restored
	restored1, err := store.Read("tic-rt1")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Round Trip 1", restored1.Title)
	require.Equal(s.T(), "Test description", restored1.Description)
	require.Equal(s.T(), []string{"backend", "api"}, restored1.Tags)
	require.Equal(s.T(), 1, restored1.Priority)
	require.Equal(s.T(), "developer", restored1.Assignee)

	restored2, err := store.Read("tic-rt2")
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Round Trip 2", restored2.Title)
	require.Equal(s.T(), tk.TypeBug, restored2.Type)
	require.Equal(s.T(), tk.StatusInProgress, restored2.Status)
	require.Equal(s.T(), []string{"tic-rt1"}, restored2.Deps)
}

func (s *CmdSuite) TestBulkCloseByTag() {
	t1 := s.createTestTicket("tic-bulk1", tk.StatusOpen, "Bulk Test 1")
	t1.Tags = []string{"sprint-1"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-bulk2", tk.StatusOpen, "Bulk Test 2")
	t2.Tags = []string{"sprint-1"}
	require.NoError(s.T(), store.Write(t2))

	t3 := s.createTestTicket("tic-bulk3", tk.StatusOpen, "Bulk Test 3")
	t3.Tags = []string{"sprint-2"}
	require.NoError(s.T(), store.Write(t3))

	output, err := s.executeCommand("bulk", "close", "--tag=sprint-1")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Successfully closed 2 ticket(s)")

	// Verify tickets were closed
	ticket1, _ := store.Read("tic-bulk1")
	require.Equal(s.T(), tk.StatusClosed, ticket1.Status)

	ticket2, _ := store.Read("tic-bulk2")
	require.Equal(s.T(), tk.StatusClosed, ticket2.Status)

	// Third ticket should still be open
	ticket3, _ := store.Read("tic-bulk3")
	require.Equal(s.T(), tk.StatusOpen, ticket3.Status)
}

func (s *CmdSuite) TestBulkStartByAssignee() {
	t1 := s.createTestTicket("tic-bulkstart1", tk.StatusOpen, "Start Test 1")
	t1.Assignee = "alice"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-bulkstart2", tk.StatusOpen, "Start Test 2")
	t2.Assignee = "alice"
	require.NoError(s.T(), store.Write(t2))

	t3 := s.createTestTicket("tic-bulkstart3", tk.StatusOpen, "Start Test 3")
	t3.Assignee = "bob"
	require.NoError(s.T(), store.Write(t3))

	output, err := s.executeCommand("bulk", "start", "--assignee=alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Successfully started 2 ticket(s)")

	// Verify tickets were started
	ticket1, _ := store.Read("tic-bulkstart1")
	require.Equal(s.T(), tk.StatusInProgress, ticket1.Status)

	ticket2, _ := store.Read("tic-bulkstart2")
	require.Equal(s.T(), tk.StatusInProgress, ticket2.Status)

	// Third ticket should still be open
	ticket3, _ := store.Read("tic-bulkstart3")
	require.Equal(s.T(), tk.StatusOpen, ticket3.Status)
}

func (s *CmdSuite) TestBulkReopenByStatus() {
	s.createTestTicket("tic-bulkreopen1", tk.StatusClosed, "Reopen Test 1")
	s.createTestTicket("tic-bulkreopen2", tk.StatusClosed, "Reopen Test 2")
	s.createTestTicket("tic-bulkreopen3", tk.StatusOpen, "Reopen Test 3")

	output, err := s.executeCommand("bulk", "reopen", "--status=closed")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Successfully reopened 2 ticket(s)")

	// Verify tickets were reopened
	ticket1, _ := store.Read("tic-bulkreopen1")
	require.Equal(s.T(), tk.StatusOpen, ticket1.Status)

	ticket2, _ := store.Read("tic-bulkreopen2")
	require.Equal(s.T(), tk.StatusOpen, ticket2.Status)
}

func (s *CmdSuite) TestBulkDryRun() {
	t1 := s.createTestTicket("tic-bulkdry1", tk.StatusOpen, "Dry Run Test 1")
	t1.Tags = []string{"test"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-bulkdry2", tk.StatusOpen, "Dry Run Test 2")
	t2.Tags = []string{"test"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("bulk", "close", "--tag=test", "--dry-run")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Dry run: would closed 2 ticket(s)")
	require.Contains(s.T(), output, "tic-bulkdry1")
	require.Contains(s.T(), output, "tic-bulkdry2")

	// Verify tickets were NOT closed
	ticket1, _ := store.Read("tic-bulkdry1")
	require.Equal(s.T(), tk.StatusOpen, ticket1.Status)

	ticket2, _ := store.Read("tic-bulkdry2")
	require.Equal(s.T(), tk.StatusOpen, ticket2.Status)
}

func (s *CmdSuite) TestBulkNoMatches() {
	s.createTestTicket("tic-bulknomatch", tk.StatusOpen, "No Match Test")

	output, err := s.executeCommand("bulk", "close", "--tag=nonexistent")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "No tickets match the specified filters")
}

func (s *CmdSuite) TestBulkAlreadyInTargetStatus() {
	s.createTestTicket("tic-bulkalready1", tk.StatusClosed, "Already Closed 1")
	s.createTestTicket("tic-bulkalready2", tk.StatusClosed, "Already Closed 2")

	output, err := s.executeCommand("bulk", "close", "--status=closed")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "No tickets needed updating")
}

func (s *CmdSuite) TestBulkMultipleFilters() {
	t1 := s.createTestTicket("tic-bulkmulti1", tk.StatusOpen, "Multi Filter 1")
	t1.Tags = []string{"urgent"}
	t1.Assignee = "alice"
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-bulkmulti2", tk.StatusOpen, "Multi Filter 2")
	t2.Tags = []string{"urgent"}
	t2.Assignee = "bob"
	require.NoError(s.T(), store.Write(t2))

	t3 := s.createTestTicket("tic-bulkmulti3", tk.StatusOpen, "Multi Filter 3")
	t3.Tags = []string{"normal"}
	t3.Assignee = "alice"
	require.NoError(s.T(), store.Write(t3))

	output, err := s.executeCommand("bulk", "close", "--tag=urgent", "--assignee=alice")

	require.NoError(s.T(), err)
	require.Contains(s.T(), output, "Successfully closed 1 ticket(s)")

	// Only first ticket should be closed
	ticket1, _ := store.Read("tic-bulkmulti1")
	require.Equal(s.T(), tk.StatusClosed, ticket1.Status)

	ticket2, _ := store.Read("tic-bulkmulti2")
	require.Equal(s.T(), tk.StatusOpen, ticket2.Status)

	ticket3, _ := store.Read("tic-bulkmulti3")
	require.Equal(s.T(), tk.StatusOpen, ticket3.Status)
}

// --- JSON output tests ---

func (s *CmdSuite) TestCreateCommandJSON() {
	output, err := s.executeCommand("create", "--json", "JSON Create Test", "--type", "bug", "--priority", "1")

	require.NoError(s.T(), err)
	var t tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &t))
	require.Equal(s.T(), "JSON Create Test", t.Title)
	require.Equal(s.T(), tk.TypeBug, t.Type)
	require.Equal(s.T(), 1, t.Priority)
	require.Equal(s.T(), tk.StatusOpen, t.Status)
}

func (s *CmdSuite) TestShowCommandJSON() {
	ticket := s.createTestTicket("tic-showjson", tk.StatusOpen, "Show JSON Test")
	ticket.Description = "Test description"
	require.NoError(s.T(), store.Write(ticket))

	output, err := s.executeCommand("show", "--json", "tic-showjson")

	require.NoError(s.T(), err)
	var t tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &t))
	require.Equal(s.T(), "tic-showjson", t.ID)
	require.Equal(s.T(), "Show JSON Test", t.Title)
	require.Equal(s.T(), "Test description", t.Description)
}

func (s *CmdSuite) TestStartCommandJSON() {
	s.createTestTicket("tic-startjson", tk.StatusOpen, "Start JSON Test")

	output, err := s.executeCommand("start", "--json", "tic-startjson")

	require.NoError(s.T(), err)
	var t tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &t))
	require.Equal(s.T(), tk.StatusInProgress, t.Status)
	require.Equal(s.T(), "tic-startjson", t.ID)
}

func (s *CmdSuite) TestCloseCommandJSON() {
	s.createTestTicket("tic-closejson", tk.StatusOpen, "Close JSON Test")

	output, err := s.executeCommand("close", "--json", "tic-closejson")

	require.NoError(s.T(), err)
	var t tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &t))
	require.Equal(s.T(), tk.StatusClosed, t.Status)
}

func (s *CmdSuite) TestReopenCommandJSON() {
	s.createTestTicket("tic-reopenjson", tk.StatusClosed, "Reopen JSON Test")

	output, err := s.executeCommand("reopen", "--json", "tic-reopenjson")

	require.NoError(s.T(), err)
	var t tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &t))
	require.Equal(s.T(), tk.StatusOpen, t.Status)
}

func (s *CmdSuite) TestStatusCommandJSON() {
	s.createTestTicket("tic-statusjson", tk.StatusOpen, "Status JSON Test")

	output, err := s.executeCommand("status", "--json", "tic-statusjson", "in_progress")

	require.NoError(s.T(), err)
	var t tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &t))
	require.Equal(s.T(), tk.StatusInProgress, t.Status)
}

func (s *CmdSuite) TestListCommandJSON() {
	s.createTestTicket("tic-listjson1", tk.StatusOpen, "List JSON A")
	s.createTestTicket("tic-listjson2", tk.StatusInProgress, "List JSON B")

	output, err := s.executeCommand("list", "--json")

	require.NoError(s.T(), err)
	var tickets []*tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &tickets))
	require.Len(s.T(), tickets, 2)
}

func (s *CmdSuite) TestListCommandJSONWithFilter() {
	s.createTestTicket("tic-listjf1", tk.StatusOpen, "Open JSON")
	s.createTestTicket("tic-listjf2", tk.StatusClosed, "Closed JSON")

	output, err := s.executeCommand("list", "--json", "--status", "open")

	require.NoError(s.T(), err)
	var tickets []*tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &tickets))
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "tic-listjf1", tickets[0].ID)
}

func (s *CmdSuite) TestReadyCommandJSON() {
	s.createTestTicket("tic-readyjson1", tk.StatusOpen, "Ready JSON")
	t2 := s.createTestTicket("tic-readyjson2", tk.StatusOpen, "Blocked JSON")
	t2.Deps = []string{"tic-readyjson1"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("ready", "--json")

	require.NoError(s.T(), err)
	var tickets []*tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &tickets))
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "tic-readyjson1", tickets[0].ID)
}

func (s *CmdSuite) TestBlockedCommandJSON() {
	s.createTestTicket("tic-blkjson1", tk.StatusOpen, "Blocker JSON")
	t2 := s.createTestTicket("tic-blkjson2", tk.StatusOpen, "Blocked JSON")
	t2.Deps = []string{"tic-blkjson1"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("blocked", "--json")

	require.NoError(s.T(), err)
	var tickets []*tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &tickets))
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "tic-blkjson2", tickets[0].ID)
}

func (s *CmdSuite) TestClosedCommandJSON() {
	s.createTestTicket("tic-clsjson1", tk.StatusOpen, "Open JSON")
	s.createTestTicket("tic-clsjson2", tk.StatusClosed, "Closed JSON")

	output, err := s.executeCommand("closed", "--json")

	require.NoError(s.T(), err)
	var tickets []*tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &tickets))
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "tic-clsjson2", tickets[0].ID)
}

func (s *CmdSuite) TestDepAddCommandJSON() {
	s.createTestTicket("tic-depjson-a", tk.StatusOpen, "Dep JSON A")
	s.createTestTicket("tic-depjson-b", tk.StatusOpen, "Dep JSON B")

	output, err := s.executeCommand("dep", "add", "--json", "tic-depjson-b", "tic-depjson-a")

	require.NoError(s.T(), err)
	var t tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &t))
	require.Equal(s.T(), "tic-depjson-b", t.ID)
	require.Equal(s.T(), []string{"tic-depjson-a"}, t.Deps)
}

func (s *CmdSuite) TestDepCheckCommandJSON() {
	s.createTestTicket("tic-chkjson1", tk.StatusOpen, "Check JSON A")
	s.createTestTicket("tic-chkjson2", tk.StatusOpen, "Check JSON B")

	output, err := s.executeCommand("dep", "check", "--json")

	require.NoError(s.T(), err)
	var result map[string]any
	require.NoError(s.T(), json.Unmarshal([]byte(output), &result))
	require.Equal(s.T(), float64(0), result["count"])
}

func (s *CmdSuite) TestVersionCommandJSON() {
	output, err := s.executeCommand("version", "--json")

	require.NoError(s.T(), err)
	var v map[string]string
	require.NoError(s.T(), json.Unmarshal([]byte(output), &v))
	require.NotEmpty(s.T(), v["version"])
}

func (s *CmdSuite) TestSearchCommandJSON() {
	ticket := s.createTestTicket("tic-srchjson", tk.StatusOpen, "Searchable JSON title")
	ticket.Description = "Has matching content"
	require.NoError(s.T(), store.Write(ticket))

	output, err := s.executeCommand("search", "--json", "Searchable")

	require.NoError(s.T(), err)
	var matches []tk.SearchMatch
	require.NoError(s.T(), json.Unmarshal([]byte(output), &matches))
	require.Len(s.T(), matches, 1)
	require.Equal(s.T(), "tic-srchjson", matches[0].Ticket.ID)
}

func (s *CmdSuite) TestBulkCloseCommandJSON() {
	t1 := s.createTestTicket("tic-bulkjson1", tk.StatusOpen, "Bulk JSON 1")
	t1.Tags = []string{"release"}
	require.NoError(s.T(), store.Write(t1))

	t2 := s.createTestTicket("tic-bulkjson2", tk.StatusOpen, "Bulk JSON 2")
	t2.Tags = []string{"other"}
	require.NoError(s.T(), store.Write(t2))

	output, err := s.executeCommand("bulk", "close", "--json", "--tag=release")

	require.NoError(s.T(), err)
	var updated []*tk.Ticket
	require.NoError(s.T(), json.Unmarshal([]byte(output), &updated))
	require.Len(s.T(), updated, 1)
	require.Equal(s.T(), tk.StatusClosed, updated[0].Status)
}
