package provider

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// ProviderConfig represents the configuration for a provider
type ProviderConfig struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Type                   string    `json:"type"` // openai, anthropic, local, etc.
	Endpoint               string    `json:"endpoint"`
	APIKey                 string    `json:"api_key,omitempty"`
	Model                  string    `json:"model"` // effective model to use
	ConfiguredModel        string    `json:"configured_model,omitempty"`
	SelectedModel          string    `json:"selected_model,omitempty"`
	SelectedGPU            string    `json:"selected_gpu,omitempty"`
	Status                 string    `json:"status,omitempty"`
	LastHeartbeatAt        time.Time `json:"last_heartbeat_at,omitempty"`
	LastHeartbeatLatencyMs int64     `json:"last_heartbeat_latency_ms,omitempty"`
	CapabilityScore        float64   `json:"capability_score,omitempty"` // Dynamic composite score from Scorer
	ContextWindow          int       `json:"context_window,omitempty"`

	// Model metadata for scoring
	ModelParamsB    float64 `json:"model_params_b,omitempty"`     // Total model parameters in billions
	CostPerMToken   float64 `json:"cost_per_mtoken,omitempty"`    // Cost per million tokens ($)
	AvgLatencyMs    float64 `json:"avg_latency_ms,omitempty"`     // Rolling average request latency
	TotalRequests   int64   `json:"total_requests,omitempty"`     // Total requests served
	SuccessRequests int64   `json:"success_requests,omitempty"`   // Successful requests
}

// MetricsCallback is called after each provider request to record metrics
type MetricsCallback func(providerID string, success bool, latencyMs int64, totalTokens int64)

// Registry manages registered AI providers
type Registry struct {
	mu              sync.RWMutex
	providers       map[string]*RegisteredProvider
	metricsCallback MetricsCallback
	rrCounter       uint64  // Round-robin counter for equal-priority providers
	scorer          *Scorer // Dynamic provider scoring
}

// RegisteredProvider wraps a provider with its configuration and protocol
type RegisteredProvider struct {
	Config   *ProviderConfig
	Protocol Protocol
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]*RegisteredProvider),
		scorer:    NewScorer(),
	}
}

// Clear removes all registered providers.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = make(map[string]*RegisteredProvider)
}

// Register registers a new provider
func (r *Registry) Register(config *ProviderConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if config.Status == "" {
		config.Status = "pending"
	}

	// Check if provider already exists
	if _, exists := r.providers[config.ID]; exists {
		return fmt.Errorf("provider %s already registered", config.ID)
	}

	// Create protocol based on provider type
	var protocol Protocol
	switch config.Type {
	case "openai", "anthropic", "local", "custom", "vllm":
		// All use OpenAI-compatible protocol
		protocol = NewOpenAIProvider(config.Endpoint, config.APIKey)
	case "ollama":
		protocol = NewOllamaProvider(config.Endpoint)
	case "mock":
		protocol = NewMockProvider()
	default:
		return fmt.Errorf("unsupported provider type: %s", config.Type)
	}

	// Register provider
	r.providers[config.ID] = &RegisteredProvider{
		Config:   config,
		Protocol: protocol,
	}

	return nil
}

// Upsert registers a provider if it doesn't exist, or replaces it if it does.
func (r *Registry) Upsert(config *ProviderConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if config.Status == "" {
		config.Status = "pending"
	}

	var protocol Protocol
	switch config.Type {
	case "openai", "anthropic", "local", "custom", "vllm":
		protocol = NewOpenAIProvider(config.Endpoint, config.APIKey)
	case "ollama":
		protocol = NewOllamaProvider(config.Endpoint)
	case "mock":
		protocol = NewMockProvider()
	default:
		return fmt.Errorf("unsupported provider type: %s", config.Type)
	}

	r.providers[config.ID] = &RegisteredProvider{Config: config, Protocol: protocol}
	return nil
}

// Unregister removes a provider from the registry
func (r *Registry) Unregister(providerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[providerID]; !exists {
		return fmt.Errorf("provider %s not found", providerID)
	}

	delete(r.providers, providerID)
	return nil
}

// Get retrieves a registered provider
func (r *Registry) Get(providerID string) (*RegisteredProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[providerID]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerID)
	}

	return provider, nil
}

// List returns all registered providers
func (r *Registry) List() []*RegisteredProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]*RegisteredProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}

// ListActive returns registered providers with active status, sorted by
// dynamic capability score (highest first). Scoring prioritizes:
// 1. Model size (larger models preferred)
// 2. Round-trip time (lower heartbeat latency preferred)
// 3. Request latency (lower average response time preferred)
// 4. Cost (lower cost preferred, currently $0 for all)
// Providers with equal scores are round-robined so all get work.
func (r *Registry) ListActive() []*RegisteredProvider {
	r.mu.RLock()
	counter := r.rrCounter
	r.mu.RUnlock()

	r.mu.RLock()
	providers := make([]*RegisteredProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		if provider != nil && provider.Config != nil && isProviderHealthy(provider.Config.Status) {
			// Update dynamic score from scorer
			if r.scorer != nil {
				if score, ok := r.scorer.GetScore(provider.Config.ID); ok {
					provider.Config.CapabilityScore = score.CompositeScore
				}
			}
			providers = append(providers, provider)
		}
	}
	r.mu.RUnlock()

	if len(providers) <= 1 {
		return providers
	}

	// Sort by capability score descending
	sort.SliceStable(providers, func(i, j int) bool {
		si := providers[i].Config.CapabilityScore
		sj := providers[j].Config.CapabilityScore
		return si > sj
	})

	// Find the group of providers with equal top score and rotate them
	topScore := providers[0].Config.CapabilityScore
	equalCount := 0
	const scoreTolerance = 0.01 // Consider scores within 0.01 as equal
	for _, p := range providers {
		if p.Config.CapabilityScore >= topScore-scoreTolerance {
			equalCount++
		} else {
			break
		}
	}

	if equalCount > 1 {
		// Round-robin within the equal-score group
		rotation := int(counter) % equalCount
		rotated := make([]*RegisteredProvider, 0, len(providers))
		rotated = append(rotated, providers[rotation:equalCount]...)
		rotated = append(rotated, providers[:rotation]...)
		rotated = append(rotated, providers[equalCount:]...)
		providers = rotated
	}

	// Increment counter for next call
	r.mu.Lock()
	r.rrCounter++
	r.mu.Unlock()

	return providers
}

// IsActive returns true if the provider is registered and active.
func (r *Registry) IsActive(providerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[providerID]
	if !exists || provider == nil || provider.Config == nil {
		return false
	}
	return isProviderHealthy(provider.Config.Status)
}

// SetMetricsCallback sets the callback function for recording metrics
func (r *Registry) SetMetricsCallback(callback MetricsCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metricsCallback = callback
}

// SendChatCompletionStream sends a streaming chat completion request to a provider
func (r *Registry) SendChatCompletionStream(ctx context.Context, providerID string, req *ChatCompletionRequest, handler StreamHandler) error {
	start := time.Now()

	// Get provider
	registered, err := r.Get(providerID)
	if err != nil {
		return err
	}

	// Check if provider supports streaming
	streamProvider, ok := registered.Protocol.(StreamingProtocol)
	if !ok {
		return fmt.Errorf("provider %s does not support streaming", providerID)
	}

	// Send streaming request
	err = streamProvider.CreateChatCompletionStream(ctx, req, handler)

	// Record metrics
	latencyMs := time.Since(start).Milliseconds()
	if r.metricsCallback != nil {
		r.metricsCallback(providerID, err == nil, latencyMs, 0)
	}

	return err
}

// SendChatCompletion sends a chat completion request to a provider
func (r *Registry) SendChatCompletion(ctx context.Context, providerID string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	startTime := time.Now()

	provider, err := r.Get(providerID)
	if err != nil {
		return nil, err
	}
	if provider.Config != nil && !isProviderHealthy(provider.Config.Status) {
		return nil, fmt.Errorf("provider %s is disabled", providerID)
	}

	// Use default model if not specified
	if req.Model == "" {
		req.Model = provider.Config.Model
	}

	// Make the request
	resp, err := provider.Protocol.CreateChatCompletion(ctx, req)

	// If model not found (404), the vLLM server may have restarted with a
	// different model. Rediscover available models and retry once.
	if err != nil && (strings.Contains(err.Error(), "status code 404") || strings.Contains(err.Error(), "not found")) {
		log.Printf("[Registry] Model %q returned 404 on provider %s — rediscovering models", req.Model, providerID)
		models, modelErr := provider.Protocol.GetModels(ctx)
		if modelErr == nil && len(models) > 0 {
			newModel := models[0].ID
			log.Printf("[Registry] Provider %s model changed: %q → %q", providerID, req.Model, newModel)
			r.mu.Lock()
			if p, ok := r.providers[providerID]; ok && p.Config != nil {
				p.Config.Model = newModel
				p.Config.SelectedModel = newModel
			}
			r.mu.Unlock()
			req.Model = newModel
			resp, err = provider.Protocol.CreateChatCompletion(ctx, req)
		}
	}

	// Record metrics
	latencyMs := time.Since(startTime).Milliseconds()
	success := err == nil
	totalTokens := int64(0)
	if resp != nil {
		totalTokens = int64(resp.Usage.TotalTokens)
	}

	// Update dynamic scoring metrics
	r.RecordRequestMetrics(providerID, latencyMs, success)

	// Call metrics callback if registered
	r.mu.RLock()
	callback := r.metricsCallback
	r.mu.RUnlock()

	if callback != nil {
		callback(providerID, success, latencyMs, totalTokens)
	}

	return resp, err
}

// GetModels retrieves available models from a provider
func (r *Registry) GetModels(ctx context.Context, providerID string) ([]Model, error) {
	provider, err := r.Get(providerID)
	if err != nil {
		return nil, err
	}

	return provider.Protocol.GetModels(ctx)
}

// GetScorer returns the registry's dynamic scorer.
func (r *Registry) GetScorer() *Scorer {
	return r.scorer
}

// UpdateProviderScore updates the dynamic score for a provider.
// This should be called after heartbeat or when metrics change.
func (r *Registry) UpdateProviderScore(providerID string, modelParamsB float64, costPerMToken float64) {
	r.mu.RLock()
	provider, exists := r.providers[providerID]
	r.mu.RUnlock()

	if !exists || provider == nil || provider.Config == nil {
		return
	}

	cfg := provider.Config
	cfg.ModelParamsB = modelParamsB
	cfg.CostPerMToken = costPerMToken

	if r.scorer != nil {
		score := r.scorer.UpdateProviderMetrics(
			providerID,
			modelParamsB,
			cfg.LastHeartbeatLatencyMs,
			cfg.AvgLatencyMs,
			costPerMToken,
		)
		cfg.CapabilityScore = score.CompositeScore
	}
}

// RecordRequestMetrics records request latency and updates the provider's rolling average.
// Called by SendChatCompletion via the metrics callback.
func (r *Registry) RecordRequestMetrics(providerID string, latencyMs int64, success bool) {
	r.mu.Lock()
	provider, exists := r.providers[providerID]
	if !exists || provider == nil || provider.Config == nil {
		r.mu.Unlock()
		return
	}

	cfg := provider.Config
	cfg.TotalRequests++
	if success {
		cfg.SuccessRequests++
	}

	// Update rolling average latency (exponential moving average, alpha=0.2)
	if cfg.AvgLatencyMs == 0 {
		cfg.AvgLatencyMs = float64(latencyMs)
	} else {
		cfg.AvgLatencyMs = 0.8*cfg.AvgLatencyMs + 0.2*float64(latencyMs)
	}
	r.mu.Unlock()

	// Update the scorer with new metrics
	if r.scorer != nil {
		score := r.scorer.UpdateProviderMetrics(
			providerID,
			cfg.ModelParamsB,
			cfg.LastHeartbeatLatencyMs,
			cfg.AvgLatencyMs,
			cfg.CostPerMToken,
		)

		r.mu.Lock()
		cfg.CapabilityScore = score.CompositeScore
		r.mu.Unlock()
	}
}

// UpdateHeartbeatLatency updates the heartbeat latency for a provider and recalculates score.
func (r *Registry) UpdateHeartbeatLatency(providerID string, latencyMs int64) {
	r.mu.Lock()
	provider, exists := r.providers[providerID]
	if !exists || provider == nil || provider.Config == nil {
		r.mu.Unlock()
		return
	}

	cfg := provider.Config
	cfg.LastHeartbeatLatencyMs = latencyMs
	cfg.LastHeartbeatAt = time.Now()
	r.mu.Unlock()

	// Update the scorer with new metrics
	if r.scorer != nil {
		score := r.scorer.UpdateProviderMetrics(
			providerID,
			cfg.ModelParamsB,
			latencyMs,
			cfg.AvgLatencyMs,
			cfg.CostPerMToken,
		)

		r.mu.Lock()
		cfg.CapabilityScore = score.CompositeScore
		r.mu.Unlock()
	}
}

// SetScoringWeights updates the scoring weights used for provider prioritization.
func (r *Registry) SetScoringWeights(weights ScoringWeights) {
	if r.scorer != nil {
		r.scorer.SetWeights(weights)
	}
}

// GetScoringWeights returns the current scoring weights.
func (r *Registry) GetScoringWeights() ScoringWeights {
	if r.scorer != nil {
		return r.scorer.GetWeights()
	}
	return DefaultWeights()
}

// ListActiveForComplexity returns registered providers sorted by suitability for a complexity level.
// Providers that match the required model tier for the complexity are ranked first.
func (r *Registry) ListActiveForComplexity(complexity ComplexityLevel) []*RegisteredProvider {
	r.mu.RLock()
	providers := make([]*RegisteredProvider, 0, len(r.providers))
	providerIDs := make([]string, 0, len(r.providers))
	providerMap := make(map[string]*RegisteredProvider)

	for _, provider := range r.providers {
		if provider != nil && provider.Config != nil && isProviderHealthy(provider.Config.Status) {
			providers = append(providers, provider)
			providerIDs = append(providerIDs, provider.Config.ID)
			providerMap[provider.Config.ID] = provider
		}
	}
	r.mu.RUnlock()

	if len(providers) <= 1 {
		return providers
	}

	// Use the scorer to rank providers for this complexity
	if r.scorer != nil {
		rankedIDs := r.scorer.RankProvidersForComplexity(providerIDs, complexity)
		result := make([]*RegisteredProvider, 0, len(rankedIDs))
		for _, id := range rankedIDs {
			if p, ok := providerMap[id]; ok {
				// Update dynamic score from scorer
				if score, ok := r.scorer.GetScore(id); ok {
					p.Config.CapabilityScore = score.CompositeScore
				}
				result = append(result, p)
			}
		}
		return result
	}

	return providers
}

// SelectProviderForComplexity selects the best provider for a given complexity level.
// Returns the provider, its score, and whether a suitable provider was found.
func (r *Registry) SelectProviderForComplexity(complexity ComplexityLevel) (*RegisteredProvider, float64, bool) {
	providers := r.ListActiveForComplexity(complexity)
	if len(providers) == 0 {
		return nil, 0, false
	}
	best := providers[0]
	return best, best.Config.CapabilityScore, true
}

// GetComplexityEstimator returns a complexity estimator for analyzing tasks.
func (r *Registry) GetComplexityEstimator() *ComplexityEstimator {
	return NewComplexityEstimator()
}

func isProviderHealthy(status string) bool {
	return status == "healthy" || status == "active"
}
