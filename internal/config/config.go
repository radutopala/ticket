// Package config handles configuration loading for the tk CLI.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// EnvTicketsDir is the environment variable for the tickets directory.
	EnvTicketsDir = "TICKETS_DIR"
	// DefaultTicketsDir is the default directory for tickets.
	DefaultTicketsDir = ".tickets"
)

// Config holds the application configuration.
type Config struct {
	TicketsDir string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	ticketsDir := os.Getenv(EnvTicketsDir)
	if ticketsDir == "" {
		// Default to .tickets in current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		ticketsDir = filepath.Join(cwd, DefaultTicketsDir)
	}

	return &Config{
		TicketsDir: ticketsDir,
	}, nil
}
