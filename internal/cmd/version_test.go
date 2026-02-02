package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type VersionSuite struct {
	suite.Suite
}

func TestVersionSuite(t *testing.T) {
	suite.Run(t, new(VersionSuite))
}

func (s *VersionSuite) TestSetVersion() {
	// Save original values
	origVersion := version
	origCommit := commit
	origDate := date
	defer func() {
		version = origVersion
		commit = origCommit
		date = origDate
	}()

	SetVersion("1.2.3", "abc123", "2024-01-01")

	require.Equal(s.T(), "1.2.3", version)
	require.Equal(s.T(), "abc123", commit)
	require.Equal(s.T(), "2024-01-01", date)
}

func (s *VersionSuite) TestVersion() {
	// Save original values
	origVersion := version
	defer func() {
		version = origVersion
	}()

	version = "test-version"
	require.Equal(s.T(), "test-version", Version())
}

func (s *VersionSuite) TestVersionDefault() {
	// Save original values
	origVersion := version
	defer func() {
		version = origVersion
	}()

	version = "dev"
	require.Equal(s.T(), "dev", Version())
}
