// Package component runs end-to-end tests against the real tk binary.
// Build the binary first: go build -o <path> ./cmd/tk
// Then run: TK_BINARY=<path> go test -v ./test/component/
package component

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ticket mirrors the JSON output from commands with --json.
type ticket struct {
	ID          string   `json:"ID"`
	Status      string   `json:"Status"`
	Type        string   `json:"Type"`
	Priority    int      `json:"Priority"`
	Assignee    string   `json:"Assignee"`
	Parent      string   `json:"Parent"`
	ExternalRef string   `json:"ExternalRef"`
	Tags        []string `json:"Tags"`
	Deps        []string `json:"Deps"`
	Links       []string `json:"Links"`
	Title       string   `json:"Title"`
	Description string   `json:"Description"`
	Notes       []note   `json:"Notes"`
}

type note struct {
	Timestamp string `json:"Timestamp"`
	Content   string `json:"Content"`
}

// searchMatch mirrors the JSON output from `tk search --json`.
type searchMatch struct {
	Ticket  ticket `json:"Ticket"`
	Context string `json:"Context"`
}

type TKSuite struct {
	suite.Suite
	binary  string
	workDir string
}

func TestTKSuite(t *testing.T) {
	binary := os.Getenv("TK_BINARY")
	if binary == "" {
		// Build the binary to a temp location
		tmpBin := filepath.Join(t.TempDir(), "tk")
		cmd := exec.Command("go", "build", "-o", tmpBin, "./cmd/tk")
		cmd.Dir = filepath.Join("..", "..")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "failed to build tk binary: %s", string(out))
		binary = tmpBin
	}

	s := &TKSuite{binary: binary}
	suite.Run(t, s)
}

func (s *TKSuite) SetupTest() {
	dir, err := os.MkdirTemp("", "tk-component-*")
	require.NoError(s.T(), err)
	s.workDir = dir
}

func (s *TKSuite) TearDownTest() {
	_ = os.RemoveAll(s.workDir)
}

// tk runs the tk binary with the given args in the work directory.
func (s *TKSuite) tk(args ...string) (string, error) {
	cmd := exec.Command(s.binary, args...)
	cmd.Env = append(os.Environ(), "TICKETS_DIR="+filepath.Join(s.workDir, ".tickets"))
	cmd.Dir = s.workDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// tkOK runs tk and asserts no error.
func (s *TKSuite) tkOK(args ...string) string {
	out, err := s.tk(args...)
	require.NoError(s.T(), err, "tk %s failed: %s", strings.Join(args, " "), out)
	return out
}

// tkFail runs tk and asserts an error.
func (s *TKSuite) tkFail(args ...string) string {
	out, err := s.tk(args...)
	require.Error(s.T(), err, "tk %s should have failed but got: %s", strings.Join(args, " "), out)
	return out
}

// listTickets returns all tickets as parsed JSON via `tk list --json`.
func (s *TKSuite) listTickets(extraArgs ...string) []ticket {
	args := append([]string{"list", "--json"}, extraArgs...)
	out := s.tkOK(args...)
	var tickets []ticket
	err := json.Unmarshal([]byte(out), &tickets)
	require.NoError(s.T(), err, "failed to parse list --json output: %s", out)
	return tickets
}

// showTicket returns a single ticket by ID via `tk show --json`.
func (s *TKSuite) showTicket(id string) ticket {
	out := s.tkOK("show", "--json", id)
	var t ticket
	err := json.Unmarshal([]byte(out), &t)
	require.NoError(s.T(), err, "failed to parse show --json output: %s", out)
	return t
}

// ticketTitles extracts titles from a slice of tickets.
func ticketTitles(tickets []ticket) []string {
	titles := make([]string, len(tickets))
	for i, t := range tickets {
		titles[i] = t.Title
	}
	return titles
}

// createTicket is a shorthand that creates a ticket and returns its ID.
func (s *TKSuite) createTicket(args ...string) string {
	out := s.tkOK(append([]string{"create"}, args...)...)
	return strings.TrimSpace(out)
}

// --- Create & Show ---

func (s *TKSuite) TestCreateAndShow() {
	// Test create with --json returns the full ticket
	out := s.tkOK("create", "--json", "My first ticket", "-t", "bug", "-p", "1", "-d", "A bug description")
	var created ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &created))
	require.True(s.T(), strings.HasPrefix(created.ID, "tic-"), "expected tic- prefix, got: %s", created.ID)
	require.Equal(s.T(), "My first ticket", created.Title)
	require.Equal(s.T(), "bug", created.Type)
	require.Equal(s.T(), 1, created.Priority)
	require.Equal(s.T(), "A bug description", created.Description)
	require.Equal(s.T(), "open", created.Status)

	// Verify show --json returns the same data
	t := s.showTicket(created.ID)
	require.Equal(s.T(), created.ID, t.ID)
	require.Equal(s.T(), created.Title, t.Title)
}

func (s *TKSuite) TestCreateWithTags() {
	id := s.createTicket("Tagged ticket", "--tags", "ui,backend")

	t := s.showTicket(id)
	require.Equal(s.T(), []string{"ui", "backend"}, t.Tags)
}

func (s *TKSuite) TestCreateWithParent() {
	parentID := s.createTicket("Parent ticket")
	childID := s.createTicket("Child ticket", "--parent", parentID)

	t := s.showTicket(childID)
	require.Equal(s.T(), parentID, t.Parent)
}

func (s *TKSuite) TestCreateInvalidPriority() {
	out := s.tkFail("create", "Bad priority", "-p", "9")
	require.Contains(s.T(), out, "invalid priority")
}

func (s *TKSuite) TestCreateInvalidPriorityNegative() {
	out := s.tkFail("create", "Bad priority", "-p", "-1")
	require.Contains(s.T(), out, "invalid priority")
}

func (s *TKSuite) TestCreateInvalidType() {
	out := s.tkFail("create", "Bad type", "-t", "invalid")
	require.Contains(s.T(), out, "invalid type")
}

// --- List ---

func (s *TKSuite) TestList() {
	s.createTicket("Ticket A", "-p", "1", "-t", "bug")
	s.createTicket("Ticket B", "-p", "3", "-t", "feature")

	tickets := s.listTickets()
	require.Len(s.T(), tickets, 2)
	titles := []string{tickets[0].Title, tickets[1].Title}
	require.Contains(s.T(), titles, "Ticket A")
	require.Contains(s.T(), titles, "Ticket B")
}

func (s *TKSuite) TestListFilterByStatus() {
	id := s.createTicket("Open ticket")
	s.tkOK("start", id)
	s.createTicket("Still open")

	tickets := s.listTickets("--status", "in_progress")
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "Open ticket", tickets[0].Title)
}

func (s *TKSuite) TestListFilterByType() {
	s.createTicket("A bug", "-t", "bug")
	s.createTicket("A feature", "-t", "feature")

	tickets := s.listTickets("-t", "bug")
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "A bug", tickets[0].Title)
}

func (s *TKSuite) TestListSortByPriority() {
	s.createTicket("Low priority", "-p", "4")
	s.createTicket("High priority", "-p", "0")

	tickets := s.listTickets("--sort", "priority")
	require.Len(s.T(), tickets, 2)
	require.Equal(s.T(), "High priority", tickets[0].Title)
	require.Equal(s.T(), "Low priority", tickets[1].Title)
}

// --- Status transitions ---

func (s *TKSuite) TestStartCloseReopen() {
	id := s.createTicket("Status test")

	// start --json returns the updated ticket
	out := s.tkOK("start", "--json", id)
	var t ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &t))
	require.Equal(s.T(), "in_progress", t.Status)

	// close --json returns the updated ticket
	out = s.tkOK("close", "--json", id)
	require.NoError(s.T(), json.Unmarshal([]byte(out), &t))
	require.Equal(s.T(), "closed", t.Status)

	// reopen --json returns the updated ticket
	out = s.tkOK("reopen", "--json", id)
	require.NoError(s.T(), json.Unmarshal([]byte(out), &t))
	require.Equal(s.T(), "open", t.Status)
}

func (s *TKSuite) TestStatusCommand() {
	id := s.createTicket("Direct status")
	s.tkOK("status", id, "in_progress")
	require.Equal(s.T(), "in_progress", s.showTicket(id).Status)
}

func (s *TKSuite) TestStatusInvalid() {
	id := s.createTicket("Bad status")
	out := s.tkFail("status", id, "invalid")
	require.Contains(s.T(), out, "invalid status")
}

func (s *TKSuite) TestStartNonexistent() {
	out := s.tkFail("start", "nonexistent")
	require.Contains(s.T(), out, "not found")
}

// --- Partial ID matching ---

func (s *TKSuite) TestPartialIDMatch() {
	id := s.createTicket("Partial ID test")
	partial := id[len(id)-4:]

	t := s.showTicket(id)
	show := s.tkOK("show", partial)
	require.Contains(s.T(), show, t.Title)
}

// --- Dependencies ---

func (s *TKSuite) TestDepAddAndTree() {
	id1 := s.createTicket("Dep parent")
	id2 := s.createTicket("Dep child")

	// dep add --json returns the updated ticket
	out := s.tkOK("dep", "add", "--json", id2, id1)
	var t ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &t))
	require.Equal(s.T(), id2, t.ID)
	require.Equal(s.T(), []string{id1}, t.Deps)

	tree := s.tkOK("dep", "tree", "--full")
	require.Contains(s.T(), tree, "Dep parent")
	require.Contains(s.T(), tree, "Dep child")
}

func (s *TKSuite) TestDepRemove() {
	id1 := s.createTicket("Dep A")
	id2 := s.createTicket("Dep B")
	s.tkOK("dep", "add", id2, id1)

	s.tkOK("dep", "remove", id2, id1)

	t := s.showTicket(id2)
	require.Empty(s.T(), t.Deps)
}

func (s *TKSuite) TestUndep() {
	id1 := s.createTicket("Undep A")
	id2 := s.createTicket("Undep B")
	s.tkOK("dep", "add", id2, id1)

	s.tkOK("undep", id2, id1)

	t := s.showTicket(id2)
	require.Empty(s.T(), t.Deps)
}

func (s *TKSuite) TestDepSelfDep() {
	id := s.createTicket("Self dep")
	out := s.tkFail("dep", "add", id, id)
	require.Contains(s.T(), out, "cannot depend on itself")
}

func (s *TKSuite) TestDepDuplicate() {
	id1 := s.createTicket("Dup dep A")
	id2 := s.createTicket("Dup dep B")
	s.tkOK("dep", "add", id1, id2)

	out := s.tkFail("dep", "add", id1, id2)
	require.Contains(s.T(), out, "already exists")
}

func (s *TKSuite) TestDepCycle() {
	id1 := s.createTicket("Cycle A")
	id2 := s.createTicket("Cycle B")
	s.tkOK("dep", "add", id1, id2)

	out := s.tkFail("dep", "add", id2, id1)
	require.Contains(s.T(), out, "cycle")
}

func (s *TKSuite) TestDepRemoveNonexistent() {
	id1 := s.createTicket("No dep A")
	id2 := s.createTicket("No dep B")

	out := s.tkFail("dep", "remove", id1, id2)
	require.Contains(s.T(), out, "not found")
}

func (s *TKSuite) TestDepCheck() {
	s.createTicket("Check A")
	s.createTicket("Check B")

	out := s.tkOK("dep", "check", "--json")
	var result map[string]any
	require.NoError(s.T(), json.Unmarshal([]byte(out), &result))
	require.Equal(s.T(), float64(0), result["count"])
}

// --- Links ---

func (s *TKSuite) TestLinkAndUnlink() {
	id1 := s.createTicket("Link A")
	id2 := s.createTicket("Link B")

	// link --json returns all linked tickets
	out := s.tkOK("link", "--json", id1, id2)
	var linked []ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &linked))
	require.Len(s.T(), linked, 2)

	t1 := s.showTicket(id1)
	t2 := s.showTicket(id2)
	require.Equal(s.T(), []string{id2}, t1.Links)
	require.Equal(s.T(), []string{id1}, t2.Links)

	// unlink --json returns both tickets
	out = s.tkOK("unlink", "--json", id1, id2)
	var unlinked []ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &unlinked))
	require.Len(s.T(), unlinked, 2)
	require.Empty(s.T(), unlinked[0].Links)
	require.Empty(s.T(), unlinked[1].Links)
}

func (s *TKSuite) TestLinkDedup() {
	id1 := s.createTicket("Dedup A")
	id2 := s.createTicket("Dedup B")

	s.tkOK("link", id1, id2)
	s.tkOK("link", id1, id2) // link again — should not duplicate

	t := s.showTicket(id1)
	require.Equal(s.T(), []string{id2}, t.Links, "link should appear exactly once")
}

func (s *TKSuite) TestLinkDuplicateID() {
	id := s.createTicket("Dup link")
	out := s.tkFail("link", id, id)
	require.Contains(s.T(), out, "duplicate ticket ID")
}

func (s *TKSuite) TestLinkMultiple() {
	id1 := s.createTicket("Multi A")
	id2 := s.createTicket("Multi B")
	id3 := s.createTicket("Multi C")

	s.tkOK("link", id1, id2, id3)

	t := s.showTicket(id1)
	require.Contains(s.T(), t.Links, id2)
	require.Contains(s.T(), t.Links, id3)
}

func (s *TKSuite) TestUnlinkNonexistent() {
	id1 := s.createTicket("Unlink A")
	id2 := s.createTicket("Unlink B")

	out := s.tkFail("unlink", id1, id2)
	require.Contains(s.T(), out, "no link found")
}

// --- Notes ---

func (s *TKSuite) TestAddNote() {
	id := s.createTicket("Note test")

	// add-note --json returns the updated ticket with notes
	out := s.tkOK("add-note", "--json", id, "This is a note")
	var t ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &t))
	require.Len(s.T(), t.Notes, 1)
	require.Equal(s.T(), "This is a note", t.Notes[0].Content)
}

func (s *TKSuite) TestAddMultipleNotes() {
	id := s.createTicket("Multi note")
	s.tkOK("add-note", id, "First note")
	s.tkOK("add-note", id, "Second note")

	t := s.showTicket(id)
	require.Len(s.T(), t.Notes, 2)
	require.Equal(s.T(), "First note", t.Notes[0].Content)
	require.Equal(s.T(), "Second note", t.Notes[1].Content)
}

// --- Search ---

func (s *TKSuite) TestSearch() {
	s.createTicket("Unique searchable title")
	s.createTicket("Another ticket")

	out := s.tkOK("search", "--json", "searchable")
	var matches []searchMatch
	require.NoError(s.T(), json.Unmarshal([]byte(out), &matches))
	require.Len(s.T(), matches, 1)
	require.Equal(s.T(), "Unique searchable title", matches[0].Ticket.Title)
}

func (s *TKSuite) TestSearchCaseInsensitive() {
	s.createTicket("CamelCase Title")

	out := s.tkOK("search", "--json", "camelcase")
	var matches []searchMatch
	require.NoError(s.T(), json.Unmarshal([]byte(out), &matches))
	require.Len(s.T(), matches, 1)
	require.Equal(s.T(), "CamelCase Title", matches[0].Ticket.Title)
}

func (s *TKSuite) TestSearchCaseSensitive() {
	s.createTicket("CamelCase Title")

	out := s.tkOK("search", "--json", "camelcase", "--case-sensitive")
	var matches []searchMatch
	require.NoError(s.T(), json.Unmarshal([]byte(out), &matches))
	require.Empty(s.T(), matches)
}

// --- Stats ---

func (s *TKSuite) TestStats() {
	id := s.createTicket("Stats A", "-t", "bug")
	s.createTicket("Stats B", "-t", "feature")
	s.tkOK("close", id)

	out := s.tkOK("stats", "--json")
	var result map[string]any
	require.NoError(s.T(), json.Unmarshal([]byte(out), &result))
	require.Equal(s.T(), float64(2), result["total"])
	byStatus := result["by_status"].(map[string]any)
	require.Equal(s.T(), float64(1), byStatus["closed"])
	byType := result["by_type"].(map[string]any)
	require.Equal(s.T(), float64(1), byType["bug"])
	require.Equal(s.T(), float64(1), byType["feature"])
}

func (s *TKSuite) TestStatsGlobalJSON() {
	s.createTicket("JSON stats", "-t", "task")

	// Global --json flag also works for stats
	out := s.tkOK("stats", "--json")
	var result map[string]any
	require.NoError(s.T(), json.Unmarshal([]byte(out), &result))
	require.Equal(s.T(), float64(1), result["total"])
}

// --- Export ---

func (s *TKSuite) TestExportJSON() {
	s.createTicket("Export ticket")

	out := s.tkOK("export")
	var tickets []ticket
	err := json.Unmarshal([]byte(out), &tickets)
	require.NoError(s.T(), err, "export should produce valid JSON")
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "Export ticket", tickets[0].Title)
}

func (s *TKSuite) TestExportCSV() {
	s.createTicket("CSV ticket")

	out := s.tkOK("export", "--format", "csv")
	require.Contains(s.T(), out, "CSV ticket")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.GreaterOrEqual(s.T(), len(lines), 2, "CSV should have header + data row")
}

func (s *TKSuite) TestExportToFile() {
	s.createTicket("File export")

	outFile := filepath.Join(s.workDir, "export.json")
	s.tkOK("export", "-o", outFile)

	data, err := os.ReadFile(outFile)
	require.NoError(s.T(), err)

	var tickets []ticket
	err = json.Unmarshal(data, &tickets)
	require.NoError(s.T(), err)
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "File export", tickets[0].Title)
}

func (s *TKSuite) TestExportInvalidFormat() {
	out := s.tkFail("export", "--format", "xml")
	require.Contains(s.T(), out, "unsupported format")
}

// --- Import ---

func (s *TKSuite) TestImport() {
	importData := `[{"ID":"tic-imp1","Status":"open","Type":"task","Priority":2,"Title":"Imported ticket","Created":"2026-01-01T00:00:00Z"}]`
	importFile := filepath.Join(s.workDir, "import.json")
	require.NoError(s.T(), os.WriteFile(importFile, []byte(importData), 0644))

	out := s.tkOK("import", "--json", importFile)
	var imported []ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &imported))
	require.Len(s.T(), imported, 1)
	require.Equal(s.T(), "tic-imp1", imported[0].ID)
	require.Equal(s.T(), "Imported ticket", imported[0].Title)
	require.Equal(s.T(), "open", imported[0].Status)
	require.Equal(s.T(), "task", imported[0].Type)
	require.Equal(s.T(), 2, imported[0].Priority)
}

func (s *TKSuite) TestImportDuplicateFails() {
	importData := `[{"ID":"tic-dup1","Status":"open","Type":"task","Priority":2,"Title":"Dup","Created":"2026-01-01T00:00:00Z"}]`
	importFile := filepath.Join(s.workDir, "import.json")
	require.NoError(s.T(), os.WriteFile(importFile, []byte(importData), 0644))

	s.tkOK("import", importFile)
	out := s.tkFail("import", importFile)
	require.Contains(s.T(), out, "already exists")
}

func (s *TKSuite) TestImportSkipExisting() {
	importData := `[{"ID":"tic-skip1","Status":"open","Type":"task","Priority":2,"Title":"Skip","Created":"2026-01-01T00:00:00Z"}]`
	importFile := filepath.Join(s.workDir, "import.json")
	require.NoError(s.T(), os.WriteFile(importFile, []byte(importData), 0644))

	s.tkOK("import", importFile)
	s.tkOK("import", importFile, "--skip-existing")
}

func (s *TKSuite) TestImportInvalidFile() {
	out := s.tkFail("import", "/nonexistent/file.json")
	require.Contains(s.T(), out, "no such file")
}

// --- Bulk ---

func (s *TKSuite) TestBulkClose() {
	idA := s.createTicket("Bulk A", "--tags", "release")
	idB := s.createTicket("Bulk B", "--tags", "release")
	idC := s.createTicket("Bulk C", "--tags", "other")

	out := s.tkOK("bulk", "close", "--json", "--tag", "release")
	var updated []ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &updated))
	require.Len(s.T(), updated, 2)

	require.Equal(s.T(), "closed", s.showTicket(idA).Status)
	require.Equal(s.T(), "closed", s.showTicket(idB).Status)
	require.Equal(s.T(), "open", s.showTicket(idC).Status)
}

func (s *TKSuite) TestBulkDryRun() {
	id := s.createTicket("Dry run ticket", "--tags", "dry")

	out := s.tkOK("bulk", "close", "--json", "--tag", "dry", "--dry-run")
	var preview []ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &preview))
	require.Len(s.T(), preview, 1)
	require.Equal(s.T(), "Dry run ticket", preview[0].Title)

	require.Equal(s.T(), "open", s.showTicket(id).Status, "dry-run should not change status")
}

// --- Query ---

func (s *TKSuite) TestListJSON() {
	s.createTicket("JSON list ticket")

	tickets := s.listTickets()
	require.Len(s.T(), tickets, 1)
	require.Equal(s.T(), "JSON list ticket", tickets[0].Title)
}

// --- Ready / Blocked ---

func (s *TKSuite) TestReadyAndBlocked() {
	id1 := s.createTicket("Blocker")
	id2 := s.createTicket("Blocked ticket")
	s.tkOK("dep", "add", id2, id1)
	s.createTicket("Ready ticket")

	readyOut := s.tkOK("ready", "--json")
	var readyTickets []ticket
	require.NoError(s.T(), json.Unmarshal([]byte(readyOut), &readyTickets))
	readyTitles := ticketTitles(readyTickets)
	require.Contains(s.T(), readyTitles, "Ready ticket")
	require.Contains(s.T(), readyTitles, "Blocker")
	require.NotContains(s.T(), readyTitles, "Blocked ticket")

	blockedOut := s.tkOK("blocked", "--json")
	var blockedTickets []ticket
	require.NoError(s.T(), json.Unmarshal([]byte(blockedOut), &blockedTickets))
	blockedTitles := ticketTitles(blockedTickets)
	require.Contains(s.T(), blockedTitles, "Blocked ticket")
	require.NotContains(s.T(), blockedTitles, "Ready ticket")
}

// --- Closed list ---

func (s *TKSuite) TestClosedList() {
	id := s.createTicket("Will close")
	s.createTicket("Stay open")
	s.tkOK("close", id)

	out := s.tkOK("closed", "--json")
	var tickets []ticket
	require.NoError(s.T(), json.Unmarshal([]byte(out), &tickets))
	titles := ticketTitles(tickets)
	require.Contains(s.T(), titles, "Will close")
	require.NotContains(s.T(), titles, "Stay open")
}

// --- Version ---

func (s *TKSuite) TestVersion() {
	out := s.tkOK("version", "--json")
	var v map[string]string
	require.NoError(s.T(), json.Unmarshal([]byte(out), &v))
	require.NotEmpty(s.T(), v["version"])
}

// --- Help ---

func (s *TKSuite) TestHelp() {
	out := s.tkOK("--help")
	require.Contains(s.T(), out, "tk")
	require.Contains(s.T(), out, "create")
	require.Contains(s.T(), out, "list")
}

// --- Export + Import round-trip ---

func (s *TKSuite) TestExportImportRoundTrip() {
	id := s.createTicket("Round trip", "-t", "bug", "-p", "1", "--tags", "test", "-d", "Description")
	s.tkOK("start", id)

	exportFile := filepath.Join(s.workDir, "roundtrip.json")
	s.tkOK("export", "-o", exportFile)

	// Create a fresh store
	s.workDir, _ = os.MkdirTemp("", "tk-component-roundtrip-*")
	s.tkOK("import", exportFile)

	t := s.showTicket(id)
	require.Equal(s.T(), "Round trip", t.Title)
	require.Equal(s.T(), "bug", t.Type)
	require.Equal(s.T(), 1, t.Priority)
	require.Equal(s.T(), "in_progress", t.Status)
	require.Equal(s.T(), []string{"test"}, t.Tags)
	require.Equal(s.T(), "Description", t.Description)
}
