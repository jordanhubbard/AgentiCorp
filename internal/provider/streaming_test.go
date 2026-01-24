package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamingChatCompletion(t *testing.T) {
	// Create mock SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming is requested
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Expected Accept: text/event-stream, got %s", r.Header.Get("Accept"))
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Send chunks
		chunks := []string{
			`data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"test","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"}}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"test","choices":[{"index":0,"delta":{"content":" world"}}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"test","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	// Create provider
	provider := NewOpenAIProvider(server.URL, "test-key")

	// Make streaming request
	req := &ChatCompletionRequest{
		Model: "test-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	// Collect chunks
	var chunks []*StreamChunk
	err := provider.CreateChatCompletionStream(context.Background(), req, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})

	if err != nil {
		t.Fatalf("Streaming failed: %v", err)
	}

	// Verify chunks received
	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	// Verify content
	var content strings.Builder
	for _, chunk := range chunks {
		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	expected := "Hello world!"
	if content.String() != expected {
		t.Errorf("Expected content %q, got %q", expected, content.String())
	}

	// Verify finish reason
	if len(chunks) > 0 && len(chunks[len(chunks)-1].Choices) > 0 {
		finishReason := chunks[len(chunks)-1].Choices[0].FinishReason
		if finishReason != "stop" {
			t.Errorf("Expected finish_reason 'stop', got %q", finishReason)
		}
	}
}

func TestStreamingContextCancellation(t *testing.T) {
	// Create server that sends infinite stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Send one chunk then hang
		w.Write([]byte(`data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"test","choices":[{"index":0,"delta":{"content":"test"}}]}` + "\n\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Block until request context is done (allows proper cleanup)
		<-r.Context().Done()
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "test-key")

	req := &ChatCompletionRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "test"}},
		Stream:   true,
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first chunk
	chunkCount := 0
	err := provider.CreateChatCompletionStream(ctx, req, func(chunk *StreamChunk) error {
		chunkCount++
		if chunkCount == 1 {
			cancel() // Cancel after first chunk
		}
		return nil
	})

	// Should get context canceled error
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if chunkCount != 1 {
		t.Errorf("Expected 1 chunk before cancellation, got %d", chunkCount)
	}
}
