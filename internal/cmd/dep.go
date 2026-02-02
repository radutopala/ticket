package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
)

var depCmd = &cobra.Command{
	Use:   "dep",
	Short: "Manage ticket dependencies",
	Long:  `Manage dependencies between tickets. Dependencies block a ticket until resolved.`,
}

var depAddCmd = &cobra.Command{
	Use:   "add <ticket-id> <dep-id>",
	Short: "Add a dependency to a ticket",
	Long:  `Add a dependency from ticket to dep-id. The ticket will be blocked until dep-id is closed.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketID, err := store.ResolveID(args[0])
		if err != nil {
			return fmt.Errorf("invalid ticket: %w", err)
		}

		depID, err := store.ResolveID(args[1])
		if err != nil {
			return fmt.Errorf("invalid dependency: %w", err)
		}

		if ticketID == depID {
			return fmt.Errorf("ticket cannot depend on itself")
		}

		ticket, err := store.Read(ticketID)
		if err != nil {
			return err
		}

		// Check if dependency already exists
		for _, d := range ticket.Deps {
			if d == depID {
				return fmt.Errorf("dependency %s already exists", depID)
			}
		}

		// Check for cycles
		if err := checkCycle(ticketID, depID); err != nil {
			return err
		}

		ticket.Deps = append(ticket.Deps, depID)
		if err := store.Write(ticket); err != nil {
			return err
		}

		fmt.Printf("Added dependency: %s -> %s\n", ticketID, depID)
		return nil
	},
}

var depRemoveCmd = &cobra.Command{
	Use:     "remove <ticket-id> <dep-id>",
	Aliases: []string{"rm"},
	Short:   "Remove a dependency from a ticket",
	Long:    `Remove a dependency from a ticket.`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketID, err := store.ResolveID(args[0])
		if err != nil {
			return fmt.Errorf("invalid ticket: %w", err)
		}

		depID, err := store.ResolveID(args[1])
		if err != nil {
			return fmt.Errorf("invalid dependency: %w", err)
		}

		ticket, err := store.Read(ticketID)
		if err != nil {
			return err
		}

		newDeps, found := removeFromSlice(ticket.Deps, depID)
		if !found {
			return fmt.Errorf("dependency %s not found on %s", depID, ticketID)
		}

		ticket.Deps = newDeps
		if err := store.Write(ticket); err != nil {
			return err
		}

		fmt.Printf("Removed dependency: %s -> %s\n", ticketID, depID)
		return nil
	},
}

// undepCmd is an alias for dep remove
var undepCmd = &cobra.Command{
	Use:   "undep <ticket-id> <dep-id>",
	Short: "Remove a dependency (alias for dep remove)",
	Long:  `Remove a dependency from a ticket. This is an alias for 'dep remove'.`,
	Args:  cobra.ExactArgs(2),
	RunE:  depRemoveCmd.RunE,
}

var depTreeFlags struct {
	full bool
}

var depTreeCmd = &cobra.Command{
	Use:   "tree [ticket-id]",
	Short: "Show dependency tree",
	Long:  `Show the dependency tree for a ticket. Use --full to show all tickets.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		ticketMap := make(map[string]*domain.Ticket)
		for _, t := range tickets {
			ticketMap[t.ID] = t
		}

		if depTreeFlags.full || len(args) == 0 {
			// Show full dependency tree for all root tickets
			roots := findRootTickets(tickets, ticketMap)
			for _, root := range roots {
				printDepTree(root, ticketMap, "", true)
			}
			return nil
		}

		ticketID, err := store.ResolveID(args[0])
		if err != nil {
			return err
		}

		ticket, ok := ticketMap[ticketID]
		if !ok {
			return fmt.Errorf("ticket not found: %s", ticketID)
		}

		printDepTree(ticket, ticketMap, "", true)
		return nil
	},
}

// checkCycle checks if adding depID as a dependency of ticketID would create a cycle.
func checkCycle(ticketID, depID string) error {
	tickets, err := store.List()
	if err != nil {
		return err
	}

	ticketMap := make(map[string]*domain.Ticket)
	for _, t := range tickets {
		ticketMap[t.ID] = t
	}

	// Build adjacency list including the proposed new edge
	deps := make(map[string][]string)
	for _, t := range tickets {
		deps[t.ID] = t.Deps
	}

	// Add proposed dependency
	deps[ticketID] = append(deps[ticketID], depID)

	// Check if ticketID is reachable from depID (which would mean a cycle)
	visited := make(map[string]bool)
	var hasCycle func(current, target string) bool
	hasCycle = func(current, target string) bool {
		if current == target {
			return true
		}
		if visited[current] {
			return false
		}
		visited[current] = true

		for _, dep := range deps[current] {
			if hasCycle(dep, target) {
				return true
			}
		}
		return false
	}

	if hasCycle(depID, ticketID) {
		return fmt.Errorf("adding dependency would create a cycle: %s -> %s", ticketID, depID)
	}

	return nil
}

// findRootTickets returns tickets that are not dependencies of any other ticket.
func findRootTickets(tickets []*domain.Ticket, ticketMap map[string]*domain.Ticket) []*domain.Ticket {
	// Find tickets that are dependencies
	isDep := make(map[string]bool)
	for _, t := range tickets {
		for _, d := range t.Deps {
			isDep[d] = true
		}
	}

	// Return tickets that are not dependencies
	var roots []*domain.Ticket
	for _, t := range tickets {
		if !isDep[t.ID] && t.Status != domain.StatusClosed {
			roots = append(roots, t)
		}
	}

	return roots
}

// formatTreeNode returns a formatted string for a ticket node.
func formatTreeNode(ticket *domain.Ticket) string {
	return fmt.Sprintf("%s %s - %s", statusIndicator(ticket.Status), ticket.ID, ticket.Title)
}

// formatMissingNode returns a formatted string for a missing dependency.
func formatMissingNode(depID string) string {
	return fmt.Sprintf("[?] %s - (not found)", depID)
}

// buildDepTreeString builds a dependency tree string recursively.
func buildDepTreeString(ticket *domain.Ticket, ticketMap map[string]*domain.Ticket, prefix string, isLast bool) string {
	var sb strings.Builder

	// Determine connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	// Format this ticket
	nodeStr := formatTreeNode(ticket)
	if prefix == "" {
		sb.WriteString(nodeStr + "\n")
	} else {
		sb.WriteString(prefix + connector + nodeStr + "\n")
	}

	// Calculate prefix for children
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	// Build dependency strings
	deps := ticket.Deps
	for i, depID := range deps {
		dep, ok := ticketMap[depID]
		if !ok {
			// Dependency ticket not found, show as missing
			depConnector := "├── "
			if i == len(deps)-1 {
				depConnector = "└── "
			}
			sb.WriteString(childPrefix + depConnector + formatMissingNode(depID) + "\n")
			continue
		}

		sb.WriteString(buildDepTreeString(dep, ticketMap, childPrefix, i == len(deps)-1))
	}

	return sb.String()
}

// printDepTree prints a dependency tree recursively.
func printDepTree(ticket *domain.Ticket, ticketMap map[string]*domain.Ticket, prefix string, isLast bool) {
	fmt.Print(buildDepTreeString(ticket, ticketMap, prefix, isLast))
}

// statusIndicator returns a status indicator for display.
func statusIndicator(status domain.Status) string {
	if symbol, ok := domain.StatusSymbols[status]; ok {
		return symbol
	}
	return "[?]"
}

// TopologicalSort returns tickets in topological order based on dependencies.
// Dependencies come before dependents in the returned slice.
func TopologicalSort(tickets []*domain.Ticket) ([]*domain.Ticket, error) {
	ticketMap := make(map[string]*domain.Ticket)
	for _, t := range tickets {
		ticketMap[t.ID] = t
	}

	// Kahn's algorithm
	inDegree := make(map[string]int)
	for _, t := range tickets {
		if _, ok := inDegree[t.ID]; !ok {
			inDegree[t.ID] = 0
		}
		for _, dep := range t.Deps {
			inDegree[t.ID]++
			if _, ok := inDegree[dep]; !ok {
				inDegree[dep] = 0
			}
		}
	}

	// Find all tickets with no dependencies
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []*domain.Ticket
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if t, ok := ticketMap[id]; ok {
			sorted = append(sorted, t)
		}

		// Reduce in-degree of dependent tickets
		for _, t := range tickets {
			for _, dep := range t.Deps {
				if dep == id {
					inDegree[t.ID]--
					if inDegree[t.ID] == 0 {
						queue = append(queue, t.ID)
					}
				}
			}
		}
	}

	if len(sorted) != len(tickets) {
		return nil, fmt.Errorf("cycle detected in dependencies")
	}

	return sorted, nil
}

// DetectCycles finds all cycles in the dependency graph.
func DetectCycles(tickets []*domain.Ticket) [][]string {
	ticketMap := make(map[string]*domain.Ticket)
	for _, t := range tickets {
		ticketMap[t.ID] = t
	}

	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(id string)
	dfs = func(id string) {
		visited[id] = true
		recStack[id] = true
		path = append(path, id)

		t, ok := ticketMap[id]
		if !ok {
			path = path[:len(path)-1]
			recStack[id] = false
			return
		}

		for _, dep := range t.Deps {
			if !visited[dep] {
				dfs(dep)
			} else if recStack[dep] {
				// Found a cycle
				cycleStart := -1
				for i, p := range path {
					if p == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		recStack[id] = false
	}

	for _, t := range tickets {
		if !visited[t.ID] {
			dfs(t.ID)
		}
	}

	return cycles
}

var depCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for dependency cycles",
	Long:  `Check for cycles in the dependency graph.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		cycles := DetectCycles(tickets)
		if len(cycles) == 0 {
			fmt.Println("No cycles detected")
			return nil
		}

		fmt.Printf("Found %d cycle(s):\n", len(cycles))
		for i, cycle := range cycles {
			fmt.Printf("  %d: %s\n", i+1, strings.Join(cycle, " -> "))
		}

		return fmt.Errorf("dependency cycles detected")
	},
}

func init() {
	depTreeCmd.Flags().BoolVar(&depTreeFlags.full, "full", false, "Show full dependency tree for all tickets")

	depCmd.AddCommand(depAddCmd)
	depCmd.AddCommand(depRemoveCmd)
	depCmd.AddCommand(depTreeCmd)
	depCmd.AddCommand(depCheckCmd)
}
