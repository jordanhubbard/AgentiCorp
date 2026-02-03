package cache

import (
	"context"
	"testing"
	"time"
)

func TestInvalidateByProvider(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	// Add entries for different providers
	providers := []string{"provider-1", "provider-2", "provider-3"}
	for _, provider := range providers {
		for j := 0; j < 3; j++ {
			key := generateTestKey(provider, j)
			metadata := map[string]interface{}{
				"provider_id": provider,
				"model_name":  "gpt-4",
			}
			if err := c.Set(ctx, key, map[string]interface{}{"result": "test"}, 1*time.Hour, metadata); err != nil {
				t.Fatalf("Failed to set cache: %v", err)
			}
		}
	}

	// Should have 9 entries total
	stats := c.GetStats(ctx)
	if stats.TotalEntries != 9 {
		t.Errorf("Expected 9 entries, got %d", stats.TotalEntries)
	}

	// Invalidate provider-1
	removed := c.InvalidateByProvider(ctx, "provider-1")
	if removed != 3 {
		t.Errorf("Expected 3 entries removed, got %d", removed)
	}

	// Should have 6 entries remaining
	stats = c.GetStats(ctx)
	if stats.TotalEntries != 6 {
		t.Errorf("Expected 6 entries remaining, got %d", stats.TotalEntries)
	}

	// Verify provider-1 entries are gone
	_, found := c.Get(ctx, generateTestKey("provider-1", 0))
	if found {
		t.Error("Expected provider-1 entry to be invalidated")
	}

	// Verify other providers still exist
	_, found = c.Get(ctx, generateTestKey("provider-2", 0))
	if !found {
		t.Error("Expected provider-2 entry to still exist")
	}
}

func TestInvalidateByModel(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	// Add entries for different models
	models := []string{"gpt-4", "gpt-3.5", "claude-3"}
	for _, model := range models {
		for j := 0; j < 2; j++ {
			key := generateTestKey(model, j)
			metadata := map[string]interface{}{
				"provider_id": "provider-1",
				"model_name":  model,
			}
			if err := c.Set(ctx, key, map[string]interface{}{"result": "test"}, 1*time.Hour, metadata); err != nil {
				t.Fatalf("Failed to set cache: %v", err)
			}
		}
	}

	// Should have 6 entries total
	stats := c.GetStats(ctx)
	if stats.TotalEntries != 6 {
		t.Errorf("Expected 6 entries, got %d", stats.TotalEntries)
	}

	// Invalidate gpt-4
	removed := c.InvalidateByModel(ctx, "gpt-4")
	if removed != 2 {
		t.Errorf("Expected 2 entries removed, got %d", removed)
	}

	// Should have 4 entries remaining
	stats = c.GetStats(ctx)
	if stats.TotalEntries != 4 {
		t.Errorf("Expected 4 entries remaining, got %d", stats.TotalEntries)
	}
}

func TestInvalidateByAge(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	// Add old entry
	if err := c.Set(ctx, "old-key", map[string]interface{}{"result": "old"}, 1*time.Hour, nil); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Manually set cached time to 2 hours ago
	c.mu.Lock()
	if entry, ok := c.entries["old-key"]; ok {
		entry.CachedAt = time.Now().Add(-2 * time.Hour)
	}
	c.mu.Unlock()

	// Add recent entry
	if err := c.Set(ctx, "new-key", map[string]interface{}{"result": "new"}, 1*time.Hour, nil); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Should have 2 entries
	stats := c.GetStats(ctx)
	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.TotalEntries)
	}

	// Invalidate entries older than 1 hour
	removed := c.InvalidateByAge(ctx, 1*time.Hour)
	if removed != 1 {
		t.Errorf("Expected 1 entry removed, got %d", removed)
	}

	// Old entry should be gone
	_, found := c.Get(ctx, "old-key")
	if found {
		t.Error("Expected old entry to be invalidated")
	}

	// New entry should remain
	_, found = c.Get(ctx, "new-key")
	if !found {
		t.Error("Expected new entry to remain")
	}
}

func TestInvalidateByPattern(t *testing.T) {
	c := New(DefaultConfig())
	ctx := context.Background()

	// Add entries with different key patterns
	_ = c.Set(ctx, "prefix-test-1", map[string]interface{}{"result": "1"}, 1*time.Hour, nil)
	_ = c.Set(ctx, "prefix-test-2", map[string]interface{}{"result": "2"}, 1*time.Hour, nil)
	_ = c.Set(ctx, "other-key-1", map[string]interface{}{"result": "3"}, 1*time.Hour, nil)
	_ = c.Set(ctx, "prefix-demo-1", map[string]interface{}{"result": "4"}, 1*time.Hour, nil)

	// Should have 4 entries
	stats := c.GetStats(ctx)
	if stats.TotalEntries != 4 {
		t.Errorf("Expected 4 entries, got %d", stats.TotalEntries)
	}

	// Invalidate all "prefix-test" entries
	removed := c.InvalidateByPattern(ctx, "prefix-test")
	if removed != 2 {
		t.Errorf("Expected 2 entries removed, got %d", removed)
	}

	// Should have 2 entries remaining
	stats = c.GetStats(ctx)
	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries remaining, got %d", stats.TotalEntries)
	}

	// Verify prefix-test entries are gone
	_, found := c.Get(ctx, "prefix-test-1")
	if found {
		t.Error("Expected prefix-test-1 to be invalidated")
	}

	// Verify other entries remain
	_, found = c.Get(ctx, "other-key-1")
	if !found {
		t.Error("Expected other-key-1 to remain")
	}

	_, found = c.Get(ctx, "prefix-demo-1")
	if !found {
		t.Error("Expected prefix-demo-1 to remain")
	}
}

// Helper function for tests
func generateTestKey(prefix string, suffix int) string {
	return prefix + "-" + string(rune('0'+suffix))
}
