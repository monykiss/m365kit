package fs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// OrganizeRule defines how files should be organized into folders.
type OrganizeRule struct {
	Strategy string // "by-type", "by-year", "by-month"
	DryRun   bool
}

// OrganizeFile organizes files into subdirectories based on the strategy.
func OrganizeFile(files []FileInfo, rootDir string, rule OrganizeRule) []RenameResult {
	var results []RenameResult

	for _, f := range files {
		var subDir string
		switch rule.Strategy {
		case "by-type":
			subDir = f.Format
		case "by-year":
			subDir = f.ModifiedAt.Format("2006")
		case "by-month":
			subDir = filepath.Join(f.ModifiedAt.Format("2006"), f.ModifiedAt.Format("01-January"))
		default:
			subDir = f.Format
		}

		targetDir := filepath.Join(rootDir, subDir)
		newPath := filepath.Join(targetDir, f.Name)

		result := RenameResult{
			OldPath: f.Path,
			NewPath: newPath,
		}

		if f.Path == newPath {
			result.Applied = false
			results = append(results, result)
			continue
		}

		if rule.DryRun {
			result.Applied = false
			results = append(results, result)
			continue
		}

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			result.Error = fmt.Sprintf("could not create dir: %v", err)
			results = append(results, result)
			continue
		}

		// Check target doesn't exist
		if _, err := os.Stat(newPath); err == nil {
			result.Error = "target already exists"
			results = append(results, result)
			continue
		}

		if err := os.Rename(f.Path, newPath); err != nil {
			result.Error = err.Error()
		} else {
			result.Applied = true
		}
		results = append(results, result)
	}

	return results
}

// StaleFiles returns files not modified since the given duration.
func StaleFiles(files []FileInfo, since time.Duration) []FileInfo {
	cutoff := time.Now().Add(-since)
	var stale []FileInfo
	for _, f := range files {
		if f.ModifiedAt.Before(cutoff) {
			stale = append(stale, f)
		}
	}
	// Sort oldest first
	sort.Slice(stale, func(i, j int) bool {
		return stale[i].ModifiedAt.Before(stale[j].ModifiedAt)
	})
	return stale
}

// Manifest generates a JSON manifest of all scanned files.
func Manifest(result *ScanResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

// FormatSize returns a human-readable file size string.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
