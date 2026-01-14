package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type ReleaseInfo struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	CreatedAt   string `json:"created_at"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	Assets      []struct {
		Name        string `json:"name"`
		Size        int64  `json:"size"`
		DownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

var (
	updateCheckOnly  bool
	updatePrerelease bool
)

var version = "dev"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update openjudges to the latest version",
	Long: `Update openjudges to the latest version from GitHub releases.
This command will check for available updates and optionally download and install them.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check-only", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVar(&updatePrerelease, "prerelease", false, "Include pre-release versions")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Checking for updates...")

	// Get current version
	currentVersion := getCurrentVersion()
	fmt.Printf("Current version: %s\n", currentVersion)

	// Fetch latest release
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	latest, err := fetchLatestRelease(ctx, updatePrerelease)
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}

	fmt.Printf("Latest version: %s\n", latest.TagName)

	// Check if update needed
	if strings.TrimPrefix(currentVersion, "v") == strings.TrimPrefix(latest.TagName, "v") {
		fmt.Println("✓ Already up to date!")
		return nil
	}

	if updateCheckOnly {
		fmt.Printf("\n⬆️  Update available: %s → %s\n", currentVersion, latest.TagName)
		return nil
	}

	// Confirm update
	fmt.Printf("\nUpdate from %s to %s? [y/N] ", currentVersion, latest.TagName)
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Update cancelled")
		return nil
	}

	// Download and install
	fmt.Println("⬇️  Downloading update...")
	if err := downloadAndInstall(ctx, latest); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	fmt.Printf("✅ Successfully updated to %s\n", latest.TagName)
	return nil
}

func getCurrentVersion() string {
	// Version should be set at build time with -ldflags
	if version != "" {
		return version
	}
	return "dev"
}

func fetchLatestRelease(ctx context.Context, includePrerelease bool) (*ReleaseInfo, error) {
	url := "https://api.github.com/repos/Asyasyarif/openjudges/releases/latest"

	if includePrerelease {
		// For prerelease, get all releases and find the latest
		url = "https://api.github.com/repos/Asyasyarif/openjudges/releases"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "openjudges")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	if includePrerelease {
		var releases []ReleaseInfo
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, err
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("no releases found")
		}
		// Find the first non-draft release
		for _, release := range releases {
			if !release.Draft {
				return &release, nil
			}
		}
		return &releases[0], nil
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func downloadAndInstall(ctx context.Context, release *ReleaseInfo) error {
	// Determine platform
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Convert arch naming convention
	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
		"arm":   "armv7l",
	}

	if mapped, ok := archMap[arch]; ok {
		arch = mapped
	}

	// Determine binary name
	binaryName := fmt.Sprintf("openjudges_%s_%s", osName, arch)

	// Find matching asset
	var assetURL string
	for _, asset := range release.Assets {
		// Match exact platform string or with variations
		if strings.HasPrefix(asset.Name, binaryName) {
			assetURL = asset.DownloadURL
			break
		}
	}

	if assetURL == "" {
		return fmt.Errorf("no binary found for %s-%s\nAvailable assets: %v", osName, arch, getAssetNames(release.Assets))
	}

	// Download
	req, err := http.NewRequestWithContext(ctx, "GET", assetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "openjudges")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create temp file for new binary
	tmpPath := execPath + ".new"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Copy downloaded content
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// Make executable
	if err := tmpFile.Chmod(0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	tmpFile.Close()

	backupPath := execPath + ".backup"
	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(execPath)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	os.Remove(backupPath)

	return nil
}

func getAssetNames(assets []struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}) []string {
	names := make([]string, len(assets))
	for i, asset := range assets {
		names[i] = asset.Name
	}
	return names
}
