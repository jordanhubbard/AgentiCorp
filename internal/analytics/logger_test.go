package analytics

import (
	"context"
	"testing"
	"time"
)

// MockStorage for testing
type MockStorage struct {
	logs  []*RequestLog
	stats *LogStats
}

func (m *MockStorage) SaveLog(ctx context.Context, log *RequestLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *MockStorage) GetLogs(ctx context.Context, filter *LogFilter) ([]*RequestLog, error) {
	return m.logs, nil
}

func (m *MockStorage) GetLogStats(ctx context.Context, filter *LogFilter) (*LogStats, error) {
	return m.stats, nil
}

func (m *MockStorage) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func TestLogRequest_PrivacyDefaults(t *testing.T) {
	storage := &MockStorage{}
	logger := NewLogger(storage, nil) // Use default privacy config

	log := &RequestLog{
		Method:       "POST",
		Path:         "/api/v1/chat/completions",
		RequestBody:  `{"prompt":"secret data"}`,
		ResponseBody: `{"response":"sensitive"}`,
	}

	err := logger.LogRequest(context.Background(), log)
	if err != nil {
		t.Fatalf("LogRequest failed: %v", err)
	}

	if len(storage.logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(storage.logs))
	}

	saved := storage.logs[0]
	
	// With default privacy config, bodies should not be logged
	if saved.RequestBody != "" {
		t.Error("Request body should be empty with default privacy config")
	}
	if saved.ResponseBody != "" {
		t.Error("Response body should be empty with default privacy config")
	}
	
	// ID should be auto-generated
	if saved.ID == "" {
		t.Error("Log ID should be auto-generated")
	}
	
	// Timestamp should be set
	if saved.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestLogRequest_WithBodiesEnabled(t *testing.T) {
	storage := &MockStorage{}
	privacy := &PrivacyConfig{
		LogRequestBodies:  true,
		LogResponseBodies: true,
		MaxBodyLength:     0, // No limit
		RedactPatterns:    []string{},
	}
	logger := NewLogger(storage, privacy)

	log := &RequestLog{
		Method:       "POST",
		Path:         "/api/v1/chat/completions",
		RequestBody:  `{"prompt":"test"}`,
		ResponseBody: `{"response":"result"}`,
	}

	err := logger.LogRequest(context.Background(), log)
	if err != nil {
		t.Fatalf("LogRequest failed: %v", err)
	}

	saved := storage.logs[0]
	
	// Bodies should be logged
	if saved.RequestBody == "" {
		t.Error("Request body should be logged")
	}
	if saved.ResponseBody == "" {
		t.Error("Response body should be logged")
	}
}

func TestLogRequest_Redaction(t *testing.T) {
	storage := &MockStorage{}
	privacy := &PrivacyConfig{
		LogRequestBodies:  true,
		LogResponseBodies: true,
		RedactPatterns: []string{
			`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
		},
	}
	logger := NewLogger(storage, privacy)

	originalBody := `Contact: user@example.com for support`
	log := &RequestLog{
		RequestBody: originalBody,
	}

	err := logger.LogRequest(context.Background(), log)
	if err != nil {
		t.Fatalf("LogRequest failed: %v", err)
	}

	saved := storage.logs[0]
	
	// Email should be redacted - check for [REDACTED] marker
	if saved.RequestBody == originalBody {
		t.Error("Email should be redacted")
	}
	if saved.RequestBody != `Contact: [REDACTED] for support` {
		t.Errorf("Expected 'Contact: [REDACTED] for support', got: %s", saved.RequestBody)
	}
}

func TestLogRequest_BodyTruncation(t *testing.T) {
	storage := &MockStorage{}
	privacy := &PrivacyConfig{
		LogRequestBodies: true,
		MaxBodyLength:    10,
	}
	logger := NewLogger(storage, privacy)

	log := &RequestLog{
		RequestBody: "this is a very long request body that should be truncated",
	}

	err := logger.LogRequest(context.Background(), log)
	if err != nil {
		t.Fatalf("LogRequest failed: %v", err)
	}

	saved := storage.logs[0]
	
	// Body should be truncated
	if len(saved.RequestBody) > 25 { // 10 chars + "... [truncated]"
		t.Errorf("Body not truncated properly, length: %d", len(saved.RequestBody))
	}
	if saved.RequestBody != "this is a ... [truncated]" {
		t.Errorf("Expected truncation marker, got: %s", saved.RequestBody)
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name           string
		costPerMToken  float64
		tokens         int64
		expectedCost   float64
	}{
		{
			name:          "Standard calculation",
			costPerMToken: 10.0,
			tokens:        1000000,
			expectedCost:  10.0,
		},
		{
			name:          "Half million tokens",
			costPerMToken: 10.0,
			tokens:        500000,
			expectedCost:  5.0,
		},
		{
			name:          "Small request",
			costPerMToken: 0.5,
			tokens:        1000,
			expectedCost:  0.0005,
		},
		{
			name:          "Zero cost provider",
			costPerMToken: 0.0,
			tokens:        1000000,
			expectedCost:  0.0,
		},
		{
			name:          "Zero tokens",
			costPerMToken: 10.0,
			tokens:        0,
			expectedCost:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateCost(tt.costPerMToken, tt.tokens)
			if cost != tt.expectedCost {
				t.Errorf("Expected cost %f, got %f", tt.expectedCost, cost)
			}
		})
	}
}

func TestSanitizeForLogging(t *testing.T) {
	data := map[string]interface{}{
		"username": "john",
		"password": "secret123",
		"api_key":  "sk-12345678901234567890",
		"data":     "normal data",
	}

	sanitized := SanitizeForLogging(data)

	// Should redact sensitive fields
	if sanitized == "" {
		t.Error("Sanitized output should not be empty")
	}
	
	// Password should be redacted
	if sanitized == `{"api_key":"sk-12345678901234567890","data":"normal data","password":"secret123","username":"john"}` {
		t.Error("Sensitive data not redacted")
	}
	
	// Username and normal data should remain
	if sanitized == "" {
		t.Error("Non-sensitive data should remain")
	}
}

func TestDefaultPrivacyConfig(t *testing.T) {
	config := DefaultPrivacyConfig()

	if config.LogRequestBodies {
		t.Error("Default should not log request bodies")
	}
	if config.LogResponseBodies {
		t.Error("Default should not log response bodies")
	}
	if len(config.RedactPatterns) == 0 {
		t.Error("Default should have redaction patterns")
	}
	if config.MaxBodyLength <= 0 {
		t.Error("Default should have max body length")
	}
}
