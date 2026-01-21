package agenticorp

import (
	"github.com/jordanhubbard/agenticorp/internal/temporal/eventbus"
)

// setupProviderMetrics configures the provider registry to record metrics
func (a *AgentiCorp) setupProviderMetrics() {
	// Set metrics callback on provider registry
	a.providerRegistry.SetMetricsCallback(func(providerID string, success bool, latencyMs int64, totalTokens int64) {
		// Skip if no database
		if a.database == nil {
			return
		}
		
		// Load provider from database
		provider, err := a.database.GetProvider(providerID)
		if err != nil {
			return
		}
		
		// Record success or failure
		if success {
			provider.RecordSuccess(latencyMs, totalTokens)
		} else {
			provider.RecordFailure(latencyMs)
		}
		
		// Persist updated metrics
		_ = a.database.UpsertProvider(provider)
		
		// Emit event for real-time updates
		if a.eventBus != nil {
			a.eventBus.Publish(&eventbus.Event{
				Type: "provider.updated",
				Data: map[string]interface{}{
					"provider_id":    providerID,
					"success":        success,
					"latency_ms":     latencyMs,
					"total_tokens":   totalTokens,
					"overall_score":  provider.GetScore(),
				},
			})
		}
	})
}
