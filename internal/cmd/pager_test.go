package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PagerSuite struct {
	suite.Suite
	originalTicketPager string
	originalPager       string
	ticketPagerSet      bool
	pagerSet            bool
}

func TestPagerSuite(t *testing.T) {
	suite.Run(t, new(PagerSuite))
}

func (s *PagerSuite) SetupTest() {
	// Save original values
	s.originalTicketPager, s.ticketPagerSet = os.LookupEnv("TICKET_PAGER")
	s.originalPager, s.pagerSet = os.LookupEnv("PAGER")

	// Clear both env vars for clean tests
	os.Unsetenv("TICKET_PAGER")
	os.Unsetenv("PAGER")
}

func (s *PagerSuite) TearDownTest() {
	// Restore original values
	if s.ticketPagerSet {
		os.Setenv("TICKET_PAGER", s.originalTicketPager)
	} else {
		os.Unsetenv("TICKET_PAGER")
	}

	if s.pagerSet {
		os.Setenv("PAGER", s.originalPager)
	} else {
		os.Unsetenv("PAGER")
	}
}

func (s *PagerSuite) TestGetPagerCommand_NoEnvVars() {
	// With no env vars set, should return empty string
	result := getPagerCommand()
	require.Equal(s.T(), "", result)
}

func (s *PagerSuite) TestGetPagerCommand_TicketPagerSet() {
	os.Setenv("TICKET_PAGER", "less -R")

	result := getPagerCommand()
	require.Equal(s.T(), "less -R", result)
}

func (s *PagerSuite) TestGetPagerCommand_OnlyPagerSet() {
	os.Setenv("PAGER", "more")

	result := getPagerCommand()
	require.Equal(s.T(), "more", result)
}

func (s *PagerSuite) TestGetPagerCommand_TicketPagerTakesPrecedence() {
	os.Setenv("TICKET_PAGER", "less -R")
	os.Setenv("PAGER", "more")

	result := getPagerCommand()
	require.Equal(s.T(), "less -R", result)
}

func (s *PagerSuite) TestGetPagerCommand_TicketPagerEmpty() {
	// Empty string should fall through to PAGER
	os.Setenv("TICKET_PAGER", "")
	os.Setenv("PAGER", "more")

	result := getPagerCommand()
	require.Equal(s.T(), "more", result)
}

func (s *PagerSuite) TestRunWithPager_NoPager() {
	// With no pager configured, fn should write directly to stdout
	var written string
	err := runWithPager(func(w io.Writer) error {
		// We can't easily capture stdout in tests, but we verify no error
		written = "test output"
		return nil
	})

	require.NoError(s.T(), err)
	require.Equal(s.T(), "test output", written)
}

func (s *PagerSuite) TestRunWithPager_WithPager() {
	// Use cat as a simple pager that passes through
	os.Setenv("TICKET_PAGER", "cat")

	var fnCalled bool
	err := runWithPager(func(w io.Writer) error {
		fnCalled = true
		_, err := w.Write([]byte("test output through pager"))
		return err
	})

	require.NoError(s.T(), err)
	require.True(s.T(), fnCalled)
}

func (s *PagerSuite) TestRunWithPager_FunctionError() {
	// Use cat as pager
	os.Setenv("TICKET_PAGER", "cat")

	expectedErr := errors.New("function error")
	err := runWithPager(func(w io.Writer) error {
		return expectedErr
	})

	require.Equal(s.T(), expectedErr, err)
}

func (s *PagerSuite) TestRunWithPager_NoPagerFunctionError() {
	// With no pager, function error should be returned
	expectedErr := errors.New("function error")
	err := runWithPager(func(w io.Writer) error {
		return expectedErr
	})

	require.Equal(s.T(), expectedErr, err)
}

func (s *PagerSuite) TestRunWithPager_WithTrueCommand() {
	// Use 'true' which immediately exits successfully
	os.Setenv("TICKET_PAGER", "true")

	err := runWithPager(func(w io.Writer) error {
		_, _ = w.Write([]byte("test"))
		return nil
	})

	require.NoError(s.T(), err)
}

func (s *PagerSuite) TestRunWithPager_OutputPassedToPager() {
	// Create a temp file to capture output
	tmpFile, err := os.CreateTemp("", "pager_test_*")
	require.NoError(s.T(), err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Use cat with output redirection
	os.Setenv("TICKET_PAGER", "cat > "+tmpFile.Name())

	err = runWithPager(func(w io.Writer) error {
		_, err := w.Write([]byte("captured output"))
		return err
	})

	require.NoError(s.T(), err)

	// Read the output file
	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(s.T(), err)
	require.Equal(s.T(), "captured output", string(content))
}

func (s *PagerSuite) TestRunWithPager_MultipleWrites() {
	tmpFile, err := os.CreateTemp("", "pager_test_*")
	require.NoError(s.T(), err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	os.Setenv("TICKET_PAGER", "cat > "+tmpFile.Name())

	err = runWithPager(func(w io.Writer) error {
		_, _ = w.Write([]byte("first "))
		_, _ = w.Write([]byte("second "))
		_, _ = w.Write([]byte("third"))
		return nil
	})

	require.NoError(s.T(), err)

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(s.T(), err)
	require.Equal(s.T(), "first second third", string(content))
}

func (s *PagerSuite) TestRunWithPager_LargeOutput() {
	tmpFile, err := os.CreateTemp("", "pager_test_*")
	require.NoError(s.T(), err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	os.Setenv("TICKET_PAGER", "cat > "+tmpFile.Name())

	// Write a large amount of data
	var largeData bytes.Buffer
	for i := 0; i < 1000; i++ {
		largeData.WriteString("line of test data\n")
	}
	expected := largeData.String()

	err = runWithPager(func(w io.Writer) error {
		_, err := w.Write([]byte(expected))
		return err
	})

	require.NoError(s.T(), err)

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(s.T(), err)
	require.Equal(s.T(), expected, string(content))
}
