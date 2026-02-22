// Package update provides update checking and self-update functionality.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	githubRepo   = "monykiss/m365kit"
	checkTimeout = 2 * time.Second
	checkCooldown = 24 * time.Hour
)

// ReleaseInfo represents a GitHub release.
type ReleaseInfo struct {
	Version     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
}

// CheckLatest queries GitHub releases API for the latest version.
// Returns nil if current version is latest or newer.
func CheckLatest(currentVersion string) (*ReleaseInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // no releases yet
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("GitHub API rate limited — try again later")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("could not parse release info: %w", err)
	}

	if !isNewer(release.Version, currentVersion) {
		return nil, nil
	}

	return &release, nil
}

// isNewer returns true if latest is newer than current.
// Simple string comparison after stripping 'v' prefix.
func isNewer(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	if current == "dev" || current == "" {
		return false // dev builds don't get update notices
	}

	return latest != current && latest > current
}

// FormatUpdateNotice returns a formatted update message.
func FormatUpdateNotice(current string, release *ReleaseInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Current version: %s\n", current))
	sb.WriteString(fmt.Sprintf("Latest version:  %s  (released %s)\n", release.Version, release.PublishedAt.Format("2006-01-02")))
	sb.WriteString("\nUpdate available! ")

	// Extract first few lines of release notes
	body := strings.TrimSpace(release.Body)
	if body != "" {
		lines := strings.Split(body, "\n")
		sb.WriteString("What's new:\n")
		max := 5
		if len(lines) < max {
			max = len(lines)
		}
		for _, line := range lines[:max] {
			sb.WriteString("  " + line + "\n")
		}
	}

	sb.WriteString("\nTo update:\n")
	sb.WriteString("  brew upgrade monykiss/tap/m365kit  (Homebrew)\n")
	sb.WriteString("  go install github.com/monykiss/m365kit@latest  (Go)\n")
	sb.WriteString("  kit update install  (self-update)\n")

	return sb.String()
}

// CheckInBackground runs update check asynchronously.
// Only checks once per 24 hours.
func CheckInBackground(currentVersion string) {
	go func() {
		if !shouldCheck() {
			return
		}

		release, err := CheckLatest(currentVersion)
		if err != nil || release == nil {
			return
		}

		// Save last check time
		saveLastCheck()

		// Print non-blocking notice
		fmt.Fprintf(os.Stderr, "\nUpdate available: %s -> run: kit update install\n", release.Version)
	}()
}

func lastCheckPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kit", "last_update_check")
}

func shouldCheck() bool {
	data, err := os.ReadFile(lastCheckPath())
	if err != nil {
		return true
	}
	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return true
	}
	return time.Since(t) > checkCooldown
}

func saveLastCheck() {
	path := lastCheckPath()
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0600)
}
