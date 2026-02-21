// Package diff provides the kit diff command for comparing documents.
package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/ai"
	"github.com/klytics/m365kit/internal/formats/docx"
)

const aiSummaryPrompt = "Summarize these document changes in plain English (under 200 words). Be specific: what was added, removed, or changed and where?"

// NewCommand returns the diff command.
func NewCommand() *cobra.Command {
	var (
		contextLines int
		stats        bool
		aiSummary    bool
	)

	cmd := &cobra.Command{
		Use:   "diff <original.docx> <revised.docx>",
		Short: "Compare two Word documents",
		Long: `Shows a colored unified diff of paragraph-level changes between two .docx files.

Examples:
  kit diff original.docx revised.docx
  kit diff original.docx revised.docx --stats
  kit diff original.docx revised.docx --ai-summary`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			providerName, _ := cmd.Flags().GetString("provider")
			modelName, _ := cmd.Flags().GetString("model")

			originalPath := args[0]
			revisedPath := args[1]

			for _, p := range []string{originalPath, revisedPath} {
				if !strings.HasSuffix(strings.ToLower(p), ".docx") {
					return fmt.Errorf("expected a .docx file, got %q", p)
				}
			}

			result, err := docx.DiffDocuments(originalPath, revisedPath, contextLines)
			if err != nil {
				return err
			}

			if jsonFlag {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			if stats {
				fmt.Println(result.Stats())
				return nil
			}

			// Colored output
			printColoredDiff(result)

			// AI summary if requested
			if aiSummary {
				fmt.Println()
				return streamAISummary(result, providerName, modelName)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&contextLines, "context", "C", 3, "Number of context lines around each change")
	cmd.Flags().BoolVar(&stats, "stats", false, "Show only insertion/deletion counts")
	cmd.Flags().BoolVar(&aiSummary, "ai-summary", false, "AI plain-English summary of changes")

	return cmd
}

func printColoredDiff(result *docx.DiffResult) {
	dim := color.New(color.FgHiBlack)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	cyan := color.New(color.FgCyan)

	origCount := result.Unchanged + result.Deletions
	revCount := result.Unchanged + result.Insertions

	red.Printf("--- %s  (%d paragraphs)\n", result.Original, origCount)
	green.Printf("+++ %s  (%d paragraphs)\n", result.Revised, revCount)

	for _, hunk := range result.Hunks {
		fmt.Println()
		cyan.Println(hunk.Header)
		for _, line := range hunk.Lines {
			switch line.Type {
			case "context":
				dim.Printf("  %s\n", line.Content)
			case "delete":
				red.Printf("- %s\n", line.Content)
			case "insert":
				green.Printf("+ %s\n", line.Content)
			}
		}
	}

	fmt.Printf("\n%s\n", result.Stats())
}

func streamAISummary(result *docx.DiffResult, providerName, modelName string) error {
	changeSummary := result.ChangeSummary()
	if result.Insertions == 0 && result.Deletions == 0 {
		fmt.Println("No changes to summarize.")
		return nil
	}

	provider, err := ai.NewProvider(providerName, modelName)
	if err != nil {
		return fmt.Errorf("AI summary failed: %w", err)
	}

	ctx := context.Background()
	textCh, errCh, err := provider.Stream(ctx, aiSummaryPrompt, []ai.Message{
		{Role: "user", Content: changeSummary},
	}, ai.InferOptions{MaxTokens: 512})
	if err != nil {
		return fmt.Errorf("AI summary failed: %w", err)
	}

	bold := color.New(color.Bold)
	bold.Println("AI Summary:")
	for text := range textCh {
		fmt.Print(text)
	}
	fmt.Println()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("streaming error: %w", err)
		}
	default:
	}
	return nil
}
