package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/ai"
)

const defaultExtractPrompt = "You are a precise entity extractor. Extract the requested fields from the following document. Return the results as a JSON object with the field names as keys. If a field cannot be found, set its value to null. Be exact — do not infer or guess values that are not present in the text."

func newExtractCommand() *cobra.Command {
	var fields string

	cmd := &cobra.Command{
		Use:   "extract [file]",
		Short: "Extract structured entities from a document using AI",
		Long:  "Uses AI to extract specific fields (e.g., names, dates, amounts) from document text and returns structured JSON.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerName, _ := cmd.Flags().GetString("provider")
			modelName, _ := cmd.Flags().GetString("model")

			if fields == "" {
				return fmt.Errorf("--fields is required — specify comma-separated field names to extract\n\nExample: kit ai extract --fields \"name,date,amount\" contract.docx")
			}

			input, err := readInput(args)
			if err != nil {
				return err
			}

			systemPrompt := defaultExtractPrompt + fmt.Sprintf("\n\nFields to extract: %s\n\nReturn ONLY valid JSON, no other text.", fields)

			provider, err := ai.NewProvider(providerName, modelName)
			if err != nil {
				return err
			}

			ctx := context.Background()
			messages := []ai.Message{
				{Role: "user", Content: input},
			}

			result, err := provider.Infer(ctx, systemPrompt, messages, ai.InferOptions{})
			if err != nil {
				return fmt.Errorf("AI inference failed: %w", err)
			}

			// Try to pretty-print if the output is valid JSON
			var parsed interface{}
			if err := json.Unmarshal([]byte(result.Content), &parsed); err == nil {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(parsed)
			}

			// Fall back to raw output
			fmt.Println(result.Content)
			return nil
		},
	}

	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated field names to extract (required)")

	return cmd
}
