package memory

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode"
)

// Embedder generates vector embeddings from text.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// ---- Provider-based embedder (OpenAI-compatible /v1/embeddings) ----

// ProviderEmbedder calls an OpenAI-compatible embedding endpoint.
type ProviderEmbedder struct {
	endpoint string // e.g. "http://localhost:11434" or "https://api.openai.com"
	apiKey   string
	model    string // e.g. "text-embedding-3-small"
	client   *http.Client
}

// NewProviderEmbedder creates an embedder that calls /v1/embeddings.
func NewProviderEmbedder(endpoint, apiKey, model string) *ProviderEmbedder {
	return &ProviderEmbedder{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		apiKey:   apiKey,
		model:    model,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (e *ProviderEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := embeddingRequest{
		Model: e.model,
		Input: texts,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	url := e.endpoint + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}

	if len(result.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(result.Data))
	}

	embeddings := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}
	return embeddings, nil
}

// ---- Hash-based embedder (TF-IDF hashing trick, no external dependencies) ----

const hashDimensions = 256

// HashEmbedder creates fixed-dimension vectors using the hashing trick.
// Each word is hashed to a position in the vector and TF weights are applied.
// This provides rough semantic similarity without any external model.
type HashEmbedder struct{}

func NewHashEmbedder() *HashEmbedder {
	return &HashEmbedder{}
}

func (e *HashEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		result[i] = hashEmbed(text)
	}
	return result, nil
}

// hashEmbed creates a vector by hashing word tokens into fixed dimensions.
func hashEmbed(text string) []float32 {
	vec := make([]float32, hashDimensions)
	words := tokenize(text)
	if len(words) == 0 {
		return vec
	}

	for _, word := range words {
		h := fnv.New32a()
		h.Write([]byte(word))
		idx := h.Sum32() % uint32(hashDimensions)

		// Use a second hash for sign (feature hashing trick)
		h2 := fnv.New32()
		h2.Write([]byte(word))
		sign := float32(1.0)
		if h2.Sum32()%2 == 0 {
			sign = -1.0
		}
		vec[idx] += sign
	}

	// L2 normalize
	normalize(vec)
	return vec
}

// tokenize splits text into lowercase word tokens, filtering short/stop words.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})

	var result []string
	for _, w := range words {
		if len(w) >= 2 && !isStopWord(w) {
			result = append(result, w)
		}
	}
	return result
}

var stopWords = map[string]bool{
	"the": true, "is": true, "at": true, "on": true, "in": true,
	"to": true, "for": true, "of": true, "and": true, "or": true,
	"an": true, "it": true, "be": true, "as": true, "do": true,
	"by": true, "so": true, "if": true, "no": true, "up": true,
	"was": true, "are": true, "has": true, "had": true, "not": true,
	"but": true, "its": true, "can": true, "did": true, "all": true,
	"this": true, "that": true, "with": true, "from": true, "have": true,
	"they": true, "been": true, "will": true, "were": true, "than": true,
	"what": true, "when": true, "each": true, "which": true, "their": true,
	"said": true, "them": true, "would": true, "there": true, "could": true,
}

func isStopWord(w string) bool {
	return stopWords[w]
}

// ---- Vector math ----

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value in [-1, 1], or 0 if either vector is zero.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return float32(dot / denom)
}

func normalize(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return
	}
	for i := range vec {
		vec[i] = float32(float64(vec[i]) / norm)
	}
}

// ---- Encoding helpers for BLOB storage ----

// EncodeEmbedding converts a float32 slice to bytes for SQLite BLOB storage.
func EncodeEmbedding(vec []float32) []byte {
	if len(vec) == 0 {
		return nil
	}
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// DecodeEmbedding converts bytes from SQLite BLOB back to a float32 slice.
func DecodeEmbedding(data []byte) []float32 {
	if len(data) == 0 || len(data)%4 != 0 {
		return nil
	}
	vec := make([]float32, len(data)/4)
	for i := range vec {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return vec
}

// ---- Fallback embedder ----

// FallbackEmbedder tries the primary embedder and falls back to hash embedding.
type FallbackEmbedder struct {
	primary  Embedder
	fallback *HashEmbedder
}

// NewFallbackEmbedder creates an embedder that tries primary first, then hash.
func NewFallbackEmbedder(primary Embedder) *FallbackEmbedder {
	return &FallbackEmbedder{
		primary:  primary,
		fallback: NewHashEmbedder(),
	}
}

func (e *FallbackEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if e.primary != nil {
		result, err := e.primary.Embed(ctx, texts)
		if err == nil {
			return result, nil
		}
		// Fall through to hash embedder
	}
	return e.fallback.Embed(ctx, texts)
}
