package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	anthropicAPIURL       = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion   = "2023-06-01"
	defaultAnthropicModel = "claude-sonnet-4-20250514"
	maxRetries            = 3
)

// AnthropicProvider implements the Provider interface for Anthropic's Claude models.
type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider with the given API key and model.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = defaultAnthropicModel
	}
	return &AnthropicProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// Name returns the provider identifier.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Model string `json:"model"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Infer sends a prompt to Claude and returns the complete response.
func (p *AnthropicProvider) Infer(ctx context.Context, system string, messages []Message, opts InferOptions) (*InferResult, error) {
	model := p.model
	if opts.Model != "" {
		model = opts.Model
	}
	maxTokens := 4096
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}

	msgs := make([]anthropicMessage, len(messages))
	for i, m := range messages {
		msgs[i] = anthropicMessage(m)
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  msgs,
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		result, err := p.doRequest(ctx, reqBody)
		if err != nil {
			lastErr = err
			// Retry on rate limit or server errors
			if isRetryable(err) {
				continue
			}
			return nil, err
		}
		return result, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

// Stream sends a prompt to Claude and returns a channel of streamed text chunks.
func (p *AnthropicProvider) Stream(ctx context.Context, system string, messages []Message, opts InferOptions) (<-chan string, <-chan error, error) {
	model := p.model
	if opts.Model != "" {
		model = opts.Model
	}
	maxTokens := 4096
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}

	msgs := make([]anthropicMessage, len(messages))
	for i, m := range messages {
		msgs[i] = anthropicMessage(m)
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  msgs,
		Stream:    true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("could not marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	textCh := make(chan string, 64)
	errCh := make(chan error, 1)

	go func() {
		defer close(textCh)
		defer close(errCh)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}

			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if event.Type == "content_block_delta" && event.Delta.Text != "" {
				select {
				case textCh <- event.Delta.Text:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	return textCh, errCh, nil
}

func (p *AnthropicProvider) doRequest(ctx context.Context, reqBody anthropicRequest) (*InferResult, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &retryableError{msg: "rate limited by Anthropic API"}
	}
	if resp.StatusCode >= 500 {
		return nil, &retryableError{msg: fmt.Sprintf("server error (HTTP %d)", resp.StatusCode)}
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("could not parse API response: %w", err)
	}

	if apiResp.Error != nil {
		if apiResp.Error.Type == "authentication_error" {
			return nil, fmt.Errorf("invalid API key — check your ANTHROPIC_API_KEY environment variable")
		}
		return nil, fmt.Errorf("API error (%s): %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("API returned empty response")
	}

	var text strings.Builder
	for _, block := range apiResp.Content {
		text.WriteString(block.Text)
	}

	return &InferResult{
		Content:      text.String(),
		Model:        apiResp.Model,
		InputTokens:  apiResp.Usage.InputTokens,
		OutputTokens: apiResp.Usage.OutputTokens,
	}, nil
}

type retryableError struct {
	msg string
}

func (e *retryableError) Error() string {
	return e.msg
}

func isRetryable(err error) bool {
	_, ok := err.(*retryableError)
	return ok
}
