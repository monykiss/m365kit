package docx

import (
	"fmt"
	"strings"
)

// DiffResult holds the result of comparing two documents.
type DiffResult struct {
	Original   string `json:"original"`
	Revised    string `json:"revised"`
	Insertions int    `json:"insertions"`
	Deletions  int    `json:"deletions"`
	Unchanged  int    `json:"unchanged"`
	Hunks      []Hunk `json:"hunks"`
}

// Hunk represents a contiguous group of changes.
type Hunk struct {
	Header string     `json:"header"`
	Lines  []DiffLine `json:"lines"`
}

// DiffLine represents a single line in a diff hunk.
type DiffLine struct {
	Type    string `json:"type"` // "insert", "delete", "context"
	Content string `json:"content"`
	OldLine int    `json:"oldLine,omitempty"`
	NewLine int    `json:"newLine,omitempty"`
}

// DiffDocuments parses both files and returns a paragraph-level diff.
func DiffDocuments(originalPath, revisedPath string, contextLines int) (*DiffResult, error) {
	origDoc, err := ParseFile(originalPath)
	if err != nil {
		return nil, fmt.Errorf("could not read original: %w", err)
	}

	revDoc, err := ParseFile(revisedPath)
	if err != nil {
		return nil, fmt.Errorf("could not read revised: %w", err)
	}

	return DiffParagraphs(origDoc.Paragraphs(), revDoc.Paragraphs(), originalPath, revisedPath, contextLines), nil
}

// DiffParagraphs computes a diff between two paragraph slices.
func DiffParagraphs(origParas, revParas []string, origName, revName string, contextLines int) *DiffResult {
	if contextLines < 0 {
		contextLines = 3
	}

	ops := myersDiff(origParas, revParas)

	result := &DiffResult{
		Original: origName,
		Revised:  revName,
	}

	// Count operations
	for _, op := range ops {
		switch op.Op {
		case "=":
			result.Unchanged++
		case "+":
			result.Insertions++
		case "-":
			result.Deletions++
		}
	}

	// Build hunks with context
	result.Hunks = buildHunks(ops, contextLines)

	return result
}

type editOp struct {
	Op   string // "=", "+", "-"
	Text string
}

// myersDiff computes the shortest edit script between a and b
// using an LCS-based approach (optimal for paragraph-level diffs).
func myersDiff(a, b []string) []editOp {
	n := len(a)
	m := len(b)

	if n == 0 && m == 0 {
		return nil
	}

	// Build LCS table
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to produce edit ops
	var ops []editOp
	i, j := n, m
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			ops = append(ops, editOp{Op: "=", Text: a[i-1]})
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			ops = append(ops, editOp{Op: "-", Text: a[i-1]})
			i--
		} else {
			ops = append(ops, editOp{Op: "+", Text: b[j-1]})
			j--
		}
	}
	for i > 0 {
		ops = append(ops, editOp{Op: "-", Text: a[i-1]})
		i--
	}
	for j > 0 {
		ops = append(ops, editOp{Op: "+", Text: b[j-1]})
		j--
	}

	// Reverse since we built backwards
	for i, j := 0, len(ops)-1; i < j; i, j = i+1, j-1 {
		ops[i], ops[j] = ops[j], ops[i]
	}

	return ops
}

// buildHunks groups edit operations into hunks with context lines.
func buildHunks(ops []editOp, contextLines int) []Hunk {
	if len(ops) == 0 {
		return nil
	}

	// Find change positions
	type changeRange struct {
		start, end int
	}
	var changes []changeRange
	for i, op := range ops {
		if op.Op != "=" {
			if len(changes) > 0 && i-changes[len(changes)-1].end <= 2*contextLines {
				// Merge with previous change
				changes[len(changes)-1].end = i + 1
			} else {
				changes = append(changes, changeRange{start: i, end: i + 1})
			}
		}
	}

	if len(changes) == 0 {
		return nil
	}

	var hunks []Hunk
	for _, cr := range changes {
		// Expand with context
		start := cr.start - contextLines
		if start < 0 {
			start = 0
		}
		end := cr.end + contextLines
		if end > len(ops) {
			end = len(ops)
		}

		// Calculate line numbers
		oldStart := 1
		newStart := 1
		for i := 0; i < start; i++ {
			switch ops[i].Op {
			case "=":
				oldStart++
				newStart++
			case "-":
				oldStart++
			case "+":
				newStart++
			}
		}

		oldCount := 0
		newCount := 0
		var lines []DiffLine
		oldLine := oldStart
		newLine := newStart

		for i := start; i < end; i++ {
			op := ops[i]
			switch op.Op {
			case "=":
				lines = append(lines, DiffLine{
					Type:    "context",
					Content: op.Text,
					OldLine: oldLine,
					NewLine: newLine,
				})
				oldLine++
				newLine++
				oldCount++
				newCount++
			case "-":
				lines = append(lines, DiffLine{
					Type:    "delete",
					Content: op.Text,
					OldLine: oldLine,
				})
				oldLine++
				oldCount++
			case "+":
				lines = append(lines, DiffLine{
					Type:    "insert",
					Content: op.Text,
					NewLine: newLine,
				})
				newLine++
				newCount++
			}
		}

		header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount)
		hunks = append(hunks, Hunk{
			Header: header,
			Lines:  lines,
		})
	}

	return hunks
}

// FormatUnified returns the diff as a unified diff string (with ANSI colors if enabled).
func (d *DiffResult) FormatUnified(useColor bool) string {
	var b strings.Builder

	origCount := d.Unchanged + d.Deletions
	revCount := d.Unchanged + d.Insertions

	b.WriteString(fmt.Sprintf("--- %s  (%d paragraphs)\n", d.Original, origCount))
	b.WriteString(fmt.Sprintf("+++ %s  (%d paragraphs)\n", d.Revised, revCount))

	for _, hunk := range d.Hunks {
		b.WriteString("\n")
		b.WriteString(hunk.Header)
		b.WriteString("\n")
		for _, line := range hunk.Lines {
			switch line.Type {
			case "context":
				b.WriteString("  " + line.Content + "\n")
			case "delete":
				b.WriteString("- " + line.Content + "\n")
			case "insert":
				b.WriteString("+ " + line.Content + "\n")
			}
		}
	}

	b.WriteString(fmt.Sprintf("\n%d insertions, %d deletions, %d unchanged\n", d.Insertions, d.Deletions, d.Unchanged))
	return b.String()
}

// Stats returns a single-line summary.
func (d *DiffResult) Stats() string {
	return fmt.Sprintf("%d insertions, %d deletions, %d unchanged", d.Insertions, d.Deletions, d.Unchanged)
}

// ChangeSummary returns a compact text of all changes for AI consumption.
func (d *DiffResult) ChangeSummary() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Comparing %s vs %s:\n", d.Original, d.Revised))
	b.WriteString(fmt.Sprintf("Stats: %s\n\n", d.Stats()))
	b.WriteString("Changes:\n")
	for _, hunk := range d.Hunks {
		b.WriteString(hunk.Header + "\n")
		for _, line := range hunk.Lines {
			switch line.Type {
			case "delete":
				b.WriteString("REMOVED: " + line.Content + "\n")
			case "insert":
				b.WriteString("ADDED: " + line.Content + "\n")
			}
		}
	}
	return b.String()
}
