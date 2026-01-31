// Package cmd contains CLI command implementations.
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/radutopala/ticket/internal/config"
	"github.com/radutopala/ticket/internal/storage"
)

var (
	cfg     *config.Config
	logger  *slog.Logger
	store   *storage.Storage
)

var rootCmd = &cobra.Command{
	Use:   "tk",
	Short: "A ticket management CLI",
	Long:  `tk is a command-line tool for managing tickets stored as markdown files.`,
	Run: func(cmd *cobra.Command, args []string) {
		printHelp()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}

		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		store = storage.New(cfg.TicketsDir)

		return nil
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func printHelp() {
	helpText := `tk - minimal ticket system with dependency tracking

Usage:
  tk [command]

Available Commands:
  create [title]           Create a new ticket
    -d, --description      Description text
    --design               Design notes
    --acceptance           Acceptance criteria
    -t, --type             Type (bug|feature|task|epic|chore) [default: task]
    -p, --priority         Priority 0-4, 0=highest [default: 2]
    -a, --assignee         Assignee
    --external-ref         External reference (e.g., gh-123, JIRA-456)
    --parent               Parent ticket ID
    --tags                 Comma-separated tags (e.g., --tags ui,backend,urgent)
  show <id>                Display a ticket
  edit <id>                Open ticket in editor
  start <id>               Set ticket status to in_progress
  close <id>               Set ticket status to closed
  reopen <id>              Set ticket status to open
  status <id> <status>     Update ticket status (open|in_progress|closed)
  list                     List tickets (alias: ls)
    --status               Filter by status (open|in_progress|closed)
    -a, --assignee         Filter by assignee
    -T, --tag              Filter by tag
  ready                    List open/in_progress tickets with resolved deps
    -a, --assignee         Filter by assignee
    -T, --tag              Filter by tag
  blocked                  List open/in_progress tickets with unresolved deps
    -a, --assignee         Filter by assignee
    -T, --tag              Filter by tag
  closed                   List recently closed tickets
    --limit                Limit number of results [default: 20]
    -a, --assignee         Filter by assignee
    -T, --tag              Filter by tag
  dep add <id> <dep-id>    Add dependency (id depends on dep-id)
  dep remove <id> <dep-id> Remove dependency (alias: rm)
  dep tree [id]            Show dependency tree
    --full                 Show full tree for all tickets
  dep check                Check for dependency cycles
  undep <id> <dep-id>      Remove dependency (alias for dep remove)
  link <id> <id> [id...]   Link tickets together (symmetric)
  unlink <id> <target-id>  Remove link between tickets
  add-note <id> [text]     Append timestamped note (text or stdin)
  query [jq-filter]        Output tickets as JSON, optionally filtered with jq
  version                  Print version information
  update                   Update tk to the latest version

Use "tk [command] --help" for more information about a command.

Tickets stored as markdown files in .tickets/
Supports partial ID matching (e.g., 'tk show 5c4' matches 'nw-5c46')
`
	fmt.Print(helpText)
}

// GetConfig returns the loaded configuration.
func GetConfig() *config.Config {
	return cfg
}

// GetLogger returns the configured logger.
func GetLogger() *slog.Logger {
	return logger
}

// GetStorage returns the storage instance.
func GetStorage() *storage.Storage {
	return store
}

func init() {
	// Store the default help function before overriding
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == rootCmd {
			printHelp()
		} else {
			// Use default Cobra help for subcommands
			defaultHelp(cmd, args)
		}
	})
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(reopenCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(readyCmd)
	rootCmd.AddCommand(blockedCmd)
	rootCmd.AddCommand(closedCmd)
	rootCmd.AddCommand(depCmd)
	rootCmd.AddCommand(undepCmd)
	rootCmd.AddCommand(linkCmd)
	rootCmd.AddCommand(unlinkCmd)
	rootCmd.AddCommand(addNoteCmd)
	rootCmd.AddCommand(queryCmd)
}
