package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query [jq-filter]",
	Short: "Output tickets as JSON, optionally filtered with jq",
	Long: `Output all tickets as a JSON array. If a jq filter is provided,
the output will be piped through jq with that filter.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		jsonData, err := json.Marshal(tickets)
		if err != nil {
			return fmt.Errorf("failed to marshal tickets: %w", err)
		}

		// If no jq filter, just output JSON
		if len(args) == 0 {
			fmt.Println(string(jsonData))
			return nil
		}

		// Pipe through jq
		jqFilter := args[0]
		jqCmd := exec.Command("jq", jqFilter)
		jqCmd.Stdout = os.Stdout
		jqCmd.Stderr = os.Stderr

		// Create a pipe for stdin
		stdin, err := jqCmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}

		if err := jqCmd.Start(); err != nil {
			// If jq is not available, output raw JSON
			fmt.Println(string(jsonData))
			return nil
		}

		if _, err := stdin.Write(jsonData); err != nil {
			_ = stdin.Close()
			return fmt.Errorf("failed to write to jq: %w", err)
		}
		if err := stdin.Close(); err != nil {
			return fmt.Errorf("failed to close stdin: %w", err)
		}

		return jqCmd.Wait()
	},
}
