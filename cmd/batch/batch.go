// Package batch provides CLI commands for batch processing files.
package batch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/formats/docx"
	"github.com/klytics/m365kit/internal/formats/xlsx"
)

type batchResultItem struct {
	File   string      `json:"file"`
	Status string      `json:"status"`
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// NewCommand returns the batch subcommand.
func NewCommand() *cobra.Command {
	var (
		action      string
		findStr     string
		replaceStr  string
		outDir      string
		concurrency int
	)

	cmd := &cobra.Command{
		Use:   "batch <glob-pattern>",
		Short: "Batch process multiple files with a single action",
		Long: `Applies an action to all files matching a glob pattern.

Actions: read, edit, summarize
On error, the batch logs the failure and continues to the next file.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")

			if action == "" {
				return fmt.Errorf("--action is required — specify an action to perform\n\nExample: kit batch '*.docx' --action read")
			}

			pattern := args[0]
			files, err := filepath.Glob(pattern)
			if err != nil {
				return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
			}

			if len(files) == 0 {
				return fmt.Errorf("no files matched pattern %q", pattern)
			}

			// Create output directory if specified
			if outDir != "" {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					return fmt.Errorf("could not create output directory %s: %w", outDir, err)
				}
			}

			results := make([]batchResultItem, len(files))
			succeeded := 0
			failed := 0

			if concurrency <= 1 {
				// Sequential processing
				for i, file := range files {
					if !jsonFlag {
						fmt.Printf("[%d/%d] Processing %s...\n", i+1, len(files), filepath.Base(file))
					}
					result := processFile(file, action, findStr, replaceStr, outDir, jsonFlag)
					results[i] = result
					if result.Status == "ok" {
						succeeded++
					} else {
						failed++
					}
				}
			} else {
				// Concurrent processing
				var mu sync.Mutex
				sem := make(chan struct{}, concurrency)
				var wg sync.WaitGroup

				for i, file := range files {
					wg.Add(1)
					go func(idx int, f string) {
						defer wg.Done()
						sem <- struct{}{}
						defer func() { <-sem }()

						if !jsonFlag {
							mu.Lock()
							fmt.Printf("[%d/%d] Processing %s...\n", idx+1, len(files), filepath.Base(f))
							mu.Unlock()
						}

						result := processFile(f, action, findStr, replaceStr, outDir, jsonFlag)
						mu.Lock()
						results[idx] = result
						if result.Status == "ok" {
							succeeded++
						} else {
							failed++
						}
						mu.Unlock()
					}(i, file)
				}
				wg.Wait()
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			fmt.Printf("\nProcessed %d files. %d succeeded, %d failed.\n", len(files), succeeded, failed)
			return nil
		},
	}

	cmd.Flags().StringVar(&action, "action", "", "Action to perform: read | edit | summarize")
	cmd.Flags().StringVar(&findStr, "find", "", "Text to find (for edit action)")
	cmd.Flags().StringVar(&replaceStr, "replace", "", "Replacement text (for edit action)")
	cmd.Flags().StringVar(&outDir, "out-dir", "", "Output directory for results")
	cmd.Flags().IntVar(&concurrency, "concurrency", 1, "Number of parallel workers")

	return cmd
}

func processFile(file, action, findStr, replaceStr, outDir string, jsonFlag bool) batchResultItem {
	result := batchResultItem{File: file, Status: "ok"}

	switch action {
	case "read":
		output, err := readFile(file)
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
			return result
		}
		if jsonFlag {
			result.Output = output
		} else {
			fmt.Println(output)
		}

	case "edit":
		if findStr == "" {
			result.Status = "error"
			result.Error = "--find is required for edit action"
			return result
		}
		count, outPath, err := editFile(file, findStr, replaceStr, outDir)
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
			return result
		}
		result.Output = map[string]interface{}{
			"replacements": count,
			"output":       outPath,
		}
		if !jsonFlag {
			fmt.Printf("  %d replacement(s) → %s\n", count, outPath)
		}

	case "summarize":
		// Read the file content for summarize (actual AI call not made without API key)
		output, err := readFile(file)
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
			return result
		}
		result.Output = output

	default:
		result.Status = "error"
		result.Error = fmt.Sprintf("unknown action %q — supported: read, edit, summarize", action)
	}

	return result
}

func readFile(file string) (interface{}, error) {
	ext := strings.ToLower(filepath.Ext(file))
	switch ext {
	case ".docx":
		doc, err := docx.ParseFile(file)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"file":       file,
			"paragraphs": doc.Paragraphs(),
			"wordCount":  doc.WordCount(),
		}, nil

	case ".xlsx":
		wb, err := xlsx.ReadFile(file)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"file":   file,
			"sheets": wb.Sheets,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported file type %q — supported: .docx, .xlsx", ext)
	}
}

func editFile(file, find, replace, outDir string) (int, string, error) {
	ext := strings.ToLower(filepath.Ext(file))
	if ext != ".docx" {
		return 0, "", fmt.Errorf("edit is only supported for .docx files, got %q", ext)
	}

	// Determine output path
	base := filepath.Base(file)
	nameNoExt := strings.TrimSuffix(base, filepath.Ext(base))
	outPath := nameNoExt + ".edited.docx"
	if outDir != "" {
		outPath = filepath.Join(outDir, outPath)
	}

	replacements := map[string]string{find: replace}
	result, err := docx.EditFile(file, replacements, outPath)
	if err != nil {
		return 0, "", err
	}

	return result.ReplacementsMade, result.OutputPath, nil
}
