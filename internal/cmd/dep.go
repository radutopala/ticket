package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
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

		ticket, err := store.Read(ticketID)
		if err != nil {
			return err
		}

		tickets, err := store.List()
		if err != nil {
			return err
		}

		if err := tk.AddDep(ticket, depID, tickets); err != nil {
			return err
		}

		if err := store.Write(ticket); err != nil {
			return err
		}

		if jsonOutput {
			return outputJSON(cmd, ticket)
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

		if err := tk.RemoveDep(ticket, depID); err != nil {
			return err
		}

		if err := store.Write(ticket); err != nil {
			return err
		}

		if jsonOutput {
			return outputJSON(cmd, ticket)
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

		ticketMap := make(map[string]*tk.Ticket)
		for _, t := range tickets {
			ticketMap[t.ID] = t
		}

		if depTreeFlags.full || len(args) == 0 {
			roots := tk.FindRootTickets(tickets)
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

var depCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for dependency cycles",
	Long:  `Check for cycles in the dependency graph.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tickets, err := store.List()
		if err != nil {
			return err
		}

		cycles := tk.DetectCycles(tickets)

		if jsonOutput {
			return outputJSON(cmd, map[string]any{
				"cycles": cycles,
				"count":  len(cycles),
			})
		}

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

// formatTreeNode returns a formatted string for a ticket node.
func formatTreeNode(ticket *tk.Ticket) string {
	return fmt.Sprintf("%s %s - %s", statusIndicator(ticket.Status), ticket.ID, ticket.Title)
}

// formatMissingNode returns a formatted string for a missing dependency.
func formatMissingNode(depID string) string {
	return fmt.Sprintf("[?] %s - (not found)", depID)
}

// buildDepTreeString builds a dependency tree string recursively.
func buildDepTreeString(ticket *tk.Ticket, ticketMap map[string]*tk.Ticket, prefix string, isLast bool) string {
	var sb strings.Builder

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	nodeStr := formatTreeNode(ticket)
	if prefix == "" {
		sb.WriteString(nodeStr + "\n")
	} else {
		sb.WriteString(prefix + connector + nodeStr + "\n")
	}

	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	deps := ticket.Deps
	for i, depID := range deps {
		dep, ok := ticketMap[depID]
		if !ok {
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
func printDepTree(ticket *tk.Ticket, ticketMap map[string]*tk.Ticket, prefix string, isLast bool) {
	fmt.Print(buildDepTreeString(ticket, ticketMap, prefix, isLast))
}

// statusIndicator returns a status indicator for display.
func statusIndicator(status tk.Status) string {
	if symbol, ok := tk.StatusSymbols[status]; ok {
		return symbol
	}
	return "[?]"
}

func init() {
	depTreeCmd.Flags().BoolVar(&depTreeFlags.full, "full", false, "Show full dependency tree for all tickets")

	depCmd.AddCommand(depAddCmd)
	depCmd.AddCommand(depRemoveCmd)
	depCmd.AddCommand(depTreeCmd)
	depCmd.AddCommand(depCheckCmd)
}
