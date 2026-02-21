// Package ai provides a unified interface to multiple AI inference providers.
package ai

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// Message represents a single message in a conversation with an AI model.
type Message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// InferOptions configures a single inference call.
type InferOptions struct {
	Model       string  `json:"model,omitempty"`
	MaxTokens   int     `json:"maxTokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"topP,omitempty"`
}

// InferResult holds the response from an inference call.
type InferResult struct {
	Content     string `json:"content"`
	Model       string `json:"model"`
	InputTokens int    `json:"inputTokens,omitempty"`
	OutputTokens int   `json:"outputTokens,omitempty"`
}

// Provider defines the interface that all AI backends must implement.
type Provider interface {
	// Infer sends a prompt and returns the complete response.
	Infer(ctx context.Context, system string, messages []Message, opts InferOptions) (*InferResult, error)

	// Stream sends a prompt and returns a channel of response chunks.
	Stream(ctx context.Context, system string, messages []Message, opts InferOptions) (<-chan string, <-chan error, error)

	// Name returns the provider identifier.
	Name() string
}

// NewProvider creates a provider instance based on the provider name.
func NewProvider(name string, model string) (Provider, error) {
	switch strings.ToLower(name) {
	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set — get your API key at https://console.anthropic.com/settings/keys")
		}
		return NewAnthropicProvider(apiKey, model), nil
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
		}
		return NewOpenAIProvider(apiKey, model), nil
	case "ollama":
		host := os.Getenv("OLLAMA_HOST")
		if host == "" {
			host = "http://localhost:11434"
		}
		return NewOllamaProvider(host, model), nil
	default:
		return nil, fmt.Errorf("unknown AI provider %q — supported providers: anthropic, openai, ollama", name)
	}
}
