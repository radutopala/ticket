package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/domain"
	"github.com/radutopala/ticket/internal/storage"
)

var createFlags struct {
	description string
	design      string
	acceptance  string
	ticketType  string
	priority    int
	assignee    string
	externalRef string
	parent      string
	tags        []string
}

var createCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a new ticket",
	Long:  `Create a new ticket with the specified title and options.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate priority (0-4)
		if createFlags.priority < 0 || createFlags.priority > 4 {
			return fmt.Errorf("invalid priority %d: must be between 0 and 4 (0=highest)", createFlags.priority)
		}

		// Validate parent exists if specified
		if createFlags.parent != "" {
			resolvedParent, err := store.ResolveID(createFlags.parent)
			if err != nil {
				return fmt.Errorf("parent ticket not found: %s", createFlags.parent)
			}
			createFlags.parent = resolvedParent
		}

		id, err := storage.GenerateID()
		if err != nil {
			return fmt.Errorf("failed to generate ID: %w", err)
		}

		assignee := createFlags.assignee
		if assignee == "" {
			assignee = getGitUserName()
		}

		ticket := &domain.Ticket{
			ID:          id,
			Status:      domain.StatusOpen,
			Priority:    createFlags.priority,
			Assignee:    assignee,
			ExternalRef: createFlags.externalRef,
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
			t, err := domain.ParseType(createFlags.ticketType)
			if err != nil {
				return err
			}
			ticket.Type = t
		} else {
			ticket.Type = domain.TypeTask
		}

		if err := store.EnsureDir(); err != nil {
			return fmt.Errorf("failed to create tickets directory: %w", err)
		}

		if err := store.Write(ticket); err != nil {
			return fmt.Errorf("failed to write ticket: %w", err)
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
	createCmd.Flags().IntVarP(&createFlags.priority, "priority", "p", 2, "Priority 0-4, 0=highest")
	createCmd.Flags().StringVarP(&createFlags.assignee, "assignee", "a", "", "Assignee")
	createCmd.Flags().StringVar(&createFlags.externalRef, "external-ref", "", "External reference (e.g., gh-123, JIRA-456)")
	createCmd.Flags().StringVar(&createFlags.parent, "parent", "", "Parent ticket ID")
	createCmd.Flags().StringSliceVar(&createFlags.tags, "tags", nil, "Comma-separated tags")
}
