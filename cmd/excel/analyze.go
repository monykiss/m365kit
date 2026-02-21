package excel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/ai"
	"github.com/klytics/m365kit/internal/formats/xlsx"
)

const excelAnalyzePrompt = `You are an expert data analyst. Analyze the following spreadsheet data and provide structured insights.

For each sheet:
1. Describe the data structure (columns, types, row count)
2. Identify trends, patterns, and correlations
3. Flag anomalies or outliers
4. Provide key summary statistics where relevant
5. Suggest actionable insights

Present your analysis in clear sections. Be specific — reference actual values, column names, and row positions.`

func newAnalyzeCommand() *cobra.Command {
	var (
		sheet  string
		prompt string
	)

	cmd := &cobra.Command{
		Use:   "analyze <file.xlsx>",
		Short: "AI-powered analysis of Excel data",
		Long:  "Reads an Excel file and uses AI to identify trends, anomalies, and insights in the spreadsheet data.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			providerName, _ := cmd.Flags().GetString("provider")
			modelName, _ := cmd.Flags().GetString("model")

			filePath := args[0]
			if !strings.HasSuffix(strings.ToLower(filePath), ".xlsx") {
				return fmt.Errorf("expected a .xlsx file, got %q", filePath)
			}

			wb, err := xlsx.ReadFile(filePath)
			if err != nil {
				return err
			}

			// Build text representation of the spreadsheet
			var input strings.Builder
			input.WriteString(fmt.Sprintf("File: %s\n\n", filePath))

			if sheet != "" {
				s, err := wb.GetSheet(sheet)
				if err != nil {
					return err
				}
				writeSheetText(&input, s)
			} else {
				for i := range wb.Sheets {
					writeSheetText(&input, &wb.Sheets[i])
				}
			}

			systemPrompt := excelAnalyzePrompt
			if prompt != "" {
				systemPrompt += "\n\nAdditional instructions: " + prompt
			}

			provider, err := ai.NewProvider(providerName, modelName)
			if err != nil {
				return err
			}

			ctx := context.Background()
			messages := []ai.Message{
				{Role: "user", Content: input.String()},
			}

			if jsonFlag {
				result, err := provider.Infer(ctx, systemPrompt, messages, ai.InferOptions{})
				if err != nil {
					return fmt.Errorf("AI inference failed: %w", err)
				}

				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"analysis": result.Content,
					"model":    result.Model,
					"tokens":   result.InputTokens + result.OutputTokens,
					"file":     filePath,
				})
			}

			textCh, errCh, err := provider.Stream(ctx, systemPrompt, messages, ai.InferOptions{})
			if err != nil {
				return fmt.Errorf("AI inference failed: %w", err)
			}

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
		},
	}

	cmd.Flags().StringVar(&sheet, "sheet", "", "Analyze a specific sheet (default: all sheets)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Additional analysis instructions")

	return cmd
}

func writeSheetText(b *strings.Builder, s *xlsx.Sheet) {
	b.WriteString(fmt.Sprintf("=== Sheet: %s (%d rows) ===\n", s.Name, s.RowCount()))
	b.WriteString(s.ToCSV())
	b.WriteString("\n")
}
