package models

import (
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// Provider represents an AI engine running on-prem or in the cloud
// Providers may require credentials (keys) to communicate
type Provider struct {
	models.EntityMetadata `json:",inline"`

	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Type                   string    `json:"type"`     // openai, anthropic, local, etc.
	Endpoint               string    `json:"endpoint"` // URL or path to the provider
	Model                  string    `json:"model"`    // Legacy/default model for this provider
	ConfiguredModel        string    `json:"configured_model"`
	SelectedModel          string         `json:"selected_model"`
	SelectionReason        string         `json:"selection_reason"`
	ModelScore             float64        `json:"model_score"`
	SelectedGPU            string         `json:"selected_gpu"`
	GPUConstraints         *GPUConstraints `json:"gpu_constraints,omitempty"`
	Description            string         `json:"description"`
	RequiresKey            bool      `json:"requires_key"` // Whether this provider needs API credentials
	KeyID                  string    `json:"key_id"`       // Reference to encrypted key in key manager
	Status                 string    `json:"status"`       // active, inactive, etc.
	LastHeartbeatAt        time.Time `json:"last_heartbeat_at"`
	LastHeartbeatLatencyMs int64     `json:"last_heartbeat_latency_ms"`
	LastHeartbeatError     string    `json:"last_heartbeat_error"`
	
	// Runtime metrics for dynamic scoring
	Metrics ProviderMetrics `json:"metrics"`
	
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// GPUConstraints defines optional GPU selection constraints for a provider
type GPUConstraints struct {
	MinVRAMGB       int      `json:"min_vram_gb,omitempty"`        // Minimum VRAM required
	RequiredGPUArch string   `json:"required_gpu_arch,omitempty"`  // e.g., "ampere", "hopper", "ada"
	AllowedGPUIDs   []string `json:"allowed_gpu_ids,omitempty"`    // Specific GPU device IDs if known
	PreferredClass  string   `json:"preferred_class,omitempty"`    // e.g., "A100-80GB", "H100"
}

// ProviderMetrics tracks runtime performance metrics for a provider
type ProviderMetrics struct {
	// Request counters
	TotalRequests   int64     `json:"total_requests"`
	SuccessRequests int64     `json:"success_requests"`
	FailedRequests  int64     `json:"failed_requests"`
	LastRequestAt   time.Time `json:"last_request_at"`
	
	// Latency metrics (in milliseconds)
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	MinLatencyMs    int64   `json:"min_latency_ms"`
	MaxLatencyMs    int64   `json:"max_latency_ms"`
	LastLatencyMs   int64   `json:"last_latency_ms"`
	
	// Throughput metrics (tokens per second)
	AvgThroughput   float64 `json:"avg_throughput"`     // tokens/sec
	TotalTokens     int64   `json:"total_tokens"`
	
	// Computed metrics
	SuccessRate     float64 `json:"success_rate"`       // 0.0 - 1.0
	AvailabilityScore float64 `json:"availability_score"` // Combined health + success rate
	PerformanceScore  float64 `json:"performance_score"`  // Combined latency + throughput
	OverallScore      float64 `json:"overall_score"`      // Final weighted score
}

// VersionedEntity interface implementation for Provider
func (p *Provider) GetEntityType() models.EntityType          { return models.EntityTypeProvider }
func (p *Provider) GetSchemaVersion() models.SchemaVersion    { return p.EntityMetadata.SchemaVersion }
func (p *Provider) SetSchemaVersion(v models.SchemaVersion)   { p.EntityMetadata.SchemaVersion = v }
func (p *Provider) GetEntityMetadata() *models.EntityMetadata { return &p.EntityMetadata }
func (p *Provider) GetID() string                             { return p.ID }

// RecordSuccess records a successful provider request and updates metrics
func (p *Provider) RecordSuccess(latencyMs int64, tokens int64) {
	p.Metrics.TotalRequests++
	p.Metrics.SuccessRequests++
	p.Metrics.LastRequestAt = time.Now()
	p.Metrics.LastLatencyMs = latencyMs
	
	// Update latency stats
	if p.Metrics.MinLatencyMs == 0 || latencyMs < p.Metrics.MinLatencyMs {
		p.Metrics.MinLatencyMs = latencyMs
	}
	if latencyMs > p.Metrics.MaxLatencyMs {
		p.Metrics.MaxLatencyMs = latencyMs
	}
	
	// Rolling average latency (exponential moving average with alpha=0.2)
	if p.Metrics.AvgLatencyMs == 0 {
		p.Metrics.AvgLatencyMs = float64(latencyMs)
	} else {
		p.Metrics.AvgLatencyMs = 0.8*p.Metrics.AvgLatencyMs + 0.2*float64(latencyMs)
	}
	
	// Update token stats
	if tokens > 0 {
		p.Metrics.TotalTokens += tokens
		tokensPerSec := float64(tokens) / (float64(latencyMs) / 1000.0)
		if p.Metrics.AvgThroughput == 0 {
			p.Metrics.AvgThroughput = tokensPerSec
		} else {
			p.Metrics.AvgThroughput = 0.8*p.Metrics.AvgThroughput + 0.2*tokensPerSec
		}
	}
	
	p.updateComputedMetrics()
}

// RecordFailure records a failed provider request and updates metrics
func (p *Provider) RecordFailure(latencyMs int64) {
	p.Metrics.TotalRequests++
	p.Metrics.FailedRequests++
	p.Metrics.LastRequestAt = time.Now()
	p.Metrics.LastLatencyMs = latencyMs
	
	p.updateComputedMetrics()
}

// updateComputedMetrics recalculates derived metrics and scores
func (p *Provider) updateComputedMetrics() {
	// Success rate
	if p.Metrics.TotalRequests > 0 {
		p.Metrics.SuccessRate = float64(p.Metrics.SuccessRequests) / float64(p.Metrics.TotalRequests)
	}
	
	// Availability score (0-100): combines health status and success rate
	healthScore := 0.0
	switch p.Status {
	case "active", "healthy":
		healthScore = 100.0
	case "pending":
		healthScore = 50.0
	case "error", "failed":
		healthScore = 0.0
	default:
		healthScore = 25.0
	}
	p.Metrics.AvailabilityScore = healthScore * p.Metrics.SuccessRate
	
	// Performance score (0-100): combines latency and throughput
	// Lower latency is better (inverse relationship)
	latencyScore := 100.0
	if p.Metrics.AvgLatencyMs > 0 {
		// Score decreases as latency increases
		// 1000ms = 50pts, 5000ms = 10pts, 10000ms+ = 0pts
		latencyScore = 100.0 / (1.0 + p.Metrics.AvgLatencyMs/1000.0)
		if latencyScore > 100 {
			latencyScore = 100
		}
	}
	
	// Higher throughput is better
	throughputScore := 0.0
	if p.Metrics.AvgThroughput > 0 {
		// Score increases with throughput
		// 10 tok/s = 25pts, 50 tok/s = 50pts, 100+ tok/s = 100pts
		throughputScore = p.Metrics.AvgThroughput
		if throughputScore > 100 {
			throughputScore = 100
		}
	}
	
	// Weight latency 70%, throughput 30%
	p.Metrics.PerformanceScore = 0.7*latencyScore + 0.3*throughputScore
	
	// Overall score: 60% availability, 40% performance
	p.Metrics.OverallScore = 0.6*p.Metrics.AvailabilityScore + 0.4*p.Metrics.PerformanceScore
}

// GetScore returns the overall provider score (0-100)
func (p *Provider) GetScore() float64 {
	return p.Metrics.OverallScore
}
