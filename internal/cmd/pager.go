package cmd

import (
	"io"
	"os"
	"os/exec"
)

// getPagerCommand returns the pager command to use.
// It checks TICKET_PAGER first, then PAGER, and returns empty if neither is set.
func getPagerCommand() string {
	if pager := os.Getenv("TICKET_PAGER"); pager != "" {
		return pager
	}
	return os.Getenv("PAGER")
}

// runWithPager executes a function that writes to a writer.
// If a pager is configured, output is piped through it.
// If no pager is configured, output goes directly to stdout.
func runWithPager(fn func(w io.Writer) error) error {
	pager := getPagerCommand()
	if pager == "" {
		return fn(os.Stdout)
	}

	cmd := exec.Command("sh", "-c", pager)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fn(os.Stdout)
	}

	if err := cmd.Start(); err != nil {
		return fn(os.Stdout)
	}

	fnErr := fn(stdin)
	_ = stdin.Close()

	// Wait for pager to finish, but prefer returning fn error if any
	_ = cmd.Wait()

	return fnErr
}
