package plugin

import (
	"context"
	"fmt"
	"time"
)

// BasePlugin provides a default implementation of common plugin methods.
// Plugin developers can embed this in their plugin implementation
// to get default behavior for optional methods.
type BasePlugin struct {
	metadata *Metadata
	config   map[string]interface{}
}

// NewBasePlugin creates a new BasePlugin with the given metadata.
func NewBasePlugin(metadata *Metadata) *BasePlugin {
	return &BasePlugin{
		metadata: metadata,
		config:   make(map[string]interface{}),
	}
}

// GetMetadata returns the plugin metadata.
func (bp *BasePlugin) GetMetadata() *Metadata {
	return bp.metadata
}

// Initialize stores the configuration.
// Plugins can override this to perform custom initialization.
func (bp *BasePlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	bp.config = config
	return nil
}

// GetConfig returns the stored configuration.
func (bp *BasePlugin) GetConfig() map[string]interface{} {
	return bp.config
}

// GetConfigString retrieves a string configuration value.
func (bp *BasePlugin) GetConfigString(key string) (string, bool) {
	val, ok := bp.config[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetConfigInt retrieves an integer configuration value.
func (bp *BasePlugin) GetConfigInt(key string) (int, bool) {
	val, ok := bp.config[key]
	if !ok {
		return 0, false
	}
	
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetConfigBool retrieves a boolean configuration value.
func (bp *BasePlugin) GetConfigBool(key string) (bool, bool) {
	val, ok := bp.config[key]
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// GetConfigFloat retrieves a float configuration value.
func (bp *BasePlugin) GetConfigFloat(key string) (float64, bool) {
	val, ok := bp.config[key]
	if !ok {
		return 0, false
	}
	
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// Cleanup is a no-op by default.
// Plugins can override this to perform cleanup.
func (bp *BasePlugin) Cleanup(ctx context.Context) error {
	return nil
}

// ValidateConfig validates the plugin configuration against the schema.
func ValidateConfig(config map[string]interface{}, schema []ConfigField) error {
	for _, field := range schema {
		value, exists := config[field.Name]

		// Check required fields
		if field.Required && !exists {
			return NewPluginError(
				ErrorCodeInvalidRequest,
				fmt.Sprintf("required field '%s' is missing", field.Name),
				false,
			)
		}

		// Use default if not provided
		if !exists {
			if field.Default != nil {
				config[field.Name] = field.Default
			}
			continue
		}

		// Validate type
		if err := validateType(value, field.Type); err != nil {
			return NewPluginError(
				ErrorCodeInvalidRequest,
				fmt.Sprintf("field '%s': %v", field.Name, err),
				false,
			)
		}

		// Validate rules
		if field.Validation != nil {
			if err := validateRules(value, field); err != nil {
				return NewPluginError(
					ErrorCodeInvalidRequest,
					fmt.Sprintf("field '%s': %v", field.Name, err),
					false,
				)
			}
		}
	}

	return nil
}

func validateType(value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "int":
		switch value.(type) {
		case int, int64, float64:
			// Accept numeric types
		default:
			return fmt.Errorf("expected int, got %T", value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	case "float":
		switch value.(type) {
		case float64, float32, int, int64:
			// Accept numeric types
		default:
			return fmt.Errorf("expected float, got %T", value)
		}
	default:
		// Unknown type - allow it
	}
	return nil
}

func validateRules(value interface{}, field ConfigField) error {
	rules := field.Validation

	// String validations
	if str, ok := value.(string); ok {
		if rules.MinLength > 0 && len(str) < rules.MinLength {
			return fmt.Errorf("string too short (min: %d)", rules.MinLength)
		}
		if rules.MaxLength > 0 && len(str) > rules.MaxLength {
			return fmt.Errorf("string too long (max: %d)", rules.MaxLength)
		}
		// TODO: Pattern validation with regex
	}

	// Numeric validations
	if num, ok := toFloat64(value); ok {
		if rules.Min != nil && num < *rules.Min {
			return fmt.Errorf("value too small (min: %v)", *rules.Min)
		}
		if rules.Max != nil && num > *rules.Max {
			return fmt.Errorf("value too large (max: %v)", *rules.Max)
		}
	}

	// Enum validation
	if len(rules.Enum) > 0 {
		found := false
		for _, allowed := range rules.Enum {
			if value == allowed {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value not in allowed list: %v", rules.Enum)
		}
	}

	return nil
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

// NewHealthyStatus creates a healthy status with the given latency.
func NewHealthyStatus(latencyMs int64) *HealthStatus {
	return &HealthStatus{
		Healthy:   true,
		Message:   "OK",
		Latency:   latencyMs,
		Timestamp: time.Now(),
	}
}

// NewUnhealthyStatus creates an unhealthy status with the given message.
func NewUnhealthyStatus(message string, latencyMs int64) *HealthStatus {
	return &HealthStatus{
		Healthy:   false,
		Message:   message,
		Latency:   latencyMs,
		Timestamp: time.Now(),
	}
}

// ApplyDefaults applies default values to a request if not set.
func ApplyDefaults(req *ChatCompletionRequest) {
	// Set default temperature if not provided
	if req.Temperature == nil {
		defaultTemp := 0.7
		req.Temperature = &defaultTemp
	}

	// Set default max tokens if not provided
	if req.MaxTokens == nil {
		defaultMaxTokens := 1000
		req.MaxTokens = &defaultMaxTokens
	}
}

// CalculateCost estimates the cost based on token usage and pricing.
func CalculateCost(usage *UsageInfo, costPerMToken float64) float64 {
	if usage == nil {
		return 0
	}
	return float64(usage.TotalTokens) * costPerMToken / 1_000_000.0
}

// IsTransientError determines if an error is transient (retry-able).
func IsTransientError(err error) bool {
	if pluginErr, ok := err.(*PluginError); ok {
		return pluginErr.Transient
	}
	return false
}

// GetErrorCode extracts the error code from a plugin error.
func GetErrorCode(err error) string {
	if pluginErr, ok := err.(*PluginError); ok {
		return pluginErr.Code
	}
	return ""
}
