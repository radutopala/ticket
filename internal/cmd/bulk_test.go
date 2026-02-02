package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BulkSuite struct {
	suite.Suite
}

func TestBulkSuite(t *testing.T) {
	suite.Run(t, new(BulkSuite))
}

// Note: filterTickets is tested comprehensively in list_test.go TestFilterTickets

func (s *BulkSuite) TestBulkFlagsInit() {
	// Verify flags are registered
	require.NotNil(s.T(), bulkCmd.PersistentFlags().Lookup("tag"))
	require.NotNil(s.T(), bulkCmd.PersistentFlags().Lookup("status"))
	require.NotNil(s.T(), bulkCmd.PersistentFlags().Lookup("assignee"))
	require.NotNil(s.T(), bulkCmd.PersistentFlags().Lookup("dry-run"))
}

func (s *BulkSuite) TestBulkSubcommandsExist() {
	// Verify subcommands are registered
	closeCmd, _, err := bulkCmd.Find([]string{"close"})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), closeCmd)

	reopenCmd, _, err := bulkCmd.Find([]string{"reopen"})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), reopenCmd)

	startCmd, _, err := bulkCmd.Find([]string{"start"})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), startCmd)
}
