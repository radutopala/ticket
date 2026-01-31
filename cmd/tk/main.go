// Package main is the entry point for the tk CLI.
package main

import (
	"os"

	"github.com/radutopala/ticket/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
