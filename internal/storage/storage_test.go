package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/radutopala/ticket/internal/domain"
)

type StorageSuite struct {
	suite.Suite
	tempDir string
	storage *Storage
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

func (s *StorageSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "ticket-storage-test-*")
	require.NoError(s.T(), err)

	ticketsDir := filepath.Join(s.tempDir, TicketsDirName)
	require.NoError(s.T(), os.MkdirAll(ticketsDir, 0755))

	s.storage = New(ticketsDir)
}

func (s *StorageSuite) TearDownTest() {
	_ = os.RemoveAll(s.tempDir)
}

func (s *StorageSuite) TestGenerateID() {
	id, err := GenerateID()
	require.NoError(s.T(), err)
	require.True(s.T(), len(id) > 0)
	require.Contains(s.T(), id, IDPrefix+"-")
}

func (s *StorageSuite) TestGenerateIDUnique() {
	ids := make(map[string]bool)
	// Test fewer iterations since we have limited ID space (4 hex chars = 65536 values)
	for range 10 {
		id, err := GenerateID()
		require.NoError(s.T(), err)
		require.False(s.T(), ids[id], "duplicate ID generated: %s", id)
		ids[id] = true
	}
}

func (s *StorageSuite) TestWriteAndRead() {
	ticket := &domain.Ticket{
		ID:          "tic-test",
		Status:      domain.StatusOpen,
		Type:        domain.TypeTask,
		Priority:    2,
		Assignee:    "Developer",
		Created:     time.Now().UTC().Truncate(time.Second),
		Title:       "Test Ticket",
		Description: "Test description",
	}

	err := s.storage.Write(ticket)
	require.NoError(s.T(), err)

	read, err := s.storage.Read("tic-test")
	require.NoError(s.T(), err)
	require.Equal(s.T(), ticket.ID, read.ID)
	require.Equal(s.T(), ticket.Status, read.Status)
	require.Equal(s.T(), ticket.Type, read.Type)
	require.Equal(s.T(), ticket.Priority, read.Priority)
	require.Equal(s.T(), ticket.Assignee, read.Assignee)
	require.Equal(s.T(), ticket.Title, read.Title)
}

func (s *StorageSuite) TestList() {
	tickets := []*domain.Ticket{
		{ID: "tic-aaa1", Status: domain.StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-bbb2", Status: domain.StatusClosed, Created: time.Now().UTC()},
		{ID: "tic-ccc3", Status: domain.StatusInProgress, Created: time.Now().UTC()},
	}

	for _, t := range tickets {
		require.NoError(s.T(), s.storage.Write(t))
	}

	list, err := s.storage.List()
	require.NoError(s.T(), err)
	require.Len(s.T(), list, 3)
}

func (s *StorageSuite) TestDelete() {
	ticket := &domain.Ticket{
		ID:      "tic-del1",
		Status:  domain.StatusOpen,
		Created: time.Now().UTC(),
	}

	require.NoError(s.T(), s.storage.Write(ticket))
	require.True(s.T(), s.storage.Exists("tic-del1"))

	require.NoError(s.T(), s.storage.Delete("tic-del1"))
	require.False(s.T(), s.storage.Exists("tic-del1"))
}

func (s *StorageSuite) TestExists() {
	require.False(s.T(), s.storage.Exists("tic-nonexistent"))

	ticket := &domain.Ticket{
		ID:      "tic-exists",
		Status:  domain.StatusOpen,
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.storage.Write(ticket))
	require.True(s.T(), s.storage.Exists("tic-exists"))
}

func (s *StorageSuite) TestResolveID() {
	tickets := []*domain.Ticket{
		{ID: "tic-abc1", Status: domain.StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-def2", Status: domain.StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-abc3", Status: domain.StatusOpen, Created: time.Now().UTC()},
	}

	for _, t := range tickets {
		require.NoError(s.T(), s.storage.Write(t))
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
			got, err := s.storage.ResolveID(tt.partial)
			if tt.wantErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), tt.want, got)
		})
	}
}

func (s *StorageSuite) TestListIDs() {
	tickets := []*domain.Ticket{
		{ID: "tic-id1", Status: domain.StatusOpen, Created: time.Now().UTC()},
		{ID: "tic-id2", Status: domain.StatusOpen, Created: time.Now().UTC()},
	}

	for _, t := range tickets {
		require.NoError(s.T(), s.storage.Write(t))
	}

	ids, err := s.storage.ListIDs()
	require.NoError(s.T(), err)
	require.Len(s.T(), ids, 2)
	require.Contains(s.T(), ids, "tic-id1")
	require.Contains(s.T(), ids, "tic-id2")
}

func (s *StorageSuite) TestFindTicketsDir() {
	// Create a nested directory structure
	nestedDir := filepath.Join(s.tempDir, "level1", "level2")
	require.NoError(s.T(), os.MkdirAll(nestedDir, 0755))

	// Save current dir
	originalDir, err := os.Getwd()
	require.NoError(s.T(), err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to nested dir
	require.NoError(s.T(), os.Chdir(nestedDir))

	// Should find .tickets in tempDir
	found, err := FindTicketsDir()
	require.NoError(s.T(), err)

	// Resolve symlinks for comparison (macOS has /var -> /private/var symlink)
	expected, err := filepath.EvalSymlinks(filepath.Join(s.tempDir, TicketsDirName))
	require.NoError(s.T(), err)
	actual, err := filepath.EvalSymlinks(found)
	require.NoError(s.T(), err)
	require.Equal(s.T(), expected, actual)
}

func (s *StorageSuite) TestEnsureDir() {
	newDir := filepath.Join(s.tempDir, "new-tickets")
	storage := New(newDir)

	require.NoError(s.T(), storage.EnsureDir())

	info, err := os.Stat(newDir)
	require.NoError(s.T(), err)
	require.True(s.T(), info.IsDir())
}

func (s *StorageSuite) TestAtomicClaim_Success() {
	ticket := &domain.Ticket{
		ID:      "tic-claim1",
		Status:  domain.StatusOpen,
		Title:   "Test Claim",
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.storage.Write(ticket))

	claimed, err := s.storage.AtomicClaim("tic-claim1")
	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.StatusInProgress, claimed.Status)

	// Verify file was updated
	read, err := s.storage.Read("tic-claim1")
	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.StatusInProgress, read.Status)
}

func (s *StorageSuite) TestAtomicClaim_AlreadyInProgress() {
	ticket := &domain.Ticket{
		ID:      "tic-claim2",
		Status:  domain.StatusInProgress,
		Title:   "Already In Progress",
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.storage.Write(ticket))

	_, err := s.storage.AtomicClaim("tic-claim2")
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, ErrAlreadyClaimed)
}

func (s *StorageSuite) TestAtomicClaim_AlreadyClosed() {
	ticket := &domain.Ticket{
		ID:      "tic-claim3",
		Status:  domain.StatusClosed,
		Title:   "Already Closed",
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.storage.Write(ticket))

	_, err := s.storage.AtomicClaim("tic-claim3")
	require.Error(s.T(), err)
	require.ErrorIs(s.T(), err, ErrAlreadyClaimed)
}

func (s *StorageSuite) TestAtomicClaim_ConcurrentClaims() {
	ticket := &domain.Ticket{
		ID:      "tic-race1",
		Status:  domain.StatusOpen,
		Title:   "Race Condition Test",
		Created: time.Now().UTC(),
	}
	require.NoError(s.T(), s.storage.Write(ticket))

	const numWorkers = 10
	results := make(chan error, numWorkers)

	// Launch concurrent claims
	for range numWorkers {
		go func() {
			_, err := s.storage.AtomicClaim("tic-race1")
			results <- err
		}()
	}

	// Collect results
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

	// Exactly one should succeed
	require.Equal(s.T(), 1, successCount, "exactly one worker should claim successfully")
	require.Equal(s.T(), numWorkers-1, failCount, "all other workers should fail")

	// Verify final state
	read, err := s.storage.Read("tic-race1")
	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.StatusInProgress, read.Status)
}
