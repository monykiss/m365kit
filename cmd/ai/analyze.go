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

const defaultAnalyzePrompt = "You are a data analyst. Analyze the following data and provide structured insights. Identify trends, anomalies, and key findings. Present your analysis in clear sections with supporting evidence from the data."

func newAnalyzeCommand() *cobra.Command {
	var prompt string

	cmd := &cobra.Command{
		Use:   "analyze [file]",
		Short: "Perform AI-powered structured analysis on data or documents",
		Long:  "Analyzes input data and provides structured insights including trends, anomalies, and key findings.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			providerName, _ := cmd.Flags().GetString("provider")
			modelName, _ := cmd.Flags().GetString("model")

			input, err := readInput(args)
			if err != nil {
				return err
			}

			systemPrompt := defaultAnalyzePrompt
			if prompt != "" {
				systemPrompt += "\n\nAdditional instructions: " + prompt
			}

			provider, err := ai.NewProvider(providerName, modelName)
			if err != nil {
				return err
			}

			ctx := context.Background()
			messages := []ai.Message{
				{Role: "user", Content: input},
			}

			if jsonFlag {
				result, err := provider.Infer(ctx, systemPrompt, messages, ai.InferOptions{})
				if err != nil {
					return fmt.Errorf("AI inference failed: %w", err)
				}

				out := map[string]interface{}{
					"analysis": result.Content,
					"model":    result.Model,
					"tokens":   result.InputTokens + result.OutputTokens,
				}

				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
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

	cmd.Flags().StringVar(&prompt, "prompt", "", "Additional analysis instructions")

	return cmd
}

func readInput(args []string) (string, error) {
	if len(args) > 0 && args[0] != "-" {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return "", fmt.Errorf("could not read file %s: %w", args[0], err)
		}
		return string(data), nil
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no input provided — pass a file path or pipe content to stdin")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("could not read from stdin: %w", err)
	}

	input := string(data)
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("input is empty")
	}

	return input, nil
}
