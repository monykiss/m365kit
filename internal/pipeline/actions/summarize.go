package actions

import (
	"context"
	"fmt"

	"github.com/klytics/m365kit/internal/ai"
	"github.com/klytics/m365kit/internal/pipeline"
)

const defaultPipelineSummarizePrompt = "You are a precise document analyst. Summarize the following document concisely, capturing key points, decisions, dates, and action items."

// AISummarizeAction summarizes text content using AI.
func AISummarizeAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("ai.summarize requires input text")
	}

	provider, err := ai.NewProvider("anthropic", "")
	if err != nil {
		return "", err
	}

	prompt := defaultPipelineSummarizePrompt
	if p, ok := step.Options["prompt"]; ok {
		prompt = p
	}

	messages := []ai.Message{
		{Role: "user", Content: input},
	}

	result, inferErr := provider.Infer(ctx, prompt, messages, ai.InferOptions{})
	if inferErr != nil {
		return "", fmt.Errorf("AI summarize failed: %w", inferErr)
	}

	return result.Content, nil
}
