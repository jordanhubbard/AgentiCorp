package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// StreamChunk represents a chunk in a streaming response
type StreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// StreamHandler handles streaming responses
type StreamHandler func(chunk *StreamChunk) error

// CreateChatCompletionStream sends a streaming chat completion request
func (p *OpenAIProvider) CreateChatCompletionStream(ctx context.Context, req *ChatCompletionRequest, handler StreamHandler) error {
	// Ensure stream is enabled
	req.Stream = true

	url := fmt.Sprintf("%s/chat/completions", p.endpoint)

	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	}

	// Use streaming client (no timeout) for streaming requests.
	// The context controls cancellation; this prevents mid-stream timeouts.
	client := p.streamingClient
	if client == nil {
		client = p.client // fallback for tests
	}

	// Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		bodyStr := string(respBody)
		if resp.StatusCode == http.StatusBadRequest && isContextLengthError(bodyStr) {
			return &ContextLengthError{StatusCode: resp.StatusCode, Body: bodyStr}
		}
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, bodyStr)
	}

	// Read streaming response
	return p.readStreamingResponse(ctx, resp.Body, handler)
}

// readStreamingResponse reads and processes SSE streaming response
func (p *OpenAIProvider) readStreamingResponse(ctx context.Context, reader io.Reader, handler StreamHandler) error {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for potentially large JSON chunks
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	chunksReceived := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			if chunksReceived > 0 {
				return fmt.Errorf("stream interrupted after %d chunks: %w", chunksReceived, ctx.Err())
			}
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE format: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end marker
		if data == "[DONE]" {
			return nil
		}

		// Parse chunk JSON
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Log error but continue reading
			continue
		}

		chunksReceived++

		// Call handler with chunk
		if err := handler(&chunk); err != nil {
			return fmt.Errorf("handler error after %d chunks: %w", chunksReceived, err)
		}
	}

	if err := scanner.Err(); err != nil {
		if chunksReceived > 0 {
			return fmt.Errorf("stream connection lost after %d chunks: %w", chunksReceived, err)
		}
		return fmt.Errorf("stream read error: %w", err)
	}

	// Stream ended without [DONE] marker — connection may have been closed
	if chunksReceived == 0 {
		return fmt.Errorf("stream ended without receiving any data")
	}

	return nil
}
