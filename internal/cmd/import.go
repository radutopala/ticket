package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

var importFlags struct {
	skipExisting bool
}

// importTicket is a struct for JSON import that mirrors tk.Ticket
// but uses concrete types for unmarshaling.
type importTicket struct {
	ID          string    `json:"ID"`
	Status      string    `json:"Status"`
	Type        string    `json:"Type"`
	Priority    int       `json:"Priority"`
	Assignee    string    `json:"Assignee"`
	Parent      string    `json:"Parent"`
	ExternalRef string    `json:"ExternalRef"`
	Tags        []string  `json:"Tags"`
	Deps        []string  `json:"Deps"`
	Links       []string  `json:"Links"`
	Created     time.Time `json:"Created"`
	Title       string    `json:"Title"`
	Description string    `json:"Description"`
	Design      string    `json:"Design"`
	Acceptance  string    `json:"Acceptance"`
	Notes       []struct {
		Timestamp time.Time `json:"Timestamp"`
		Content   string    `json:"Content"`
	} `json:"Notes"`
}

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import tickets from a JSON file",
	Long: `Import tickets from a JSON file. The file should contain an array of tickets
in the same format as produced by 'tk export' or 'tk query'.

Examples:
  tk import tickets.json                  # Import tickets, fail on ID conflicts
  tk import tickets.json --skip-existing  # Skip tickets that already exist
  cat tickets.json | tk import -          # Import from stdin`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		var data []byte
		var err error

		if filePath == "-" {
			data, err = readAllFromStdin()
		} else {
			data, err = os.ReadFile(filePath)
		}
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		var tickets []importTicket
		if err := json.Unmarshal(data, &tickets); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}

		if err := store.EnsureDir(); err != nil {
			return fmt.Errorf("failed to ensure tickets directory: %w", err)
		}

		var importedTickets []*tk.Ticket
		var skipped, generated int
		for _, t := range tickets {
			// Generate ID if not provided
			if t.ID == "" {
				newID, err := tk.GenerateID()
				if err != nil {
					return fmt.Errorf("failed to generate ID: %w", err)
				}
				t.ID = newID
				generated++
			}

			// Check if ticket exists
			if store.Exists(t.ID) {
				if importFlags.skipExisting {
					skipped++
					continue
				}
				return fmt.Errorf("ticket %s already exists (use --skip-existing to skip)", t.ID)
			}

			// Convert to tk.Ticket
			ticket, err := convertImportTicket(t)
			if err != nil {
				return fmt.Errorf("failed to convert ticket %s: %w", t.ID, err)
			}

			if err := store.Write(ticket); err != nil {
				return fmt.Errorf("failed to write ticket %s: %w", t.ID, err)
			}
			importedTickets = append(importedTickets, ticket)
		}

		if jsonOutput {
			return outputJSON(cmd, importedTickets)
		}

		fmt.Printf("Imported %d ticket(s)", len(importedTickets))
		if skipped > 0 {
			fmt.Printf(", skipped %d existing", skipped)
		}
		if generated > 0 {
			fmt.Printf(", generated %d ID(s)", generated)
		}
		fmt.Println()

		return nil
	},
}

func readAllFromStdin() ([]byte, error) {
	return os.ReadFile("/dev/stdin")
}

func convertImportTicket(t importTicket) (*tk.Ticket, error) {
	// Parse status
	status := tk.StatusOpen
	if t.Status != "" {
		parsed, err := tk.ParseStatus(t.Status)
		if err != nil {
			return nil, fmt.Errorf("invalid status: %w", err)
		}
		status = parsed
	}

	// Parse type
	ticketType := tk.TypeTask
	if t.Type != "" {
		parsed, err := tk.ParseType(t.Type)
		if err != nil {
			return nil, fmt.Errorf("invalid type: %w", err)
		}
		ticketType = parsed
	}

	// Set created time if not provided
	created := t.Created
	if created.IsZero() {
		created = time.Now().UTC()
	}

	// Convert notes
	notes := make([]tk.Note, len(t.Notes))
	for i, n := range t.Notes {
		notes[i] = tk.Note{
			Timestamp: n.Timestamp,
			Content:   n.Content,
		}
	}

	return &tk.Ticket{
		ID:          t.ID,
		Status:      status,
		Type:        ticketType,
		Priority:    t.Priority,
		Assignee:    t.Assignee,
		Parent:      t.Parent,
		ExternalRef: t.ExternalRef,
		Tags:        t.Tags,
		Deps:        t.Deps,
		Links:       t.Links,
		Created:     created,
		Title:       t.Title,
		Description: t.Description,
		Design:      t.Design,
		Acceptance:  t.Acceptance,
		Notes:       notes,
	}, nil
}

func init() {
	importCmd.Flags().BoolVar(&importFlags.skipExisting, "skip-existing", false, "Skip tickets that already exist instead of failing")
}
