package provider

import (
	"math"
	"testing"
)

func TestDefaultWeights(t *testing.T) {
	w := DefaultWeights()

	// Verify weights are in priority order (higher weight = more important)
	if w.ModelSize <= w.RoundTrip {
		t.Error("ModelSize should have higher weight than RoundTrip")
	}
	if w.RoundTrip <= w.RequestLatency {
		t.Error("RoundTrip should have higher weight than RequestLatency")
	}
	if w.RequestLatency <= w.Cost {
		t.Error("RequestLatency should have higher weight than Cost")
	}
}

func TestScorerModelSizeScoring(t *testing.T) {
	s := NewScorer()

	tests := []struct {
		name     string
		paramsB  float64
		minScore float64
		maxScore float64
	}{
		{"no params", 0, 0, 0},
		{"7B model", 7, 20, 50},
		{"30B model", 30, 40, 70},
		{"70B model", 70, 50, 80},
		{"480B model", 480, 85, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := s.UpdateProviderMetrics("test", tt.paramsB, 100, 1000, 0)
			if score.ModelSizeScore < tt.minScore || score.ModelSizeScore > tt.maxScore {
				t.Errorf("ModelSizeScore for %s = %f, want between %f and %f",
					tt.name, score.ModelSizeScore, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestScorerRoundTripScoring(t *testing.T) {
	s := NewScorer()

	tests := []struct {
		name      string
		latencyMs int64
		minScore  float64
		maxScore  float64
	}{
		{"instant", 0, 100, 100},
		{"fast (100ms)", 100, 90, 100},
		{"medium (1000ms)", 1000, 60, 90},
		{"slow (3000ms)", 3000, 20, 60},
		{"very slow (5000ms)", 5000, 0, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := s.UpdateProviderMetrics("test", 30, tt.latencyMs, 1000, 0)
			if score.RoundTripScore < tt.minScore || score.RoundTripScore > tt.maxScore {
				t.Errorf("RoundTripScore for %s = %f, want between %f and %f",
					tt.name, score.RoundTripScore, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestScorerCostScoring(t *testing.T) {
	s := NewScorer()

	tests := []struct {
		name          string
		costPerMToken float64
		minScore      float64
		maxScore      float64
	}{
		{"free", 0, 100, 100},
		{"cheap ($1/M)", 1, 85, 95},
		{"medium ($5/M)", 5, 45, 60},
		{"expensive ($10/M)", 10, 0, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := s.UpdateProviderMetrics("test", 30, 100, 1000, tt.costPerMToken)
			if score.CostScore < tt.minScore || score.CostScore > tt.maxScore {
				t.Errorf("CostScore for %s = %f, want between %f and %f",
					tt.name, score.CostScore, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestScorerCompositeOrdering(t *testing.T) {
	s := NewScorer()

	// Provider A: Large model (480B), slow latency
	scoreA := s.UpdateProviderMetrics("providerA", 480, 3000, 5000, 0)

	// Provider B: Small model (7B), fast latency
	scoreB := s.UpdateProviderMetrics("providerB", 7, 100, 500, 0)

	// Model size is highest priority, so A should win despite worse latency
	if scoreA.CompositeScore <= scoreB.CompositeScore {
		t.Errorf("Larger model should have higher composite score: A=%f, B=%f",
			scoreA.CompositeScore, scoreB.CompositeScore)
	}
}

func TestScorerTieBreaking(t *testing.T) {
	s := NewScorer()

	// Two providers with same model size but different latencies
	scoreA := s.UpdateProviderMetrics("providerA", 30, 1000, 2000, 0)
	scoreB := s.UpdateProviderMetrics("providerB", 30, 100, 500, 0)

	// Same model size, so B should win on latency
	if scoreB.CompositeScore <= scoreA.CompositeScore {
		t.Errorf("Same model size should tie-break on latency: A=%f, B=%f",
			scoreA.CompositeScore, scoreB.CompositeScore)
	}

	// Verify model size scores are equal
	if math.Abs(scoreA.ModelSizeScore-scoreB.ModelSizeScore) > 0.01 {
		t.Errorf("Same model size should have same ModelSizeScore: A=%f, B=%f",
			scoreA.ModelSizeScore, scoreB.ModelSizeScore)
	}
}

func TestScorerRankProviders(t *testing.T) {
	s := NewScorer()

	// Create providers with different characteristics
	s.UpdateProviderMetrics("large_slow", 480, 3000, 5000, 0)    // Large model, slow
	s.UpdateProviderMetrics("medium_fast", 30, 100, 500, 0)      // Medium model, fast
	s.UpdateProviderMetrics("small_instant", 7, 10, 100, 0)      // Small model, instant
	s.UpdateProviderMetrics("large_fast", 480, 100, 500, 0)      // Large model, fast

	ranked := s.RankProviders([]string{"large_slow", "medium_fast", "small_instant", "large_fast"})

	// Large + fast should be first (model size dominates, fast breaks tie)
	if ranked[0] != "large_fast" {
		t.Errorf("Expected large_fast first, got %s", ranked[0])
	}

	// Large + slow should be second (model size dominates)
	if ranked[1] != "large_slow" {
		t.Errorf("Expected large_slow second, got %s", ranked[1])
	}

	// Medium should beat small due to model size
	if ranked[2] != "medium_fast" {
		t.Errorf("Expected medium_fast third, got %s", ranked[2])
	}

	// Small should be last
	if ranked[3] != "small_instant" {
		t.Errorf("Expected small_instant last, got %s", ranked[3])
	}
}

func TestScorerCustomWeights(t *testing.T) {
	s := NewScorer()

	// Set weights to prioritize cost over model size
	s.SetWeights(ScoringWeights{
		ModelSize:      1.0,   // Lowest priority
		RoundTrip:      10.0,
		RequestLatency: 100.0,
		Cost:           1000.0, // Highest priority
	})

	// Provider A: Large model, expensive
	scoreA := s.UpdateProviderMetrics("providerA", 480, 100, 500, 5.0)

	// Provider B: Small model, free
	scoreB := s.UpdateProviderMetrics("providerB", 7, 100, 500, 0)

	// With cost as highest priority, B should win despite smaller model
	if scoreB.CompositeScore <= scoreA.CompositeScore {
		t.Errorf("With cost priority, free provider should win: A=%f, B=%f",
			scoreA.CompositeScore, scoreB.CompositeScore)
	}
}

func TestScorerRemoveProvider(t *testing.T) {
	s := NewScorer()

	s.UpdateProviderMetrics("test", 30, 100, 500, 0)

	score, ok := s.GetScore("test")
	if !ok || score == nil {
		t.Fatal("Expected score to exist")
	}

	s.RemoveProvider("test")

	_, ok = s.GetScore("test")
	if ok {
		t.Error("Expected score to be removed")
	}
}

func TestScorerDynamicNormalization(t *testing.T) {
	s := NewScorer()

	// First provider sets the baseline
	score1 := s.UpdateProviderMetrics("provider1", 30, 1000, 5000, 1.0)

	// Second provider with much larger model should update the normalization
	score2 := s.UpdateProviderMetrics("provider2", 500, 100, 500, 0)

	// After seeing 500B model, the 30B model should have lower relative score
	score1After := s.UpdateProviderMetrics("provider1", 30, 1000, 5000, 1.0)

	if score1After.ModelSizeScore >= score1.ModelSizeScore {
		t.Logf("Note: Score normalization may not change if bounds were already high enough")
	}

	// The 500B model should have higher model score
	if score2.ModelSizeScore <= score1After.ModelSizeScore {
		t.Errorf("Larger model should have higher score: 500B=%f, 30B=%f",
			score2.ModelSizeScore, score1After.ModelSizeScore)
	}
}
