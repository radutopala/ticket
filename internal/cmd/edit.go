package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Open ticket in editor",
	Long:  `Open the ticket file in $EDITOR for editing. Supports partial ID matching.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := store.ResolveID(args[0])
		if err != nil {
			return err
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		ticketPath := filepath.Join(store.TicketsDir(), id+".md")

		editorCmd := exec.Command(editor, ticketPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		return nil
	},
}
