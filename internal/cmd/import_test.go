package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/radutopala/ticket/internal/domain"
)

type ImportSuite struct {
	suite.Suite
}

func TestImportSuite(t *testing.T) {
	suite.Run(t, new(ImportSuite))
}

func (s *ImportSuite) TestConvertImportTicketValidStatuses() {
	testCases := []struct {
		name           string
		status         string
		expectedStatus domain.Status
	}{
		{"open status", "open", domain.StatusOpen},
		{"in_progress status", "in_progress", domain.StatusInProgress},
		{"closed status", "closed", domain.StatusClosed},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			input := importTicket{
				ID:     "tic-test",
				Status: tc.status,
				Title:  "Test Ticket",
			}

			result, err := convertImportTicket(input)

			require.NoError(s.T(), err)
			require.Equal(s.T(), tc.expectedStatus, result.Status)
		})
	}
}

func (s *ImportSuite) TestConvertImportTicketEmptyStatusDefaultsToOpen() {
	input := importTicket{
		ID:     "tic-test",
		Status: "",
		Title:  "Test Ticket",
	}

	result, err := convertImportTicket(input)

	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.StatusOpen, result.Status)
}

func (s *ImportSuite) TestConvertImportTicketInvalidStatus() {
	input := importTicket{
		ID:     "tic-test",
		Status: "invalid_status",
		Title:  "Test Ticket",
	}

	result, err := convertImportTicket(input)

	require.Error(s.T(), err)
	require.Nil(s.T(), result)
	require.Contains(s.T(), err.Error(), "invalid status")
}

func (s *ImportSuite) TestConvertImportTicketValidTypes() {
	testCases := []struct {
		name         string
		ticketType   string
		expectedType domain.Type
	}{
		{"task type", "task", domain.TypeTask},
		{"bug type", "bug", domain.TypeBug},
		{"feature type", "feature", domain.TypeFeature},
		{"epic type", "epic", domain.TypeEpic},
		{"chore type", "chore", domain.TypeChore},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			input := importTicket{
				ID:    "tic-test",
				Type:  tc.ticketType,
				Title: "Test Ticket",
			}

			result, err := convertImportTicket(input)

			require.NoError(s.T(), err)
			require.Equal(s.T(), tc.expectedType, result.Type)
		})
	}
}

func (s *ImportSuite) TestConvertImportTicketEmptyTypeDefaultsToTask() {
	input := importTicket{
		ID:    "tic-test",
		Type:  "",
		Title: "Test Ticket",
	}

	result, err := convertImportTicket(input)

	require.NoError(s.T(), err)
	require.Equal(s.T(), domain.TypeTask, result.Type)
}

func (s *ImportSuite) TestConvertImportTicketInvalidType() {
	input := importTicket{
		ID:    "tic-test",
		Type:  "invalid_type",
		Title: "Test Ticket",
	}

	result, err := convertImportTicket(input)

	require.Error(s.T(), err)
	require.Nil(s.T(), result)
	require.Contains(s.T(), err.Error(), "invalid type")
}

func (s *ImportSuite) TestConvertImportTicketZeroCreatedTimeDefaultsToNow() {
	input := importTicket{
		ID:      "tic-test",
		Title:   "Test Ticket",
		Created: time.Time{}, // Zero time
	}

	beforeConvert := time.Now().UTC()
	result, err := convertImportTicket(input)
	afterConvert := time.Now().UTC()

	require.NoError(s.T(), err)
	require.False(s.T(), result.Created.IsZero())
	require.True(s.T(), result.Created.After(beforeConvert.Add(-time.Second)))
	require.True(s.T(), result.Created.Before(afterConvert.Add(time.Second)))
}

func (s *ImportSuite) TestConvertImportTicketProvidedCreatedTimePreserved() {
	expectedTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	input := importTicket{
		ID:      "tic-test",
		Title:   "Test Ticket",
		Created: expectedTime,
	}

	result, err := convertImportTicket(input)

	require.NoError(s.T(), err)
	require.Equal(s.T(), expectedTime, result.Created)
}

func (s *ImportSuite) TestConvertImportTicketNotesConversion() {
	now := time.Now().UTC()
	input := importTicket{
		ID:    "tic-test",
		Title: "Test Ticket",
		Notes: []struct {
			Timestamp time.Time `json:"Timestamp"`
			Content   string    `json:"Content"`
		}{
			{Timestamp: now, Content: "First note"},
			{Timestamp: now.Add(time.Hour), Content: "Second note"},
		},
	}

	result, err := convertImportTicket(input)

	require.NoError(s.T(), err)
	require.Len(s.T(), result.Notes, 2)
	require.Equal(s.T(), "First note", result.Notes[0].Content)
	require.Equal(s.T(), now, result.Notes[0].Timestamp)
	require.Equal(s.T(), "Second note", result.Notes[1].Content)
	require.Equal(s.T(), now.Add(time.Hour), result.Notes[1].Timestamp)
}

func (s *ImportSuite) TestConvertImportTicketEmptyNotes() {
	input := importTicket{
		ID:    "tic-test",
		Title: "Test Ticket",
		Notes: nil,
	}

	result, err := convertImportTicket(input)

	require.NoError(s.T(), err)
	require.Empty(s.T(), result.Notes)
}

func (s *ImportSuite) TestConvertImportTicketAllFieldsCopied() {
	created := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	input := importTicket{
		ID:          "tic-full",
		Status:      "in_progress",
		Type:        "feature",
		Priority:    1,
		Assignee:    "developer",
		Parent:      "tic-parent",
		ExternalRef: "gh-123",
		Tags:        []string{"backend", "api"},
		Deps:        []string{"tic-dep1", "tic-dep2"},
		Links:       []string{"tic-link1"},
		Created:     created,
		Title:       "Full Ticket",
		Description: "A complete ticket description",
		Design:      "Design notes",
		Acceptance:  "Acceptance criteria",
	}

	result, err := convertImportTicket(input)

	require.NoError(s.T(), err)
	require.Equal(s.T(), "tic-full", result.ID)
	require.Equal(s.T(), domain.StatusInProgress, result.Status)
	require.Equal(s.T(), domain.TypeFeature, result.Type)
	require.Equal(s.T(), 1, result.Priority)
	require.Equal(s.T(), "developer", result.Assignee)
	require.Equal(s.T(), "tic-parent", result.Parent)
	require.Equal(s.T(), "gh-123", result.ExternalRef)
	require.Equal(s.T(), []string{"backend", "api"}, result.Tags)
	require.Equal(s.T(), []string{"tic-dep1", "tic-dep2"}, result.Deps)
	require.Equal(s.T(), []string{"tic-link1"}, result.Links)
	require.Equal(s.T(), created, result.Created)
	require.Equal(s.T(), "Full Ticket", result.Title)
	require.Equal(s.T(), "A complete ticket description", result.Description)
	require.Equal(s.T(), "Design notes", result.Design)
	require.Equal(s.T(), "Acceptance criteria", result.Acceptance)
}

func (s *ImportSuite) TestConvertImportTicketMinimalInput() {
	input := importTicket{
		ID: "tic-minimal",
	}

	result, err := convertImportTicket(input)

	require.NoError(s.T(), err)
	require.Equal(s.T(), "tic-minimal", result.ID)
	require.Equal(s.T(), domain.StatusOpen, result.Status)
	require.Equal(s.T(), domain.TypeTask, result.Type)
	require.Equal(s.T(), 0, result.Priority)
	require.Empty(s.T(), result.Assignee)
	require.Empty(s.T(), result.Title)
	require.Empty(s.T(), result.Notes)
}

func (s *ImportSuite) TestConvertImportTicketBothStatusAndTypeInvalid() {
	input := importTicket{
		ID:     "tic-test",
		Status: "bad_status",
		Type:   "bad_type",
		Title:  "Test Ticket",
	}

	// Status is checked first, so we expect status error
	result, err := convertImportTicket(input)

	require.Error(s.T(), err)
	require.Nil(s.T(), result)
	require.Contains(s.T(), err.Error(), "invalid status")
}

func (s *ImportSuite) TestConvertImportTicketValidStatusInvalidType() {
	input := importTicket{
		ID:     "tic-test",
		Status: "open",
		Type:   "bad_type",
		Title:  "Test Ticket",
	}

	result, err := convertImportTicket(input)

	require.Error(s.T(), err)
	require.Nil(s.T(), result)
	require.Contains(s.T(), err.Error(), "invalid type")
}
