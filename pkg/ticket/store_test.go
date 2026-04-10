package ticket

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type StoreSuite struct {
	suite.Suite
	tempDir string
	store   *Store
}

func TestStoreSuite(t *testing.T) {
	suite.Run(t, new(StoreSuite))
}

func (s *StoreSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "ticket-store-test-*")
	require.NoError(s.T(), err)

	ticketsDir := filepath.Join(s.tempDir, TicketsDirName)
	require.NoError(s.T(), os.MkdirAll(ticketsDir, 0755))

	s.store = OpenDir(ticketsDir)
}

func (s *StoreSuite) TearDownTest() {
	_ = os.RemoveAll(s.tempDir)
}

func (s *StoreSuite) TestOpen() {
	store := Open(s.tempDir)
	require.Equal(s.T(), filepath.Join(s.tempDir, TicketsDirName), store.TicketsDir())
}

func (s *StoreSuite) TestGenerateID() {
	id, err := GenerateID()
	require.NoError(s.T(), err)
	require.True(s.T(), len(id) > 0)
	require.Contains(s.T(), id, IDPrefix+"-")
}

func (s *StoreSuite) TestGenerateIDUnique() {
	ids := make(map[string]bool)
	for range 10 {
		id, err := GenerateID()
		require.NoError(s.T(), err)
		require.False(s.T(), ids[id], "duplicate ID generated: %s", id)
		ids[id] = true
	}
}

func (s *StoreSuite) TestWriteAndRead() {
	ticket := &Ticket{
		ID:          "tic-test",
		Status:      StatusOpen,
		Type:        TypeTask,
		Priority:    2,
		Assignee:    "Developer",
		Created:     time.Now().UTC().Truncate(time.Second),
		Title:       "Test Ticket",
		Description: "Test description",
	}

	err := s.store.Write(ticket)
	require.NoError(s.T(), err)

	read, err := s.store.Read("tic-test")
	require.NoError(s.T(), err)
	require.Equal(s.T(), ticket.ID, read.ID)
	require.Equal(s.T(), ticket.Status, read.Status)
	require.Equal(s.T(), ticket.Type, read.Type)
	require.Equal(s.T(), ticket.Priority, read.Priority)
	require.Equal(s.T(), ticket.Assignee, read.Assignee)
	require.Equal(s.T(), ticket.Title, read.Title)
}

func (s *StoreSuite) TestList() {
	tickets := []*Ticket{
		{ID: "tic-aaa1", Status: StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-bbb2", Status: StatusClosed, Created: time.Now().UTC()},
		{ID: "tic-ccc3", Status: StatusInProgress, Created: time.Now().UTC()},
	}

	for _, t := range tickets {
		require.NoError(s.T(), s.store.Write(t))
	}

	list, err := s.store.List()
	require.NoError(s.T(), err)
	require.Len(s.T(), list, 3)
}

func (s *StoreSuite) TestList_EmptyDirectory() {
	list, err := s.store.List()
	require.NoError(s.T(), err)
	require.Len(s.T(), list, 0)
}

func (s *StoreSuite) TestList_NonExistentDirectory() {
	store := OpenDir(filepath.Join(s.tempDir, "nonexistent"))
	list, err := store.List()
	require.NoError(s.T(), err)
	require.Nil(s.T(), list)
}

func (s *StoreSuite) TestList_SkipsSubdirectories() {
	subDir := filepath.Join(s.store.TicketsDir(), "subdir.md")
	require.NoError(s.T(), os.MkdirAll(subDir, 0755))

	ticket := &Ticket{
		ID:      "tic-real",
		Status:  StatusOpen,
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.store.Write(ticket))

	list, err := s.store.List()
	require.NoError(s.T(), err)
	require.Len(s.T(), list, 1)
	require.Equal(s.T(), "tic-real", list[0].ID)
}

func (s *StoreSuite) TestList_SkipsNonMdFiles() {
	txtFile := filepath.Join(s.store.TicketsDir(), "tic-fake.txt")
	require.NoError(s.T(), os.WriteFile(txtFile, []byte("not a ticket"), 0644))

	ticket := &Ticket{
		ID:      "tic-actual",
		Status:  StatusOpen,
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.store.Write(ticket))

	list, err := s.store.List()
	require.NoError(s.T(), err)
	require.Len(s.T(), list, 1)
	require.Equal(s.T(), "tic-actual", list[0].ID)
}

func (s *StoreSuite) TestList_ReadError() {
	invalidFile := filepath.Join(s.store.TicketsDir(), "tic-invalid.md")
	require.NoError(s.T(), os.WriteFile(invalidFile, []byte("not valid yaml frontmatter"), 0644))

	list, err := s.store.List()
	require.Error(s.T(), err)
	require.Nil(s.T(), list)
}

func (s *StoreSuite) TestDelete() {
	ticket := &Ticket{
		ID:      "tic-del1",
		Status:  StatusOpen,
		Created: time.Now().UTC(),
	}

	require.NoError(s.T(), s.store.Write(ticket))
	require.True(s.T(), s.store.Exists("tic-del1"))

	require.NoError(s.T(), s.store.Delete("tic-del1"))
	require.False(s.T(), s.store.Exists("tic-del1"))
}

func (s *StoreSuite) TestExists() {
	require.False(s.T(), s.store.Exists("tic-nonexistent"))

	ticket := &Ticket{
		ID:      "tic-exists",
		Status:  StatusOpen,
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.store.Write(ticket))
	require.True(s.T(), s.store.Exists("tic-exists"))
}

func (s *StoreSuite) TestResolveID() {
	tickets := []*Ticket{
		{ID: "tic-abc1", Status: StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-def2", Status: StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-abc3", Status: StatusOpen, Created: time.Now().UTC()},
	}

	for _, t := range tickets {
		require.NoError(s.T(), s.store.Write(t))
	}

	tests := []struct {
		name    string
		partial string
		want    string
		wantErr bool
	}{
		{name: "exact match", partial: "tic-abc1", want: "tic-abc1"},
		{name: "partial match unique", partial: "def2", want: "tic-def2"},
		{name: "partial match ambiguous", partial: "abc", wantErr: true},
		{name: "no match", partial: "xyz", wantErr: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.store.ResolveID(tt.partial)
			if tt.wantErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), tt.want, got)
		})
	}
}

func (s *StoreSuite) TestListIDs() {
	tickets := []*Ticket{
		{ID: "tic-id1", Status: StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-id2", Status: StatusOpen, Created: time.Now().UTC()},
	}

	for _, t := range tickets {
		require.NoError(s.T(), s.store.Write(t))
	}

	ids, err := s.store.ListIDs()
	require.NoError(s.T(), err)
	require.Len(s.T(), ids, 2)
	require.Contains(s.T(), ids, "tic-id1")
	require.Contains(s.T(), ids, "tic-id2")
}

func (s *StoreSuite) TestListIDs_EmptyDirectory() {
	ids, err := s.store.ListIDs()
	require.NoError(s.T(), err)
	require.Len(s.T(), ids, 0)
}

func (s *StoreSuite) TestListIDs_NonExistentDirectory() {
	store := OpenDir(filepath.Join(s.tempDir, "nonexistent-ids"))
	ids, err := store.ListIDs()
	require.NoError(s.T(), err)
	require.Nil(s.T(), ids)
}

func (s *StoreSuite) TestEnsureDir() {
	newDir := filepath.Join(s.tempDir, "new-tickets")
	store := OpenDir(newDir)

	require.NoError(s.T(), store.EnsureDir())

	info, err := os.Stat(newDir)
	require.NoError(s.T(), err)
	require.True(s.T(), info.IsDir())
}

func (s *StoreSuite) TestAtomicClaim_Success() {
	ticket := &Ticket{
		ID:      "tic-claim1",
		Status:  StatusOpen,
		Title:   "Test Claim",
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.store.Write(ticket))

	claimed, err := s.store.AtomicClaim("tic-claim1")
	require.NoError(s.T(), err)
	require.Equal(s.T(), StatusInProgress, claimed.Status)

	// Verify file was updated
	read, err := s.store.Read("tic-claim1")
	require.NoError(s.T(), err)
	require.Equal(s.T(), StatusInProgress, read.Status)
}

func (s *StoreSuite) TestAtomicClaim_AlreadyInProgress() {
	ticket := &Ticket{
		ID:      "tic-claim2",
		Status:  StatusInProgress,
		Title:   "Already In Progress",
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.store.Write(ticket))

	_, err := s.store.AtomicClaim("tic-claim2")
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, ErrAlreadyClaimed)
}

func (s *StoreSuite) TestAtomicClaim_AlreadyClosed() {
	ticket := &Ticket{
		ID:      "tic-claim3",
		Status:  StatusClosed,
		Title:   "Already Closed",
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.store.Write(ticket))

	_, err := s.store.AtomicClaim("tic-claim3")
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, ErrAlreadyClaimed)
}

func (s *StoreSuite) TestAtomicClaim_ConcurrentClaims() {
	ticket := &Ticket{
		ID:      "tic-race1",
		Status:  StatusOpen,
		Title:   "Race Condition Test",
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.store.Write(ticket))

	const numWorkers = 10
	results := make(chan error, numWorkers)

	for range numWorkers {
		go func() {
			_, err := s.store.AtomicClaim("tic-race1")
			results <- err
		}()
	}

	var successCount, failCount int
	for range numWorkers {
		err := <-results
		if err == nil {
			successCount++
		} else {
			require.ErrorIs(s.T(), err, ErrAlreadyClaimed)
			failCount++
		}
	}

	require.Equal(s.T(), 1, successCount, "exactly one worker should claim successfully")
	require.Equal(s.T(), numWorkers-1, failCount, "all other workers should fail")

	read, err := s.store.Read("tic-race1")
	require.NoError(s.T(), err)
	require.Equal(s.T(), StatusInProgress, read.Status)
}

func (s *StoreSuite) TestDelete_NonExistent() {
	err := s.store.Delete("nonexistent-ticket")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "failed to delete ticket")
}

func (s *StoreSuite) TestAtomicClaim_FileNotFound() {
	_, err := s.store.AtomicClaim("nonexistent-ticket")
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "failed to open ticket file")
}

func (s *StoreSuite) TestRead_NotFound() {
	_, err := s.store.Read("nonexistent")
	require.Error(s.T(), err)
}

func (s *StoreSuite) TestFindTicketsDir() {
	nestedDir := filepath.Join(s.tempDir, "level1", "level2")
	require.NoError(s.T(), os.MkdirAll(nestedDir, 0755))

	originalDir, err := os.Getwd()
	require.NoError(s.T(), err)
	defer func() { _ = os.Chdir(originalDir) }()

	require.NoError(s.T(), os.Chdir(nestedDir))

	found, err := FindTicketsDir()
	require.NoError(s.T(), err)

	expected, err := filepath.EvalSymlinks(filepath.Join(s.tempDir, TicketsDirName))
	require.NoError(s.T(), err)
	actual, err := filepath.EvalSymlinks(found)
	require.NoError(s.T(), err)
	require.Equal(s.T(), expected, actual)
}
