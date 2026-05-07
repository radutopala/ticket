package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	tk "github.com/radutopala/ticket/pkg/ticket"
)

var createFlags struct {
	description string
	design      string
	acceptance  string
	ticketType  string
	priority    int
	assignee    string
	externalRef string
	pr          string
	parent      string
	tags        []string
}

var createCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a new ticket",
	Long:  `Create a new ticket with the specified title and options.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := tk.ValidatePriority(createFlags.priority); err != nil {
			return err
		}

		// Validate parent exists if specified
		if createFlags.parent != "" {
			resolvedParent, err := store.ResolveID(createFlags.parent)
			if err != nil {
				return fmt.Errorf("parent ticket not found: %s", createFlags.parent)
			}
			createFlags.parent = resolvedParent
		}

		id, err := tk.GenerateID()
		if err != nil {
			return fmt.Errorf("failed to generate ID: %w", err)
		}

		assignee := createFlags.assignee
		if assignee == "" {
			assignee = getGitUserName()
		}

		ticket := &tk.Ticket{
			ID:          id,
			Status:      tk.StatusOpen,
			Priority:    createFlags.priority,
			Assignee:    assignee,
			ExternalRef: createFlags.externalRef,
			PR:          createFlags.pr,
			Parent:      createFlags.parent,
			Tags:        createFlags.tags,
			Created:     time.Now().UTC(),
			Description: createFlags.description,
			Design:      createFlags.design,
			Acceptance:  createFlags.acceptance,
		}

		if len(args) > 0 {
			ticket.Title = args[0]
		}

		if createFlags.ticketType != "" {
			t, err := tk.ParseType(createFlags.ticketType)
			if err != nil {
				return err
			}
			ticket.Type = t
		} else {
			ticket.Type = tk.TypeTask
		}

		if err := store.EnsureDir(); err != nil {
			return fmt.Errorf("failed to create tickets directory: %w", err)
		}

		if err := store.Write(ticket); err != nil {
			return fmt.Errorf("failed to write ticket: %w", err)
		}

		if jsonOutput {
			return outputJSON(cmd, ticket)
		}

		fmt.Println(id)
		return nil
	},
}

// getGitUserName returns the git user.name config value, or empty string if unavailable.
func getGitUserName() string {
	cmd := exec.Command("git", "config", "user.name")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func init() {
	createCmd.Flags().StringVarP(&createFlags.description, "description", "d", "", "Description text")
	createCmd.Flags().StringVar(&createFlags.design, "design", "", "Design notes")
	createCmd.Flags().StringVar(&createFlags.acceptance, "acceptance", "", "Acceptance criteria")
	createCmd.Flags().StringVarP(&createFlags.ticketType, "type", "t", "task", "Type (bug|feature|task|epic|chore)")
	createCmd.Flags().IntVarP(&createFlags.priority, "priority", "p", tk.DefaultPriority, fmt.Sprintf("Priority %d-%d, %d=highest", tk.MinPriority, tk.MaxPriority, tk.MinPriority))
	createCmd.Flags().StringVarP(&createFlags.assignee, "assignee", "a", "", "Assignee")
	createCmd.Flags().StringVar(&createFlags.externalRef, "external-ref", "", "External reference (e.g., gh-123, JIRA-456)")
	createCmd.Flags().StringVar(&createFlags.pr, "pr", "", "Pull/merge request reference (e.g., gh-pr-42, !123, URL)")
	createCmd.Flags().StringVar(&createFlags.parent, "parent", "", "Parent ticket ID")
	createCmd.Flags().StringSliceVar(&createFlags.tags, "tags", nil, "Comma-separated tags")
}
