package provider

import (
	"context"
	"fmt"
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
}

// Registry manages registered AI providers
type Registry struct {
	mu        sync.RWMutex
	providers map[string]*RegisteredProvider
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
	case "openai", "anthropic", "local", "custom":
		// All use OpenAI-compatible protocol
		protocol = NewOpenAIProvider(config.Endpoint, config.APIKey)
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
	case "openai", "anthropic", "local", "custom":
		protocol = NewOpenAIProvider(config.Endpoint, config.APIKey)
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

// ListActive returns registered providers with active status.
func (r *Registry) ListActive() []*RegisteredProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]*RegisteredProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		if provider != nil && provider.Config != nil && isProviderHealthy(provider.Config.Status) {
			providers = append(providers, provider)
		}
	}

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

// SendChatCompletion sends a chat completion request to a provider
func (r *Registry) SendChatCompletion(ctx context.Context, providerID string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
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

	return provider.Protocol.CreateChatCompletion(ctx, req)
}

// GetModels retrieves available models from a provider
func (r *Registry) GetModels(ctx context.Context, providerID string) ([]Model, error) {
	provider, err := r.Get(providerID)
	if err != nil {
		return nil, err
	}

	return provider.Protocol.GetModels(ctx)
}

func isProviderHealthy(status string) bool {
	switch status {
	case "healthy", "active":
		return true
	default:
		return false
	}
}
