package ticket

// Stats holds aggregated ticket statistics.
type Stats struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"by_status"`
	ByType     map[string]int `json:"by_type"`
	ByAssignee map[string]int `json:"by_assignee"`
}

// ComputeStats computes statistics from a list of tickets.
func ComputeStats(tickets []*Ticket) Stats {
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
