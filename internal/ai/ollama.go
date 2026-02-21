package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultOllamaModel = "llama3.1"

// OllamaProvider implements the Provider interface for local Ollama models.
type OllamaProvider struct {
	host   string
	model  string
	client *http.Client
}

// NewOllamaProvider creates a new Ollama provider with the given host and model.
func NewOllamaProvider(host, model string) *OllamaProvider {
	if model == "" {
		model = defaultOllamaModel
	}
	return &OllamaProvider{
		host:   host,
		model:  model,
		client: &http.Client{Timeout: 300 * time.Second},
	}
}

// Name returns the provider identifier.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// Infer sends a prompt to Ollama and returns the complete response.
func (p *OllamaProvider) Infer(ctx context.Context, system string, messages []Message, opts InferOptions) (*InferResult, error) {
	model := p.model
	if opts.Model != "" {
		model = opts.Model
	}

	msgs := make([]ollamaMessage, 0, len(messages)+1)
	if system != "" {
		msgs = append(msgs, ollamaMessage{Role: "system", Content: system})
	}
	for _, m := range messages {
		msgs = append(msgs, ollamaMessage(m))
	}

	reqBody := ollamaRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request: %w", err)
	}

	url := p.host + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not connect to Ollama at %s — is Ollama running? Start it with 'ollama serve'", p.host)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp ollamaResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("could not parse response: %w", err)
	}

	return &InferResult{
		Content: apiResp.Message.Content,
		Model:   model,
	}, nil
}

// Stream sends a prompt to Ollama and returns a channel of streamed text chunks.
func (p *OllamaProvider) Stream(ctx context.Context, system string, messages []Message, opts InferOptions) (<-chan string, <-chan error, error) {
	model := p.model
	if opts.Model != "" {
		model = opts.Model
	}

	msgs := make([]ollamaMessage, 0, len(messages)+1)
	if system != "" {
		msgs = append(msgs, ollamaMessage{Role: "system", Content: system})
	}
	for _, m := range messages {
		msgs = append(msgs, ollamaMessage(m))
	}

	reqBody := ollamaRequest{
		Model:    model,
		Messages: msgs,
		Stream:   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("could not marshal request: %w", err)
	}

	url := p.host + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to Ollama at %s — is Ollama running?", p.host)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(respBody))
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
			var chunk ollamaResponse
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue
			}

			if chunk.Message.Content != "" {
				select {
				case textCh <- chunk.Message.Content:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}

			if chunk.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	return textCh, errCh, nil
}
