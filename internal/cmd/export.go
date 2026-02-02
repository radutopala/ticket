package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var exportFlags struct {
	format string
	output string
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export tickets to JSON or CSV format",
	Long: `Export all tickets to a specified format (JSON or CSV).
Output goes to stdout by default, or to a file with --output.

Examples:
  tk export                              # Export as JSON to stdout
  tk export --format=json > tickets.json # Export as JSON, redirect to file
  tk export --format=csv > tickets.csv   # Export as CSV
  tk export --output=backup.json         # Export to file directly
  tk export --format=csv --output=t.csv  # Export CSV to file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		var w io.Writer = os.Stdout
		if exportFlags.output != "" {
			f, err := os.Create(exportFlags.output)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer func() { _ = f.Close() }()
			w = f
		}

		switch exportFlags.format {
		case "json":
			return exportJSON(w, tickets)
		case "csv":
			return exportCSV(w, tickets)
		default:
			return fmt.Errorf("unsupported format: %s (use json or csv)", exportFlags.format)
		}
	},
}

func exportJSON(w io.Writer, tickets any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tickets); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func exportCSV(w io.Writer, tickets any) error {
	ticketSlice, ok := tickets.([]*struct {
		ID          string
		Status      string
		Type        string
		Priority    int
		Assignee    string
		Parent      string
		ExternalRef string
		Tags        []string
		Deps        []string
		Links       []string
		Created     string
		Title       string
		Description string
		Design      string
		Acceptance  string
	})
	// Type assertion won't work directly, let's use the actual type
	_ = ticketSlice
	_ = ok

	// Re-marshal and unmarshal to get a generic structure
	jsonData, err := json.Marshal(tickets)
	if err != nil {
		return fmt.Errorf("failed to marshal tickets: %w", err)
	}

	var ticketMaps []map[string]any
	if err := json.Unmarshal(jsonData, &ticketMaps); err != nil {
		return fmt.Errorf("failed to unmarshal tickets: %w", err)
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// CSV header
	headers := []string{
		"ID", "Status", "Type", "Priority", "Assignee", "Parent",
		"ExternalRef", "Tags", "Deps", "Links", "Created",
		"Title", "Description", "Design", "Acceptance",
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, t := range ticketMaps {
		row := []string{
			getString(t, "ID"),
			getString(t, "Status"),
			getString(t, "Type"),
			fmt.Sprintf("%v", t["Priority"]),
			getString(t, "Assignee"),
			getString(t, "Parent"),
			getString(t, "ExternalRef"),
			joinStrings(t, "Tags"),
			joinStrings(t, "Deps"),
			joinStrings(t, "Links"),
			getString(t, "Created"),
			getString(t, "Title"),
			getString(t, "Description"),
			getString(t, "Design"),
			getString(t, "Acceptance"),
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func joinStrings(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if slice, ok := v.([]any); ok {
			strs := make([]string, len(slice))
			for i, s := range slice {
				strs[i] = fmt.Sprintf("%v", s)
			}
			return strings.Join(strs, ";")
		}
	}
	return ""
}

func init() {
	exportCmd.Flags().StringVar(&exportFlags.format, "format", "json", "Output format (json or csv)")
	exportCmd.Flags().StringVarP(&exportFlags.output, "output", "o", "", "Output file (default: stdout)")
}
