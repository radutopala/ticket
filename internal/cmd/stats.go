package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

// Stats holds aggregated ticket statistics.
type Stats struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"by_status"`
	ByType     map[string]int `json:"by_type"`
	ByAssignee map[string]int `json:"by_assignee"`
}

var statsFlags struct {
	json bool
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display project metrics",
	Long: `Display aggregated statistics about tickets in the project.

Shows total ticket count along with breakdowns by status, type, and assignee.

Examples:
  tk stats         # Show stats in human-readable format
  tk stats --json  # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		stats := computeStats(tickets)

		if statsFlags.json {
			return outputStatsJSON(cmd.OutOrStdout(), stats)
		}

		return runWithPager(func(w io.Writer) error {
			return outputStatsText(w, stats)
		})
	},
}

func computeStats(tickets []*domain.Ticket) Stats {
	stats := Stats{
		Total:      len(tickets),
		ByStatus:   make(map[string]int),
		ByType:     make(map[string]int),
		ByAssignee: make(map[string]int),
	}

	for _, t := range tickets {
		stats.ByStatus[string(t.Status)]++

		if t.Type != "" {
			stats.ByType[string(t.Type)]++
		}

		assignee := t.Assignee
		if assignee == "" {
			assignee = "unassigned"
		}
		stats.ByAssignee[assignee]++
	}

	return stats
}

func outputStatsJSON(w io.Writer, stats Stats) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func outputStatsText(w io.Writer, stats Stats) error {
	if _, err := fmt.Fprintf(w, "Total: %d tickets\n\n", stats.Total); err != nil {
		return err
	}

	// Status breakdown
	if _, err := fmt.Fprintln(w, "By Status:"); err != nil {
		return err
	}
	statusOrder := statusStrings(domain.ValidStatuses)
	maxStatusLen := maxKeyLen(statusOrder)
	for _, status := range statusOrder {
		count := stats.ByStatus[status]
		if _, err := fmt.Fprintf(w, "  %-*s %d\n", maxStatusLen+1, status+":", count); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	// Type breakdown
	if _, err := fmt.Fprintln(w, "By Type:"); err != nil {
		return err
	}
	typeOrder := typeStrings(domain.ValidTypes)
	maxTypeLen := maxKeyLen(typeOrder)
	for _, typ := range typeOrder {
		count := stats.ByType[typ]
		if count > 0 {
			if _, err := fmt.Fprintf(w, "  %-*s %d\n", maxTypeLen+1, typ+":", count); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	// Assignee breakdown
	if _, err := fmt.Fprintln(w, "By Assignee:"); err != nil {
		return err
	}
	assignees := sortedKeys(stats.ByAssignee)
	maxAssigneeLen := maxKeyLen(assignees)
	for _, assignee := range assignees {
		count := stats.ByAssignee[assignee]
		if _, err := fmt.Fprintf(w, "  %-*s %d\n", maxAssigneeLen+1, assignee+":", count); err != nil {
			return err
		}
	}

	return nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func maxKeyLen(keys []string) int {
	max := 0
	for _, k := range keys {
		if len(k) > max {
			max = len(k)
		}
	}
	return max
}

// statusStrings converts a slice of Status to a slice of strings.
func statusStrings(statuses []domain.Status) []string {
	result := make([]string, len(statuses))
	for i, s := range statuses {
		result[i] = string(s)
	}
	return result
}

// typeStrings converts a slice of Type to a slice of strings.
func typeStrings(types []domain.Type) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

func init() {
	statsCmd.Flags().BoolVar(&statsFlags.json, "json", false, "Output as JSON")
}
