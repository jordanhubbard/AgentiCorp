package cache

import (
	"time"
)

// DuplicateRequest represents a detected duplicate request pattern
type DuplicateRequest struct {
	RequestHash     string    `json:"request_hash"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	OccurrenceCount int       `json:"occurrence_count"`
	ProviderID      string    `json:"provider_id"`
	ModelName       string    `json:"model_name"`
	TotalTokens     int64     `json:"total_tokens"`
	TotalCost       float64   `json:"total_cost"`
	AvgLatencyMs    int64     `json:"avg_latency_ms"`
	SampleRequest   string    `json:"sample_request,omitempty"`
	RequestIDs      []string  `json:"request_ids,omitempty"`
}

// CacheOpportunity represents a caching optimization opportunity
type CacheOpportunity struct {
	ID               string        `json:"id"`
	Pattern          string        `json:"pattern"`
	Description      string        `json:"description"`
	RequestCount     int           `json:"request_count"`
	PotentialHits    int           `json:"potential_hits"`
	HitRatePercent   float64       `json:"hit_rate_percent"`
	TokensSavable    int64         `json:"tokens_savable"`
	CostSavableUSD   float64       `json:"cost_savable_usd"`
	LatencySavableMs int64         `json:"latency_savable_ms"`
	Recommendation   string        `json:"recommendation"`
	AutoEnableable   bool          `json:"auto_enableable"`
	SuggestedTTL     time.Duration `json:"suggested_ttl"`
	Priority         string        `json:"priority"` // "high", "medium", "low"
	ProviderID       string        `json:"provider_id"`
	ModelName        string        `json:"model_name"`
}

// AnalysisReport provides a comprehensive cache analysis
type AnalysisReport struct {
	AnalyzedAt        time.Time           `json:"analyzed_at"`
	TimeWindow        time.Duration       `json:"time_window"`
	TimeWindowStart   time.Time           `json:"time_window_start"`
	TimeWindowEnd     time.Time           `json:"time_window_end"`
	TotalRequests     int64               `json:"total_requests"`
	UniqueRequests    int64               `json:"unique_requests"`
	DuplicateCount    int64               `json:"duplicate_count"`
	DuplicatePercent  float64             `json:"duplicate_percent"`
	Opportunities     []*CacheOpportunity `json:"opportunities"`
	TotalSavingsUSD   float64             `json:"total_savings_usd"`
	TotalTokensSaved  int64               `json:"total_tokens_saved"`
	TotalLatencySaved int64               `json:"total_latency_saved_ms"`
	MonthlyProjection float64             `json:"monthly_projection_usd"`
	Recommendations   []string            `json:"recommendations"`
	AutoOptimizations []string            `json:"auto_optimizations,omitempty"`
}

// AnalysisConfig configures the cache analysis
type AnalysisConfig struct {
	TimeWindow        time.Duration
	MinOccurrences    int     // Minimum duplicate count to consider
	MinSavingsUSD     float64 // Minimum savings threshold
	AutoEnable        bool    // Auto-enable caching for high-value opportunities
	AutoEnableMinUSD  float64 // Minimum monthly savings to auto-enable ($10 default)
	AutoEnableMinRate float64 // Minimum hit rate to auto-enable (0.5 = 50% default)
}

// DefaultAnalysisConfig returns sensible defaults
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		TimeWindow:        7 * 24 * time.Hour, // 7 days
		MinOccurrences:    2,                  // At least 2 occurrences
		MinSavingsUSD:     0.01,               // At least $0.01 savings
		AutoEnable:        false,              // Don't auto-enable by default
		AutoEnableMinUSD:  10.0,               // $10/month minimum for auto
		AutoEnableMinRate: 0.5,                // 50% hit rate minimum for auto
	}
}

// OptimizationResult represents the result of applying optimizations
type OptimizationResult struct {
	AppliedCount    int      `json:"applied_count"`
	SkippedCount    int      `json:"skipped_count"`
	TotalSavingsUSD float64  `json:"total_savings_usd"`
	AppliedPatterns []string `json:"applied_patterns"`
	SkippedPatterns []string `json:"skipped_patterns,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}
