package fs

import (
	"fmt"
	"os"
)

// DuplicateGroup represents a set of files with the same content hash.
type DuplicateGroup struct {
	SHA256    string     `json:"sha256"`
	Size      int64      `json:"size"`
	Files     []FileInfo `json:"files"`
	WastedMB  float64    `json:"wastedMB"`
}

// DedupeResult holds deduplication analysis results.
type DedupeResult struct {
	Groups      []DuplicateGroup `json:"groups"`
	TotalDupes  int              `json:"totalDuplicates"`
	WastedBytes int64            `json:"wastedBytes"`
}

// FindDuplicates identifies files with identical content by SHA-256 hash.
// Files must have been scanned with WithHash=true.
func FindDuplicates(files []FileInfo) *DedupeResult {
	hashGroups := make(map[string][]FileInfo)
	for _, f := range files {
		if f.SHA256 == "" {
			continue
		}
		hashGroups[f.SHA256] = append(hashGroups[f.SHA256], f)
	}

	result := &DedupeResult{}
	for hash, group := range hashGroups {
		if len(group) < 2 {
			continue
		}
		wasted := int64(len(group)-1) * group[0].Size
		result.Groups = append(result.Groups, DuplicateGroup{
			SHA256:   hash,
			Size:     group[0].Size,
			Files:    group,
			WastedMB: float64(wasted) / (1024 * 1024),
		})
		result.TotalDupes += len(group) - 1
		result.WastedBytes += wasted
	}

	return result
}

// RemoveDuplicates deletes duplicate files, keeping the first (oldest by path) in each group.
func RemoveDuplicates(groups []DuplicateGroup, dryRun bool) []RenameResult {
	var results []RenameResult

	for _, g := range groups {
		// Keep first file, remove the rest
		for i := 1; i < len(g.Files); i++ {
			result := RenameResult{
				OldPath: g.Files[i].Path,
				NewPath: "(deleted)",
			}

			if dryRun {
				result.Applied = false
				results = append(results, result)
				continue
			}

			if err := os.Remove(g.Files[i].Path); err != nil {
				result.Error = err.Error()
			} else {
				result.Applied = true
			}
			results = append(results, result)
		}
	}

	return results
}

// FormatDedupeReport returns a human-readable duplicate report.
func FormatDedupeReport(result *DedupeResult) string {
	if len(result.Groups) == 0 {
		return "No duplicates found"
	}

	s := fmt.Sprintf("Found %d duplicate groups (%d files, %.1f MB wasted):\n\n",
		len(result.Groups), result.TotalDupes,
		float64(result.WastedBytes)/(1024*1024))

	for i, g := range result.Groups {
		s += fmt.Sprintf("Group %d (%s, %d copies):\n", i+1, FormatSize(g.Size), len(g.Files))
		for j, f := range g.Files {
			marker := "  "
			if j == 0 {
				marker = "* " // keep
			}
			s += fmt.Sprintf("  %s%s\n", marker, f.Path)
		}
		s += "\n"
	}

	return s
}
