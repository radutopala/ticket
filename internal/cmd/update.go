package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update tk to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return doUpdate(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func doUpdate(ctx context.Context) error {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{
		APIToken: os.Getenv("GITHUB_TOKEN"),
	})
	if err != nil {
		return fmt.Errorf("failed to create GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug("radutopala", "ticket"))
	if err != nil {
		return fmt.Errorf("failed to detect latest version: %w", err)
	}
	if !found {
		return fmt.Errorf("no release found")
	}

	currentVersion := version
	if currentVersion == "dev" {
		currentVersion = "0.0.0"
	}

	if latest.LessOrEqual(currentVersion) {
		fmt.Printf("Current version %s is up to date\n", version)
		return nil
	}

	fmt.Printf("Updating from %s to %s...\n", version, latest.Version())

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Successfully updated to %s\n", latest.Version())
	fmt.Printf("  OS:   %s\n", runtime.GOOS)
	fmt.Printf("  Arch: %s\n", runtime.GOARCH)

	return nil
}
