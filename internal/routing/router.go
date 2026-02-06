package routing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	internalmodels "github.com/jordanhubbard/loom/internal/models"
)

// RoutingPolicy defines how providers should be selected
type RoutingPolicy string

const (
	PolicyMinimizeCost    RoutingPolicy = "minimize_cost"
	PolicyMinimizeLatency RoutingPolicy = "minimize_latency"
	PolicyMaximizeQuality RoutingPolicy = "maximize_quality"
	PolicyBalanced        RoutingPolicy = "balanced"
)

// ProviderRequirements defines what capabilities a provider must have
type ProviderRequirements struct {
	MinContextWindow int      // Minimum context window size
	RequiresFunction bool     // Requires function calling support
	RequiresVision   bool     // Requires vision/multimodal support
	MaxCostPerMToken float64  // Maximum cost per million tokens (0 = no limit)
	MaxLatencyMs     int64    // Maximum acceptable latency (0 = no limit)
	RequiredTags     []string // Provider must have these tags
}

// Router selects optimal providers based on policies and requirements
type Router struct {
	policy RoutingPolicy
}

// NewRouter creates a new provider router
func NewRouter(policy RoutingPolicy) *Router {
	return &Router{policy: policy}
}

// SelectProvider chooses the best provider from available options
func (r *Router) SelectProvider(
	ctx context.Context,
	providers []*internalmodels.Provider,
	requirements *ProviderRequirements,
) (*internalmodels.Provider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Filter providers that meet requirements
	candidates := r.filterByRequirements(providers, requirements)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no providers meet requirements")
	}

	// Score candidates based on policy
	scored := r.scoreCandidates(candidates, requirements)
	if len(scored) == 0 {
		return nil, fmt.Errorf("no providers available after scoring")
	}

	// Return highest scored provider
	return scored[0].provider, nil
}

// SelectProviderWithFailover attempts primary selection, falls back on failure
func (r *Router) SelectProviderWithFailover(
	ctx context.Context,
	providers []*internalmodels.Provider,
	requirements *ProviderRequirements,
	excludeIDs []string,
) (*internalmodels.Provider, error) {
	// Filter out excluded providers (previously failed)
	filtered := make([]*internalmodels.Provider, 0, len(providers))
	for _, p := range providers {
		excluded := false
		for _, id := range excludeIDs {
			if p.ID == id {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, p)
		}
	}

	return r.SelectProvider(ctx, filtered, requirements)
}

// filterByRequirements removes providers that don't meet requirements
func (r *Router) filterByRequirements(
	providers []*internalmodels.Provider,
	requirements *ProviderRequirements,
) []*internalmodels.Provider {
	if requirements == nil {
		// No requirements, all healthy providers are candidates
		return filterHealthy(providers)
	}

	candidates := make([]*internalmodels.Provider, 0, len(providers))
	for _, p := range providers {
		if !isHealthy(p) {
			continue
		}

		// Check cost constraint
		if requirements.MaxCostPerMToken > 0 && p.CostPerMToken > requirements.MaxCostPerMToken {
			continue
		}

		// Check latency constraint
		if requirements.MaxLatencyMs > 0 && p.Metrics.AvgLatencyMs > float64(requirements.MaxLatencyMs) {
			continue
		}

		// Check context window
		if requirements.MinContextWindow > 0 && p.ContextWindow < requirements.MinContextWindow {
			continue
		}

		// Check capabilities
		if requirements.RequiresFunction && !p.SupportsFunction {
			continue
		}
		if requirements.RequiresVision && !p.SupportsVision {
			continue
		}

		// Check required tags
		if len(requirements.RequiredTags) > 0 {
			hasAll := true
			for _, reqTag := range requirements.RequiredTags {
				found := false
				for _, tag := range p.Tags {
					if tag == reqTag {
						found = true
						break
					}
				}
				if !found {
					hasAll = false
					break
				}
			}
			if !hasAll {
				continue
			}
		}

		candidates = append(candidates, p)
	}

	return candidates
}

type scoredProvider struct {
	provider *internalmodels.Provider
	score    float64
}

// scoreCandidates ranks providers based on routing policy
func (r *Router) scoreCandidates(
	providers []*internalmodels.Provider,
	requirements *ProviderRequirements,
) []scoredProvider {
	scored := make([]scoredProvider, 0, len(providers))

	for _, p := range providers {
		var score float64

		switch r.policy {
		case PolicyMinimizeCost:
			score = r.scoreByCost(p)
		case PolicyMinimizeLatency:
			score = r.scoreByLatency(p)
		case PolicyMaximizeQuality:
			score = r.scoreByQuality(p)
		case PolicyBalanced:
			score = r.scoreBalanced(p)
		default:
			score = r.scoreBalanced(p)
		}

		scored = append(scored, scoredProvider{
			provider: p,
			score:    score,
		})
	}

	// Sort by score descending (highest first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored
}

// scoreByCost prioritizes lowest cost providers
func (r *Router) scoreByCost(p *internalmodels.Provider) float64 {
	// Lower cost = higher score
	// Score range: 0-100, inverted by cost
	if p.CostPerMToken <= 0 {
		return 100.0 // Free/unknown cost gets highest score
	}

	// Typical costs: $0.50-$30 per million tokens
	// Map to 0-100 scale, inverse relationship
	// $0.50 = 100, $10 = 50, $30 = 10
	baseCost := 0.5
	maxCost := 30.0
	normalizedCost := math.Min((p.CostPerMToken-baseCost)/(maxCost-baseCost), 1.0)
	score := 100.0 * (1.0 - normalizedCost)

	// Bonus for high availability
	score += p.Metrics.AvailabilityScore * 0.2

	return score
}

// scoreByLatency prioritizes lowest latency providers
func (r *Router) scoreByLatency(p *internalmodels.Provider) float64 {
	// Lower latency = higher score
	// Score based on performance metrics already computed
	score := p.Metrics.PerformanceScore

	// Bonus for high availability
	score += p.Metrics.AvailabilityScore * 0.2

	return score
}

// scoreByQuality prioritizes highest quality/capability providers
func (r *Router) scoreByQuality(p *internalmodels.Provider) float64 {
	score := 0.0

	// Base score from context window size (larger = better)
	// 4k = 20, 8k = 40, 32k = 80, 128k+ = 100
	contextScore := math.Min(float64(p.ContextWindow)/128000.0*100.0, 100.0)
	score += contextScore * 0.4

	// Capability bonuses
	if p.SupportsFunction {
		score += 15.0
	}
	if p.SupportsVision {
		score += 15.0
	}
	if p.SupportsStreaming {
		score += 10.0
	}

	// Provider type bonus (some types generally higher quality)
	switch p.Type {
	case "openai":
		score += 10.0
	case "anthropic":
		score += 10.0
	case "gemini":
		score += 8.0
	}

	// Success rate bonus
	score += p.Metrics.SuccessRate * 10.0

	return score
}

// scoreBalanced balances cost, latency, and quality
func (r *Router) scoreBalanced(p *internalmodels.Provider) float64 {
	costScore := r.scoreByCost(p)
	latencyScore := r.scoreByLatency(p)
	qualityScore := r.scoreByQuality(p)

	// Weighted average: 30% cost, 30% latency, 40% quality
	return costScore*0.3 + latencyScore*0.3 + qualityScore*0.4
}

// filterHealthy returns only healthy providers
func filterHealthy(providers []*internalmodels.Provider) []*internalmodels.Provider {
	healthy := make([]*internalmodels.Provider, 0, len(providers))
	for _, p := range providers {
		if isHealthy(p) {
			healthy = append(healthy, p)
		}
	}
	return healthy
}

// isHealthy checks if a provider is healthy and available
func isHealthy(p *internalmodels.Provider) bool {
	if p == nil {
		return false
	}

	// Check status
	if p.Status != "active" && p.Status != "healthy" {
		return false
	}

	// Check recent heartbeat (within last 5 minutes)
	if !p.LastHeartbeatAt.IsZero() {
		if time.Since(p.LastHeartbeatAt) > 5*time.Minute {
			return false
		}
	}

	// Check success rate (must be > 50%)
	if p.Metrics.TotalRequests > 10 && p.Metrics.SuccessRate < 0.5 {
		return false
	}

	return true
}
