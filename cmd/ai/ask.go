package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/klytics/m365kit/internal/ai"
)

const defaultAskPrompt = "You are a document question-answering assistant. Answer the user's question based on the document content provided. Only use information from the document. If the answer is not in the document, say so. Be concise and cite relevant sections when possible."

func newAskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask <question> [file]",
		Short: "Ask a question about a document using AI",
		Long:  "Sends a natural language question along with document content to an AI model and returns the answer.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			providerName, _ := cmd.Flags().GetString("provider")
			modelName, _ := cmd.Flags().GetString("model")

			question := args[0]
			fileArgs := args[1:]

			input, err := readInput(fileArgs)
			if err != nil {
				return err
			}

			userMessage := fmt.Sprintf("Document:\n%s\n\nQuestion: %s", input, question)

			provider, err := ai.NewProvider(providerName, modelName)
			if err != nil {
				return err
			}

			ctx := context.Background()
			messages := []ai.Message{
				{Role: "user", Content: userMessage},
			}

			if jsonFlag {
				result, err := provider.Infer(ctx, defaultAskPrompt, messages, ai.InferOptions{})
				if err != nil {
					return fmt.Errorf("AI inference failed: %w", err)
				}

				out := map[string]interface{}{
					"question": question,
					"answer":   result.Content,
					"model":    result.Model,
					"tokens":   result.InputTokens + result.OutputTokens,
				}

				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			textCh, errCh, err := provider.Stream(ctx, defaultAskPrompt, messages, ai.InferOptions{})
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

	return cmd
}
