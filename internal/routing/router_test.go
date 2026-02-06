package routing

import (
	"context"
	"testing"
	"time"

	internalmodels "github.com/jordanhubbard/loom/internal/models"
)

func TestSelectProvider_MinimizeCost(t *testing.T) {
	router := NewRouter(PolicyMinimizeCost)

	providers := []*internalmodels.Provider{
		{
			ID:              "expensive",
			Name:            "Expensive Provider",
			Status:          "active",
			CostPerMToken:   30.0,
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				AvailabilityScore: 100.0,
				SuccessRate:       1.0,
			},
		},
		{
			ID:              "cheap",
			Name:            "Cheap Provider",
			Status:          "active",
			CostPerMToken:   0.5,
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				AvailabilityScore: 100.0,
				SuccessRate:       1.0,
			},
		},
		{
			ID:              "moderate",
			Name:            "Moderate Provider",
			Status:          "active",
			CostPerMToken:   10.0,
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				AvailabilityScore: 100.0,
				SuccessRate:       1.0,
			},
		},
	}

	selected, err := router.SelectProvider(context.Background(), providers, nil)
	if err != nil {
		t.Fatalf("SelectProvider failed: %v", err)
	}

	if selected.ID != "cheap" {
		t.Errorf("Expected cheapest provider, got %s", selected.ID)
	}
}

func TestSelectProvider_MinimizeLatency(t *testing.T) {
	router := NewRouter(PolicyMinimizeLatency)

	providers := []*internalmodels.Provider{
		{
			ID:              "slow",
			Name:            "Slow Provider",
			Status:          "active",
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				AvgLatencyMs:      5000.0,
				PerformanceScore:  20.0,
				AvailabilityScore: 100.0,
				SuccessRate:       1.0,
			},
		},
		{
			ID:              "fast",
			Name:            "Fast Provider",
			Status:          "active",
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				AvgLatencyMs:      100.0,
				PerformanceScore:  90.0,
				AvailabilityScore: 100.0,
				SuccessRate:       1.0,
			},
		},
	}

	selected, err := router.SelectProvider(context.Background(), providers, nil)
	if err != nil {
		t.Fatalf("SelectProvider failed: %v", err)
	}

	if selected.ID != "fast" {
		t.Errorf("Expected fastest provider, got %s", selected.ID)
	}
}

func TestSelectProvider_MaximizeQuality(t *testing.T) {
	router := NewRouter(PolicyMaximizeQuality)

	providers := []*internalmodels.Provider{
		{
			ID:               "basic",
			Name:             "Basic Provider",
			Status:           "active",
			ContextWindow:    4096,
			SupportsFunction: false,
			SupportsVision:   false,
			LastHeartbeatAt:  time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				SuccessRate: 1.0,
			},
		},
		{
			ID:                "advanced",
			Name:              "Advanced Provider",
			Type:              "openai",
			Status:            "active",
			ContextWindow:     128000,
			SupportsFunction:  true,
			SupportsVision:    true,
			SupportsStreaming: true,
			LastHeartbeatAt:   time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				SuccessRate: 1.0,
			},
		},
	}

	selected, err := router.SelectProvider(context.Background(), providers, nil)
	if err != nil {
		t.Fatalf("SelectProvider failed: %v", err)
	}

	if selected.ID != "advanced" {
		t.Errorf("Expected advanced provider, got %s", selected.ID)
	}
}

func TestSelectProvider_WithRequirements(t *testing.T) {
	router := NewRouter(PolicyBalanced)

	providers := []*internalmodels.Provider{
		{
			ID:               "no-functions",
			Name:             "No Functions",
			Status:           "active",
			ContextWindow:    4096,
			SupportsFunction: false,
			LastHeartbeatAt:  time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				SuccessRate: 1.0,
			},
		},
		{
			ID:               "has-functions",
			Name:             "Has Functions",
			Status:           "active",
			ContextWindow:    8192,
			SupportsFunction: true,
			LastHeartbeatAt:  time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				SuccessRate: 1.0,
			},
		},
	}

	requirements := &ProviderRequirements{
		RequiresFunction: true,
		MinContextWindow: 8000,
	}

	selected, err := router.SelectProvider(context.Background(), providers, requirements)
	if err != nil {
		t.Fatalf("SelectProvider failed: %v", err)
	}

	if selected.ID != "has-functions" {
		t.Errorf("Expected provider with functions, got %s", selected.ID)
	}
}

func TestSelectProvider_NoHealthyProviders(t *testing.T) {
	router := NewRouter(PolicyBalanced)

	providers := []*internalmodels.Provider{
		{
			ID:              "unhealthy",
			Name:            "Unhealthy Provider",
			Status:          "error",
			LastHeartbeatAt: time.Now().Add(-10 * time.Minute),
		},
	}

	_, err := router.SelectProvider(context.Background(), providers, nil)
	if err == nil {
		t.Error("Expected error for no healthy providers")
	}
}

func TestSelectProviderWithFailover(t *testing.T) {
	router := NewRouter(PolicyBalanced)

	providers := []*internalmodels.Provider{
		{
			ID:              "primary",
			Name:            "Primary",
			Status:          "active",
			CostPerMToken:   1.0,
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				AvailabilityScore: 100.0,
				SuccessRate:       1.0,
			},
		},
		{
			ID:              "backup",
			Name:            "Backup",
			Status:          "active",
			CostPerMToken:   5.0,
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				AvailabilityScore: 100.0,
				SuccessRate:       1.0,
			},
		},
	}

	// Exclude primary, should get backup
	selected, err := router.SelectProviderWithFailover(
		context.Background(),
		providers,
		nil,
		[]string{"primary"},
	)
	if err != nil {
		t.Fatalf("SelectProviderWithFailover failed: %v", err)
	}

	if selected.ID != "backup" {
		t.Errorf("Expected backup provider, got %s", selected.ID)
	}
}

func TestFilterByRequirements_CostConstraint(t *testing.T) {
	router := NewRouter(PolicyBalanced)

	providers := []*internalmodels.Provider{
		{
			ID:              "expensive",
			Status:          "active",
			CostPerMToken:   30.0,
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				SuccessRate: 1.0,
			},
		},
		{
			ID:              "affordable",
			Status:          "active",
			CostPerMToken:   5.0,
			LastHeartbeatAt: time.Now(),
			Metrics: internalmodels.ProviderMetrics{
				SuccessRate: 1.0,
			},
		},
	}

	requirements := &ProviderRequirements{
		MaxCostPerMToken: 10.0,
	}

	candidates := router.filterByRequirements(providers, requirements)
	if len(candidates) != 1 {
		t.Fatalf("Expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].ID != "affordable" {
		t.Errorf("Expected affordable provider, got %s", candidates[0].ID)
	}
}

func TestIsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		provider *internalmodels.Provider
		want     bool
	}{
		{
			name:     "nil provider",
			provider: nil,
			want:     false,
		},
		{
			name: "active with recent heartbeat",
			provider: &internalmodels.Provider{
				Status:          "active",
				LastHeartbeatAt: time.Now(),
				Metrics: internalmodels.ProviderMetrics{
					TotalRequests: 100,
					SuccessRate:   0.95,
				},
			},
			want: true,
		},
		{
			name: "inactive status",
			provider: &internalmodels.Provider{
				Status:          "inactive",
				LastHeartbeatAt: time.Now(),
			},
			want: false,
		},
		{
			name: "stale heartbeat",
			provider: &internalmodels.Provider{
				Status:          "active",
				LastHeartbeatAt: time.Now().Add(-10 * time.Minute),
			},
			want: false,
		},
		{
			name: "low success rate",
			provider: &internalmodels.Provider{
				Status:          "active",
				LastHeartbeatAt: time.Now(),
				Metrics: internalmodels.ProviderMetrics{
					TotalRequests: 100,
					SuccessRate:   0.3,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHealthy(tt.provider)
			if got != tt.want {
				t.Errorf("isHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}
