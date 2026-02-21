package fs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// RenameRule defines how files should be renamed.
type RenameRule struct {
	Pattern string // "kebab", "snake", "camel", "date-prefix", "lower"
	DryRun  bool
}

// RenameResult holds rename operation results.
type RenameResult struct {
	OldPath string `json:"oldPath"`
	NewPath string `json:"newPath"`
	Applied bool   `json:"applied"`
	Error   string `json:"error,omitempty"`
}

// Rename applies naming conventions to office documents.
func Rename(files []FileInfo, rule RenameRule) []RenameResult {
	var results []RenameResult

	for _, f := range files {
		dir := filepath.Dir(f.Path)
		ext := filepath.Ext(f.Name)
		base := strings.TrimSuffix(f.Name, ext)

		var newBase string
		switch rule.Pattern {
		case "kebab":
			newBase = toKebab(base)
		case "snake":
			newBase = toSnake(base)
		case "lower":
			newBase = strings.ToLower(base)
		case "date-prefix":
			date := f.ModifiedAt.Format("2006-01-02")
			cleanBase := toKebab(base)
			// Don't double-prefix if already date-prefixed
			if !isDatePrefixed(cleanBase) {
				newBase = date + "-" + cleanBase
			} else {
				newBase = cleanBase
			}
		default:
			newBase = base
		}

		newName := newBase + strings.ToLower(ext)
		newPath := filepath.Join(dir, newName)

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

		// Check target doesn't already exist
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

var (
	nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	multiDash   = regexp.MustCompile(`-{2,}`)
	datePrefix  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
)

func toKebab(s string) string {
	// Insert dash before uppercase sequences
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(rune(s[i-1])) {
			result = append(result, '-')
		}
		result = append(result, unicode.ToLower(r))
	}
	out := string(result)
	out = nonAlphaNum.ReplaceAllString(out, "-")
	out = multiDash.ReplaceAllString(out, "-")
	out = strings.Trim(out, "-")
	return out
}

func toSnake(s string) string {
	return strings.ReplaceAll(toKebab(s), "-", "_")
}

func isDatePrefixed(s string) bool {
	if !datePrefix.MatchString(s) {
		return false
	}
	// Validate it's actually a reasonable date
	dateStr := s[:10]
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}
