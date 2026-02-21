package actions

import (
	"context"
	"fmt"

	"github.com/klytics/m365kit/internal/ai"
	"github.com/klytics/m365kit/internal/pipeline"
)

const defaultPipelineExtractPrompt = "You are a precise entity extractor. Extract the requested fields from the following document. Return the results as a JSON object."

// AIAnalyzeAction performs structured analysis using AI.
func AIAnalyzeAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("ai.analyze requires input text")
	}

	provider, err := ai.NewProvider("anthropic", "")
	if err != nil {
		return "", err
	}

	prompt := "You are a data analyst. Analyze the following data and provide structured insights."
	if p, ok := step.Options["prompt"]; ok {
		prompt = p
	}

	messages := []ai.Message{
		{Role: "user", Content: input},
	}

	result, inferErr := provider.Infer(ctx, prompt, messages, ai.InferOptions{})
	if inferErr != nil {
		return "", fmt.Errorf("AI analyze failed: %w", inferErr)
	}

	return result.Content, nil
}

// AIExtractAction extracts structured entities using AI.
func AIExtractAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("ai.extract requires input text")
	}

	provider, err := ai.NewProvider("anthropic", "")
	if err != nil {
		return "", err
	}

	prompt := defaultPipelineExtractPrompt
	if fields, ok := step.Options["fields"]; ok {
		prompt += fmt.Sprintf("\n\nFields to extract: %s", fields)
	}

	messages := []ai.Message{
		{Role: "user", Content: input},
	}

	result, inferErr := provider.Infer(ctx, prompt, messages, ai.InferOptions{})
	if inferErr != nil {
		return "", fmt.Errorf("AI extract failed: %w", inferErr)
	}

	return result.Content, nil
}
