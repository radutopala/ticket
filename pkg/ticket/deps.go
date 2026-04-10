package ticket

import (
	"fmt"
	"slices"
)

// CheckCycle checks if adding depID as a dependency of ticketID would create a cycle.
func CheckCycle(tickets []*Ticket, ticketID, depID string) error {
	// Build adjacency list including the proposed new edge
	deps := make(map[string][]string)
	for _, t := range tickets {
		deps[t.ID] = t.Deps
	}

	// Add proposed dependency
	deps[ticketID] = append(append([]string{}, deps[ticketID]...), depID)

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

// TopologicalSort returns tickets in topological order based on dependencies.
// Dependencies come before dependents in the returned slice.
func TopologicalSort(tickets []*Ticket) ([]*Ticket, error) {
	ticketMap := make(map[string]*Ticket)
	for _, t := range tickets {
		ticketMap[t.ID] = t
	}

	// Kahn's algorithm
	inDegree := make(map[string]int)
	for _, t := range tickets {
		if _, ok := inDegree[t.ID]; !ok {
			inDegree[t.ID] = 0
		}
		for range t.Deps {
			inDegree[t.ID]++
		}
		for _, dep := range t.Deps {
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

	var sorted []*Ticket
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
func DetectCycles(tickets []*Ticket) [][]string {
	ticketMap := make(map[string]*Ticket)
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

// FindRootTickets returns non-closed tickets that are not dependencies of any other ticket.
func FindRootTickets(tickets []*Ticket) []*Ticket {
	isDep := make(map[string]bool)
	for _, t := range tickets {
		for _, d := range t.Deps {
			isDep[d] = true
		}
	}

	var roots []*Ticket
	for _, t := range tickets {
		if !isDep[t.ID] && t.Status != StatusClosed {
			roots = append(roots, t)
		}
	}

	return roots
}

// BuildOpenIDSet builds a set of IDs for all non-closed tickets.
func BuildOpenIDSet(tickets []*Ticket) map[string]bool {
	openIDs := make(map[string]bool)
	for _, t := range tickets {
		if t.Status != StatusClosed {
			openIDs[t.ID] = true
		}
	}
	return openIDs
}

// AddDep adds depID as a dependency of ticket, enforcing data integrity:
// no self-deps, no duplicates, and no cycles.
// The caller must persist the ticket after a successful call.
func AddDep(ticket *Ticket, depID string, allTickets []*Ticket) error {
	if ticket.ID == depID {
		return fmt.Errorf("ticket cannot depend on itself")
	}

	if slices.Contains(ticket.Deps, depID) {
		return fmt.Errorf("dependency %s already exists", depID)
	}

	if err := CheckCycle(allTickets, ticket.ID, depID); err != nil {
		return err
	}

	ticket.Deps = append(ticket.Deps, depID)
	return nil
}

// RemoveDep removes depID from ticket's dependencies.
// Returns an error if the dependency is not found.
func RemoveDep(ticket *Ticket, depID string) error {
	newDeps, found := RemoveFromSlice(ticket.Deps, depID)
	if !found {
		return fmt.Errorf("dependency %s not found on %s", depID, ticket.ID)
	}
	ticket.Deps = newDeps
	return nil
}

// RemoveFromSlice removes all occurrences of value from slice.
// Returns the new slice and whether the value was found.
func RemoveFromSlice(slice []string, value string) ([]string, bool) {
	found := false
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item == value {
			found = true
		} else {
			result = append(result, item)
		}
	}
	return result, found
}
