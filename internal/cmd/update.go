package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
)

const (
	repoOwner = "radutopala"
	repoName  = "ticket"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update tk to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return doUpdate()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func doUpdate() error {
	// Get latest version by following redirect
	latestVersion, err := getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	currentVersion := version
	if currentVersion == "dev" {
		currentVersion = "0.0.0"
	}

	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to parse current version: %w", err)
	}

	latest, err := semver.NewVersion(latestVersion)
	if err != nil {
		return fmt.Errorf("failed to parse latest version: %w", err)
	}

	if !latest.GreaterThan(current) {
		fmt.Printf("Current version %s is up to date\n", version)
		return nil
	}

	fmt.Printf("Updating from %s to %s...\n", version, latestVersion)

	// Get current executable path
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Download and extract new binary
	if err := downloadAndReplace(latestVersion, exe); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Successfully updated to %s\n", latestVersion)
	fmt.Printf("  OS:   %s\n", runtime.GOOS)
	fmt.Printf("  Arch: %s\n", runtime.GOARCH)

	return nil
}

func getLatestVersion() (string, error) {
	// GitHub redirects /releases/latest to /releases/tag/vX.Y.Z
	url := fmt.Sprintf("https://github.com/%s/%s/releases/latest", repoOwner, repoName)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusMovedPermanently {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	// Location is like: https://github.com/radutopala/ticket/releases/tag/v0.2.1
	parts := strings.Split(location, "/tag/")
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected redirect location: %s", location)
	}

	return strings.TrimPrefix(parts[1], "v"), nil
}

func downloadAndReplace(version, exePath string) error {
	// Construct download URL
	arch := runtime.GOARCH
	if arch == "amd64" {
		// Keep as-is
	} else if arch == "arm64" {
		// Keep as-is
	}

	var ext string
	if runtime.GOOS == "windows" {
		ext = "zip"
	} else {
		ext = "tar.gz"
	}

	url := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/tk_%s_%s_%s.%s",
		repoOwner, repoName, version, version, runtime.GOOS, arch, ext,
	)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Create temp file for new binary
	tmpFile, err := os.CreateTemp("", "tk-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Extract binary from tarball
	if ext == "tar.gz" {
		if err := extractTarGz(resp.Body, tmpFile); err != nil {
			tmpFile.Close()
			return err
		}
	} else {
		if err := extractZip(resp.Body, tmpFile); err != nil {
			tmpFile.Close()
			return err
		}
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}

	// Replace old binary
	oldPath := exePath + ".old"
	if err := os.Rename(exePath, oldPath); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		// Try to restore old binary
		os.Rename(oldPath, exePath)
		return err
	}

	// Remove old binary
	os.Remove(oldPath)

	return nil
}

func extractTarGz(r io.Reader, w io.Writer) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for the tk binary
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "tk" {
			_, err := io.Copy(w, tr)
			return err
		}
	}

	return fmt.Errorf("tk binary not found in archive")
}

func extractZip(r io.Reader, w io.Writer) error {
	// Read all content into memory since zip requires seeking
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		// Look for the tk.exe binary
		if filepath.Base(f.Name) == "tk.exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			_, err = io.Copy(w, rc)
			return err
		}
	}

	return fmt.Errorf("tk.exe binary not found in archive")
}
