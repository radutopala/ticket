package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

var addNoteCmd = &cobra.Command{
	Use:   "add-note <id> [text]",
	Short: "Append a timestamped note to a ticket",
	Long:  `Append a timestamped note to a ticket. Text can be provided as an argument or piped via stdin.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := store.ResolveID(args[0])
		if err != nil {
			return fmt.Errorf("failed to resolve ticket ID: %w", err)
		}

		ticket, err := store.Read(id)
		if err != nil {
			return err
		}

		// Get note text from args or stdin
		var noteText string
		if len(args) > 1 {
			noteText = strings.Join(args[1:], " ")
		} else {
			// Check if stdin has data
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				scanner := bufio.NewScanner(os.Stdin)
				var lines []string
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("error reading stdin: %w", err)
				}
				noteText = strings.Join(lines, "\n")
			}
		}

		if noteText == "" {
			return fmt.Errorf("no note text provided")
		}

		// Add the note
		note := domain.Note{
			Timestamp: time.Now().UTC(),
			Content:   noteText,
		}
		ticket.Notes = append(ticket.Notes, note)

		if err := store.Write(ticket); err != nil {
			return err
		}

		fmt.Printf("Added note to %s\n", id)
		return nil
	},
}
