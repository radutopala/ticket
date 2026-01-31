package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (s *ConfigSuite) TestLoadWithEnvVar() {
	customDir := "/custom/tickets/dir"
	s.T().Setenv(EnvTicketsDir, customDir)

	cfg, err := Load()

	require.NoError(s.T(), err)
	require.NotNil(s.T(), cfg)
	require.Equal(s.T(), customDir, cfg.TicketsDir)
}

func (s *ConfigSuite) TestLoadWithDefaultDir() {
	s.T().Setenv(EnvTicketsDir, "")

	cfg, err := Load()

	require.NoError(s.T(), err)
	require.NotNil(s.T(), cfg)

	cwd, err := os.Getwd()
	require.NoError(s.T(), err)
	expectedDir := filepath.Join(cwd, DefaultTicketsDir)
	require.Equal(s.T(), expectedDir, cfg.TicketsDir)
}

func (s *ConfigSuite) TestConstants() {
	require.Equal(s.T(), "TICKETS_DIR", EnvTicketsDir)
	require.Equal(s.T(), ".tickets", DefaultTicketsDir)
}
