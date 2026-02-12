package memory

import (
	"context"
	"math"
	"testing"
)

func TestHashEmbedder_Basic(t *testing.T) {
	e := NewHashEmbedder()
	ctx := context.Background()

	embeddings, err := e.Embed(ctx, []string{"build failure in Go compiler"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(embeddings))
	}
	if len(embeddings[0]) != hashDimensions {
		t.Fatalf("expected %d dimensions, got %d", hashDimensions, len(embeddings[0]))
	}
}

func TestHashEmbedder_MultipleTexts(t *testing.T) {
	e := NewHashEmbedder()
	ctx := context.Background()

	texts := []string{"compiler error", "test failure", "edit problem"}
	embeddings, err := e.Embed(ctx, texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embeddings) != 3 {
		t.Fatalf("expected 3 embeddings, got %d", len(embeddings))
	}
}

func TestHashEmbedder_Normalized(t *testing.T) {
	e := NewHashEmbedder()
	ctx := context.Background()

	embeddings, err := e.Embed(ctx, []string{"some text to embed for normalization check"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check L2 norm is approximately 1.0
	var sum float64
	for _, v := range embeddings[0] {
		sum += float64(v) * float64(v)
	}
	norm := math.Sqrt(sum)
	if math.Abs(norm-1.0) > 0.01 {
		t.Fatalf("expected L2 norm ~1.0, got %f", norm)
	}
}

func TestCosineSimilarity_Identical(t *testing.T) {
	vec := []float32{1.0, 2.0, 3.0}
	sim := CosineSimilarity(vec, vec)
	if math.Abs(float64(sim)-1.0) > 0.001 {
		t.Fatalf("expected similarity ~1.0 for identical vectors, got %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1.0, 0.0, 0.0}
	b := []float32{0.0, 1.0, 0.0}
	sim := CosineSimilarity(a, b)
	if math.Abs(float64(sim)) > 0.001 {
		t.Fatalf("expected similarity ~0.0 for orthogonal vectors, got %f", sim)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float32{1.0, 0.0}
	b := []float32{-1.0, 0.0}
	sim := CosineSimilarity(a, b)
	if math.Abs(float64(sim)+1.0) > 0.001 {
		t.Fatalf("expected similarity ~-1.0 for opposite vectors, got %f", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1.0, 2.0}
	b := []float32{1.0}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Fatalf("expected 0 for different length vectors, got %f", sim)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0.0, 0.0}
	b := []float32{1.0, 2.0}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Fatalf("expected 0 for zero vector, got %f", sim)
	}
}

func TestHashEmbedder_SimilarTexts(t *testing.T) {
	e := NewHashEmbedder()
	ctx := context.Background()

	embeddings, err := e.Embed(ctx, []string{
		"build failure in Go compiler with syntax error",
		"Go compiler build failure due to syntax issue",
		"database connection timeout in production server",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Similar texts should have higher similarity than dissimilar ones
	simSimilar := CosineSimilarity(embeddings[0], embeddings[1])
	simDifferent := CosineSimilarity(embeddings[0], embeddings[2])

	if simSimilar <= simDifferent {
		t.Fatalf("expected similar texts to have higher similarity (%f) than different texts (%f)",
			simSimilar, simDifferent)
	}
}

func TestEncodeDecodeEmbedding(t *testing.T) {
	original := []float32{1.0, -2.5, 3.14, 0.0, -0.001}
	encoded := EncodeEmbedding(original)
	decoded := DecodeEmbedding(encoded)

	if len(decoded) != len(original) {
		t.Fatalf("expected %d elements, got %d", len(original), len(decoded))
	}
	for i := range original {
		if original[i] != decoded[i] {
			t.Fatalf("element %d: expected %f, got %f", i, original[i], decoded[i])
		}
	}
}

func TestEncodeDecodeEmbedding_Empty(t *testing.T) {
	encoded := EncodeEmbedding(nil)
	if encoded != nil {
		t.Fatalf("expected nil for nil input")
	}

	decoded := DecodeEmbedding(nil)
	if decoded != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestEncodeDecodeEmbedding_InvalidBytes(t *testing.T) {
	decoded := DecodeEmbedding([]byte{1, 2, 3}) // not divisible by 4
	if decoded != nil {
		t.Fatalf("expected nil for invalid byte length")
	}
}

func TestFallbackEmbedder_UsesPrimary(t *testing.T) {
	primary := NewHashEmbedder()
	fb := NewFallbackEmbedder(primary)
	ctx := context.Background()

	embeddings, err := fb.Embed(ctx, []string{"test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(embeddings))
	}
}

func TestFallbackEmbedder_NilPrimary(t *testing.T) {
	fb := NewFallbackEmbedder(nil)
	ctx := context.Background()

	embeddings, err := fb.Embed(ctx, []string{"test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(embeddings))
	}
}

func TestHashEmbedder_EmptyText(t *testing.T) {
	e := NewHashEmbedder()
	ctx := context.Background()

	embeddings, err := e.Embed(ctx, []string{""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(embeddings))
	}
	// Empty text should produce zero vector
	for _, v := range embeddings[0] {
		if v != 0 {
			t.Fatalf("expected zero vector for empty text")
		}
	}
}
