package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/ai"
)

const defaultSummarizePrompt = "You are a precise document analyst. Summarize the following document concisely, capturing key points, decisions, dates, and action items. Structure your summary with clear sections. Be factual and avoid speculation."

type summarizeOutput struct {
	Summary   string   `json:"summary"`
	KeyPoints []string `json:"keyPoints,omitempty"`
	Model     string   `json:"model"`
	Tokens    int      `json:"tokens"`
}

func newSummarizeCommand() *cobra.Command {
	var focus string

	cmd := &cobra.Command{
		Use:   "summarize [file]",
		Short: "Generate an AI summary of a document or piped text",
		Long:  "Reads a file or piped stdin and produces a concise AI-generated summary. Works with plain text, JSON, or Markdown input.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			providerName, _ := cmd.Flags().GetString("provider")
			modelName, _ := cmd.Flags().GetString("model")

			// Read input
			var input string
			if len(args) > 0 && args[0] != "-" {
				data, err := os.ReadFile(args[0])
				if err != nil {
					return fmt.Errorf("could not read file %s: %w", args[0], err)
				}
				input = string(data)
			} else {
				// Check if stdin has data
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return fmt.Errorf("no input provided — pass a file path or pipe content to stdin\n\nExample: kit word read contract.docx | kit ai summarize")
				}
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("could not read from stdin: %w", err)
				}
				input = string(data)
			}

			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("input is empty — provide document content to summarize")
			}

			// Auto-detect if input is JSON and extract text
			input = extractTextFromInput(input)

			// Build system prompt
			systemPrompt := defaultSummarizePrompt
			if focus != "" {
				systemPrompt += fmt.Sprintf("\n\nFocus your summary on these areas: %s", focus)
			}

			// Create provider
			provider, err := ai.NewProvider(providerName, modelName)
			if err != nil {
				return err
			}

			ctx := context.Background()
			messages := []ai.Message{
				{Role: "user", Content: input},
			}

			if jsonFlag {
				// Non-streaming for JSON output
				result, err := provider.Infer(ctx, systemPrompt, messages, ai.InferOptions{})
				if err != nil {
					return fmt.Errorf("AI inference failed: %w", err)
				}

				out := summarizeOutput{
					Summary: result.Content,
					Model:   result.Model,
					Tokens:  result.InputTokens + result.OutputTokens,
				}

				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			// Streaming output
			textCh, errCh, err := provider.Stream(ctx, systemPrompt, messages, ai.InferOptions{})
			if err != nil {
				return fmt.Errorf("AI inference failed: %w", err)
			}

			for text := range textCh {
				fmt.Print(text)
			}
			fmt.Println()

			// Check for streaming errors
			select {
			case err := <-errCh:
				if err != nil {
					return fmt.Errorf("streaming error: %w", err)
				}
			default:
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&focus, "focus", "", "Comma-separated focus areas (e.g., \"risks,dates,parties\")")

	return cmd
}

// extractTextFromInput attempts to detect if input is JSON from kit word read --json
// and extracts the text content from it.
func extractTextFromInput(input string) string {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "{") {
		return input
	}

	var parsed struct {
		Paragraphs []string `json:"paragraphs"`
	}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return input
	}

	if len(parsed.Paragraphs) > 0 {
		return strings.Join(parsed.Paragraphs, "\n\n")
	}

	return input
}
