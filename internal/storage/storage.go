// Package storage handles file operations for tickets.
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/radutopala/ticket/internal/domain"
)

// ErrAlreadyClaimed is returned when trying to claim a ticket that is not open.
var ErrAlreadyClaimed = errors.New("ticket already claimed")

const (
	// TicketsDirName is the name of the tickets directory.
	TicketsDirName = ".tickets"
	// IDPrefix is the prefix for ticket IDs.
	IDPrefix = "tic"
	// IDRandomLength is the length of the random part of the ID.
	IDRandomLength = 4
)

// Storage handles ticket file operations.
type Storage struct {
	ticketsDir string
}

// New creates a new Storage instance.
func New(ticketsDir string) *Storage {
	return &Storage{
		ticketsDir: ticketsDir,
	}
}

// FindTicketsDir finds the .tickets directory by walking up parent directories.
func FindTicketsDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		ticketsPath := filepath.Join(dir, TicketsDirName)
		if info, err := os.Stat(ticketsPath); err == nil && info.IsDir() {
			return ticketsPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding .tickets
			return "", fmt.Errorf("no %s directory found", TicketsDirName)
		}
		dir = parent
	}
}

// TicketsDir returns the tickets directory path.
func (s *Storage) TicketsDir() string {
	return s.ticketsDir
}

// GenerateID generates a unique ticket ID.
func GenerateID() (string, error) {
	bytes := make([]byte, IDRandomLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return fmt.Sprintf("%s-%s", IDPrefix, hex.EncodeToString(bytes)[:IDRandomLength]), nil
}

// List returns all tickets in the storage directory.
func (s *Storage) List() ([]*domain.Ticket, error) {
	entries, err := os.ReadDir(s.ticketsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read tickets directory: %w", err)
	}

	var tickets []*domain.Ticket
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".md")
		ticket, err := s.Read(id)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

// Read reads a ticket by ID.
func (s *Storage) Read(id string) (*domain.Ticket, error) {
	path := filepath.Join(s.ticketsDir, id+".md")
	return domain.ParseFromFile(path)
}

// Write saves a ticket to storage.
func (s *Storage) Write(ticket *domain.Ticket) error {
	path := filepath.Join(s.ticketsDir, ticket.ID+".md")
	return ticket.WriteToFile(path)
}

// Delete removes a ticket from storage.
func (s *Storage) Delete(id string) error {
	path := filepath.Join(s.ticketsDir, id+".md")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete ticket %s: %w", id, err)
	}
	return nil
}

// Exists checks if a ticket exists.
func (s *Storage) Exists(id string) bool {
	path := filepath.Join(s.ticketsDir, id+".md")
	_, err := os.Stat(path)
	return err == nil
}

// ResolveID resolves a partial ID to a full ticket ID.
// Returns the full ID if exactly one match is found.
// Returns an error if no match or multiple matches are found.
func (s *Storage) ResolveID(partial string) (string, error) {
	entries, err := os.ReadDir(s.ticketsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("ticket not found: %s", partial)
		}
		return "", fmt.Errorf("failed to read tickets directory: %w", err)
	}

	var matches []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".md")
		if strings.Contains(id, partial) {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("ticket not found: %s", partial)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous ID %s matches: %s", partial, strings.Join(matches, ", "))
	}
}

// ListIDs returns all ticket IDs.
func (s *Storage) ListIDs() ([]string, error) {
	entries, err := os.ReadDir(s.ticketsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read tickets directory: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".md"))
	}

	return ids, nil
}

// EnsureDir ensures the tickets directory exists.
func (s *Storage) EnsureDir() error {
	return os.MkdirAll(s.ticketsDir, 0755)
}

// AtomicClaim atomically claims a ticket by acquiring an exclusive file lock,
// checking the current status, and updating to in_progress only if the ticket is open.
// Returns ErrAlreadyClaimed if the ticket is not in open status.
func (s *Storage) AtomicClaim(id string) (*domain.Ticket, error) {
	path := filepath.Join(s.ticketsDir, id+".md")

	// Open file for read/write
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open ticket file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Acquire exclusive lock (blocking)
	if err := lockFile(file); err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() { _ = unlockFile(file) }()

	// Read current content
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read ticket: %w", err)
	}

	ticket, err := domain.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ticket: %w", err)
	}

	// Check if claimable
	if ticket.Status != domain.StatusOpen {
		return nil, fmt.Errorf("%w: status is %s", ErrAlreadyClaimed, ticket.Status)
	}

	// Update status
	ticket.Status = domain.StatusInProgress

	// Write back (truncate and write)
	newData, err := ticket.Render()
	if err != nil {
		return nil, fmt.Errorf("failed to render ticket: %w", err)
	}

	if err := file.Truncate(0); err != nil {
		return nil, fmt.Errorf("failed to truncate file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}
	if _, err := file.Write(newData); err != nil {
		return nil, fmt.Errorf("failed to write ticket: %w", err)
	}

	return ticket, nil
}
