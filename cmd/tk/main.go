// Package main is the entry point for the tk CLI.
package main

import (
	"os"

	"github.com/radutopala/ticket/internal/cmd"
)

// Set by goreleaser ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
