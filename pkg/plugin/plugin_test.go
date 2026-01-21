package plugin

import (
	"context"
	"testing"
	"time"
)

// TestMetadata tests basic metadata creation
func TestMetadata(t *testing.T) {
	metadata := &Metadata{
		Name:             "Test Plugin",
		Version:          "1.0.0",
		PluginAPIVersion: PluginVersion,
		ProviderType:     "test-provider",
		Description:      "A test plugin",
		Author:           "Test Author",
		License:          "MIT",
		Capabilities: Capabilities{
			Streaming:       true,
			FunctionCalling: false,
			Vision:          false,
		},
	}

	if metadata.Name != "Test Plugin" {
		t.Errorf("Expected name 'Test Plugin', got '%s'", metadata.Name)
	}

	if metadata.PluginAPIVersion != PluginVersion {
		t.Errorf("Expected plugin API version '%s', got '%s'", PluginVersion, metadata.PluginAPIVersion)
	}

	if !metadata.Capabilities.Streaming {
		t.Error("Expected streaming capability to be true")
	}
}

// TestBasePlugin tests the BasePlugin implementation
func TestBasePlugin(t *testing.T) {
	metadata := &Metadata{
		Name:    "Base Test",
		Version: "1.0.0",
	}

	bp := NewBasePlugin(metadata)

	// Test GetMetadata
	if bp.GetMetadata().Name != "Base Test" {
		t.Errorf("Expected name 'Base Test', got '%s'", bp.GetMetadata().Name)
	}

	// Test Initialize
	config := map[string]interface{}{
		"api_key":  "test-key",
		"timeout":  30,
		"enabled":  true,
		"max_rate": 100.5,
	}

	ctx := context.Background()
	err := bp.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test GetConfigString
	key, ok := bp.GetConfigString("api_key")
	if !ok || key != "test-key" {
		t.Errorf("Expected api_key='test-key', got '%s' (ok=%v)", key, ok)
	}

	// Test GetConfigInt
	timeout, ok := bp.GetConfigInt("timeout")
	if !ok || timeout != 30 {
		t.Errorf("Expected timeout=30, got %d (ok=%v)", timeout, ok)
	}

	// Test GetConfigBool
	enabled, ok := bp.GetConfigBool("enabled")
	if !ok || !enabled {
		t.Errorf("Expected enabled=true, got %v (ok=%v)", enabled, ok)
	}

	// Test GetConfigFloat
	maxRate, ok := bp.GetConfigFloat("max_rate")
	if !ok || maxRate != 100.5 {
		t.Errorf("Expected max_rate=100.5, got %v (ok=%v)", maxRate, ok)
	}

	// Test missing config
	_, ok = bp.GetConfigString("nonexistent")
	if ok {
		t.Error("Expected nonexistent key to return ok=false")
	}

	// Test Cleanup
	err = bp.Cleanup(ctx)
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
}

// TestValidateConfig tests configuration validation
func TestValidateConfig(t *testing.T) {
	schema := []ConfigField{
		{
			Name:        "api_key",
			Type:        "string",
			Required:    true,
			Description: "API Key",
			Sensitive:   true,
		},
		{
			Name:        "timeout",
			Type:        "int",
			Required:    false,
			Description: "Timeout in seconds",
			Default:     30,
			Validation: &ValidationRule{
				Min: floatPtr(1),
				Max: floatPtr(300),
			},
		},
		{
			Name:        "enabled",
			Type:        "bool",
			Required:    false,
			Default:     true,
			Description: "Enable plugin",
		},
	}

	// Valid config
	config1 := map[string]interface{}{
		"api_key": "test-key",
		"timeout": 60,
		"enabled": true,
	}
	err := ValidateConfig(config1, schema)
	if err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// Missing required field
	config2 := map[string]interface{}{
		"timeout": 60,
	}
	err = ValidateConfig(config2, schema)
	if err == nil {
		t.Error("Expected error for missing required field")
	}

	// Wrong type
	config3 := map[string]interface{}{
		"api_key": "test-key",
		"timeout": "not-an-int",
	}
	err = ValidateConfig(config3, schema)
	if err == nil {
		t.Error("Expected error for wrong type")
	}

	// Out of range
	config4 := map[string]interface{}{
		"api_key": "test-key",
		"timeout": 500, // Exceeds max of 300
	}
	err = ValidateConfig(config4, schema)
	if err == nil {
		t.Error("Expected error for out of range value")
	}

	// Test defaults
	config5 := map[string]interface{}{
		"api_key": "test-key",
	}
	err = ValidateConfig(config5, schema)
	if err != nil {
		t.Errorf("Config with defaults failed: %v", err)
	}
	if config5["timeout"] != 30 {
		t.Errorf("Expected default timeout=30, got %v", config5["timeout"])
	}
	if config5["enabled"] != true {
		t.Errorf("Expected default enabled=true, got %v", config5["enabled"])
	}
}

// TestPluginError tests error handling
func TestPluginError(t *testing.T) {
	err := NewPluginError(ErrorCodeAuthenticationFailed, "Invalid API key", false)

	if err.Code != ErrorCodeAuthenticationFailed {
		t.Errorf("Expected code '%s', got '%s'", ErrorCodeAuthenticationFailed, err.Code)
	}

	if err.Message != "Invalid API key" {
		t.Errorf("Expected message 'Invalid API key', got '%s'", err.Message)
	}

	if err.Transient {
		t.Error("Expected transient=false")
	}

	errStr := err.Error()
	expectedStr := ErrorCodeAuthenticationFailed + ": Invalid API key"
	if errStr != expectedStr {
		t.Errorf("Expected error string '%s', got '%s'", expectedStr, errStr)
	}

	// Test transient error
	transientErr := NewPluginError(ErrorCodeRateLimitExceeded, "Rate limit", true)
	if !transientErr.Transient {
		t.Error("Expected transient=true for rate limit error")
	}

	if !IsTransientError(transientErr) {
		t.Error("IsTransientError should return true for transient error")
	}

	if GetErrorCode(transientErr) != ErrorCodeRateLimitExceeded {
		t.Errorf("Expected error code '%s', got '%s'", ErrorCodeRateLimitExceeded, GetErrorCode(transientErr))
	}
}

// TestHealthStatus tests health status creation
func TestHealthStatus(t *testing.T) {
	healthy := NewHealthyStatus(50)
	if !healthy.Healthy {
		t.Error("Expected healthy=true")
	}
	if healthy.Message != "OK" {
		t.Errorf("Expected message 'OK', got '%s'", healthy.Message)
	}
	if healthy.Latency != 50 {
		t.Errorf("Expected latency=50, got %d", healthy.Latency)
	}
	if time.Since(healthy.Timestamp) > time.Second {
		t.Error("Timestamp should be recent")
	}

	unhealthy := NewUnhealthyStatus("Connection failed", 1000)
	if unhealthy.Healthy {
		t.Error("Expected healthy=false")
	}
	if unhealthy.Message != "Connection failed" {
		t.Errorf("Expected message 'Connection failed', got '%s'", unhealthy.Message)
	}
	if unhealthy.Latency != 1000 {
		t.Errorf("Expected latency=1000, got %d", unhealthy.Latency)
	}
}

// TestCalculateCost tests cost calculation
func TestCalculateCost(t *testing.T) {
	usage := &UsageInfo{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	// Cost: $0.01 per 1M tokens
	costPerMToken := 0.01
	cost := CalculateCost(usage, costPerMToken)

	expected := 1500.0 * 0.01 / 1_000_000.0 // = 0.000015
	if cost != expected {
		t.Errorf("Expected cost=%v, got %v", expected, cost)
	}

	// Test nil usage
	cost = CalculateCost(nil, costPerMToken)
	if cost != 0 {
		t.Errorf("Expected cost=0 for nil usage, got %v", cost)
	}
}

// TestApplyDefaults tests default application
func TestApplyDefaults(t *testing.T) {
	req := &ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	ApplyDefaults(req)

	if req.Temperature == nil {
		t.Error("Expected temperature to be set")
	} else if *req.Temperature != 0.7 {
		t.Errorf("Expected default temperature=0.7, got %v", *req.Temperature)
	}

	if req.MaxTokens == nil {
		t.Error("Expected max_tokens to be set")
	} else if *req.MaxTokens != 1000 {
		t.Errorf("Expected default max_tokens=1000, got %v", *req.MaxTokens)
	}

	// Test that existing values are not overwritten
	customTemp := 0.9
	customMaxTokens := 2000
	req2 := &ChatCompletionRequest{
		Model:       "gpt-4",
		Temperature: &customTemp,
		MaxTokens:   &customMaxTokens,
	}

	ApplyDefaults(req2)

	if *req2.Temperature != 0.9 {
		t.Errorf("Expected temperature=0.9 to be preserved, got %v", *req2.Temperature)
	}
	if *req2.MaxTokens != 2000 {
		t.Errorf("Expected max_tokens=2000 to be preserved, got %v", *req2.MaxTokens)
	}
}

// Helper function
func floatPtr(f float64) *float64 {
	return &f
}
