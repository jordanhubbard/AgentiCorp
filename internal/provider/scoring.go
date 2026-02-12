package provider

import (
	"math"
	"sync"
	"time"
)

// ScoringWeights defines the priority weights for provider selection.
// Higher weight = more important. Weights are used for tie-breaking in priority order.
type ScoringWeights struct {
	ModelSize    float64 `json:"model_size"`    // Weight 1: Larger models are better (highest priority)
	RoundTrip    float64 `json:"round_trip"`    // Weight 2: Heartbeat/connectivity latency
	RequestLatency float64 `json:"request_latency"` // Weight 3: Per-request response time
	Cost         float64 `json:"cost"`          // Weight 4: $/token cost (lowest priority, placeholder)
}

// DefaultWeights returns the default scoring weights.
// The weights are set so that factors are evaluated in priority order:
// model size > round trip > request latency > cost
func DefaultWeights() ScoringWeights {
	return ScoringWeights{
		ModelSize:      1000.0, // Dominates all other factors
		RoundTrip:      100.0,  // Secondary factor
		RequestLatency: 10.0,   // Tertiary factor
		Cost:           1.0,    // Tie-breaker (currently $0 for all)
	}
}

// ProviderScore holds the computed dynamic score for a provider.
type ProviderScore struct {
	ProviderID string `json:"provider_id"`

	// Component scores (0-100 scale, higher is better)
	ModelSizeScore      float64 `json:"model_size_score"`
	RoundTripScore      float64 `json:"round_trip_score"`
	RequestLatencyScore float64 `json:"request_latency_score"`
	CostScore           float64 `json:"cost_score"`

	// Weighted composite score
	CompositeScore float64 `json:"composite_score"`

	// Raw metrics used for scoring
	ModelParamsB       float64 `json:"model_params_b"`        // Total or active parameters in billions
	HeartbeatLatencyMs int64   `json:"heartbeat_latency_ms"`  // Last heartbeat round-trip time
	AvgRequestLatencyMs float64 `json:"avg_request_latency_ms"` // Rolling average request latency
	CostPerMToken      float64 `json:"cost_per_mtoken"`       // Cost per million tokens

	LastUpdated time.Time `json:"last_updated"`
}

// Scorer computes dynamic provider scores based on runtime metrics.
type Scorer struct {
	mu      sync.RWMutex
	weights ScoringWeights
	scores  map[string]*ProviderScore // providerID -> score

	// Normalization bounds (learned from observed data)
	maxModelParams       float64
	maxHeartbeatLatency  float64
	maxRequestLatency    float64
	maxCost              float64
}

// NewScorer creates a new provider scorer with default weights.
func NewScorer() *Scorer {
	return &Scorer{
		weights:              DefaultWeights(),
		scores:               make(map[string]*ProviderScore),
		maxModelParams:       500.0,  // 500B params as baseline max
		maxHeartbeatLatency:  5000.0, // 5 seconds as baseline max
		maxRequestLatency:    30000.0, // 30 seconds as baseline max
		maxCost:              10.0,    // $10/M tokens as baseline max
	}
}

// SetWeights updates the scoring weights.
func (s *Scorer) SetWeights(w ScoringWeights) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weights = w
}

// GetWeights returns the current scoring weights.
func (s *Scorer) GetWeights() ScoringWeights {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.weights
}

// UpdateProviderMetrics updates the metrics for a provider and recalculates its score.
func (s *Scorer) UpdateProviderMetrics(
	providerID string,
	modelParamsB float64,
	heartbeatLatencyMs int64,
	avgRequestLatencyMs float64,
	costPerMToken float64,
) *ProviderScore {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update normalization bounds if we see larger values
	if modelParamsB > s.maxModelParams {
		s.maxModelParams = modelParamsB
	}
	if float64(heartbeatLatencyMs) > s.maxHeartbeatLatency {
		s.maxHeartbeatLatency = float64(heartbeatLatencyMs)
	}
	if avgRequestLatencyMs > s.maxRequestLatency {
		s.maxRequestLatency = avgRequestLatencyMs
	}
	if costPerMToken > s.maxCost && costPerMToken > 0 {
		s.maxCost = costPerMToken
	}

	score := &ProviderScore{
		ProviderID:          providerID,
		ModelParamsB:        modelParamsB,
		HeartbeatLatencyMs:  heartbeatLatencyMs,
		AvgRequestLatencyMs: avgRequestLatencyMs,
		CostPerMToken:       costPerMToken,
		LastUpdated:         time.Now(),
	}

	// Calculate component scores (0-100 scale)
	score.ModelSizeScore = s.scoreModelSize(modelParamsB)
	score.RoundTripScore = s.scoreRoundTrip(heartbeatLatencyMs)
	score.RequestLatencyScore = s.scoreRequestLatency(avgRequestLatencyMs)
	score.CostScore = s.scoreCost(costPerMToken)

	// Calculate weighted composite score
	score.CompositeScore = s.calculateComposite(score)

	s.scores[providerID] = score
	return score
}

// GetScore returns the current score for a provider.
func (s *Scorer) GetScore(providerID string) (*ProviderScore, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	score, ok := s.scores[providerID]
	return score, ok
}

// GetCompositeScore returns just the composite score for a provider (0 if not found).
func (s *Scorer) GetCompositeScore(providerID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if score, ok := s.scores[providerID]; ok {
		return score.CompositeScore
	}
	return 0
}

// scoreModelSize converts model params to a 0-100 score.
// Larger models score higher. Uses log scale for better differentiation.
func (s *Scorer) scoreModelSize(paramsB float64) float64 {
	if paramsB <= 0 {
		return 0
	}
	// Log scale: 1B -> ~0, 10B -> ~33, 100B -> ~67, 500B -> ~100
	logParams := math.Log10(paramsB + 1)
	logMax := math.Log10(s.maxModelParams + 1)
	if logMax <= 0 {
		return 50 // Default if no max set
	}
	score := (logParams / logMax) * 100
	return clamp(score, 0, 100)
}

// scoreRoundTrip converts heartbeat latency to a 0-100 score.
// Lower latency scores higher.
func (s *Scorer) scoreRoundTrip(latencyMs int64) float64 {
	if latencyMs <= 0 {
		return 100 // No latency data = assume fast
	}
	// Inverse relationship: lower latency = higher score
	// 0ms -> 100, 1000ms -> 50, 5000ms -> 0
	ratio := float64(latencyMs) / s.maxHeartbeatLatency
	score := (1 - ratio) * 100
	return clamp(score, 0, 100)
}

// scoreRequestLatency converts average request latency to a 0-100 score.
// Lower latency scores higher.
func (s *Scorer) scoreRequestLatency(avgLatencyMs float64) float64 {
	if avgLatencyMs <= 0 {
		return 100 // No data = assume fast
	}
	// Inverse relationship with diminishing returns
	// 0ms -> 100, 5000ms -> 50, 30000ms -> 0
	ratio := avgLatencyMs / s.maxRequestLatency
	score := (1 - ratio) * 100
	return clamp(score, 0, 100)
}

// scoreCost converts cost per million tokens to a 0-100 score.
// Lower cost scores higher. $0 = 100 (best).
func (s *Scorer) scoreCost(costPerMToken float64) float64 {
	if costPerMToken <= 0 {
		return 100 // Free = best score
	}
	// Inverse relationship: lower cost = higher score
	ratio := costPerMToken / s.maxCost
	score := (1 - ratio) * 100
	return clamp(score, 0, 100)
}

// calculateComposite computes the weighted composite score.
// The weights ensure that factors are evaluated in priority order.
func (s *Scorer) calculateComposite(score *ProviderScore) float64 {
	// Each factor contributes: weight * score / 100
	// This gives each factor a maximum contribution equal to its weight.
	composite := 0.0
	composite += s.weights.ModelSize * (score.ModelSizeScore / 100)
	composite += s.weights.RoundTrip * (score.RoundTripScore / 100)
	composite += s.weights.RequestLatency * (score.RequestLatencyScore / 100)
	composite += s.weights.Cost * (score.CostScore / 100)
	return composite
}

// RankProviders returns provider IDs sorted by their composite score (highest first).
func (s *Scorer) RankProviders(providerIDs []string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type ranked struct {
		id    string
		score float64
	}

	providers := make([]ranked, 0, len(providerIDs))
	for _, id := range providerIDs {
		score := 0.0
		if ps, ok := s.scores[id]; ok {
			score = ps.CompositeScore
		}
		providers = append(providers, ranked{id: id, score: score})
	}

	// Sort by score descending
	for i := 0; i < len(providers)-1; i++ {
		for j := i + 1; j < len(providers); j++ {
			if providers[j].score > providers[i].score {
				providers[i], providers[j] = providers[j], providers[i]
			}
		}
	}

	result := make([]string, len(providers))
	for i, p := range providers {
		result[i] = p.id
	}
	return result
}

// RankProvidersForComplexity returns provider IDs ranked by suitability for a complexity level.
// Providers that match the complexity tier are ranked first (by their other metrics),
// followed by overqualified providers (smallest first to minimize waste), then underqualified ones.
func (s *Scorer) RankProvidersForComplexity(providerIDs []string, complexity ComplexityLevel) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requiredTier := RequiredModelTier(complexity)

	type ranked struct {
		id          string
		score       float64
		paramsB     float64
		modelTier   ModelTier
		tierMatch   int // 0 = exact match, 1 = overqualified, 2 = underqualified
	}

	providers := make([]ranked, 0, len(providerIDs))
	for _, id := range providerIDs {
		score := 0.0
		var paramsB float64
		if ps, ok := s.scores[id]; ok {
			score = ps.CompositeScore
			paramsB = ps.ModelParamsB
		}

		tier := GetModelTier(paramsB)
		var tierMatch int
		switch {
		case tier == requiredTier:
			tierMatch = 0 // Perfect match - highest priority
		case tier > requiredTier:
			tierMatch = 1 // Overqualified - can do the job but wasteful
		default:
			tierMatch = 2 // Underqualified - may not be capable
		}

		providers = append(providers, ranked{
			id:        id,
			score:     score,
			paramsB:   paramsB,
			modelTier: tier,
			tierMatch: tierMatch,
		})
	}

	// Sort by:
	// 1. tierMatch ascending (exact > over > under)
	// 2. For overqualified: prefer smaller models (less waste)
	// 3. For exact match: prefer higher score (better metrics)
	// 4. For underqualified: prefer larger models (closer to capable)
	for i := 0; i < len(providers)-1; i++ {
		for j := i + 1; j < len(providers); j++ {
			swap := false
			if providers[j].tierMatch < providers[i].tierMatch {
				swap = true
			} else if providers[j].tierMatch == providers[i].tierMatch {
				switch providers[i].tierMatch {
				case 0: // Exact match - prefer higher score
					swap = providers[j].score > providers[i].score
				case 1: // Overqualified - prefer smaller model (less waste)
					swap = providers[j].paramsB < providers[i].paramsB
				case 2: // Underqualified - prefer larger model (closer to capable)
					swap = providers[j].paramsB > providers[i].paramsB
				}
			}
			if swap {
				providers[i], providers[j] = providers[j], providers[i]
			}
		}
	}

	result := make([]string, len(providers))
	for i, p := range providers {
		result[i] = p.id
	}
	return result
}

// SelectBestForComplexity selects the best provider for a given complexity.
// It prefers providers that match the required tier, falling back to overqualified
// providers if no exact match exists.
func (s *Scorer) SelectBestForComplexity(providerIDs []string, complexity ComplexityLevel) (string, float64, bool) {
	ranked := s.RankProvidersForComplexity(providerIDs, complexity)
	if len(ranked) == 0 {
		return "", 0, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	bestID := ranked[0]
	score := 0.0
	if ps, ok := s.scores[bestID]; ok {
		score = ps.CompositeScore
	}
	return bestID, score, true
}

// RemoveProvider removes a provider from the scorer.
func (s *Scorer) RemoveProvider(providerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.scores, providerID)
}

// clamp restricts a value to a range.
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
