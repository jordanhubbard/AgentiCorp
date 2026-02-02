package patterns

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jordanhubbard/agenticorp/internal/analytics"
)

// InMemoryStorage is a simple in-memory implementation for testing
type testStorage struct {
	logs []*analytics.RequestLog
}

func newTestStorage() *testStorage {
	return &testStorage{
		logs: make([]*analytics.RequestLog, 0),
	}
}

func (s *testStorage) SaveLog(ctx context.Context, log *analytics.RequestLog) error {
	s.logs = append(s.logs, log)
	return nil
}

func (s *testStorage) GetLogs(ctx context.Context, filter *analytics.LogFilter) ([]*analytics.RequestLog, error) {
	filtered := make([]*analytics.RequestLog, 0)
	for _, log := range s.logs {
		if filter.UserID != "" && log.UserID != filter.UserID {
			continue
		}
		if filter.ProviderID != "" && log.ProviderID != filter.ProviderID {
			continue
		}
		if !filter.StartTime.IsZero() && log.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && log.Timestamp.After(filter.EndTime) {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered, nil
}

func (s *testStorage) GetLogStats(ctx context.Context, filter *analytics.LogFilter) (*analytics.LogStats, error) {
	logs, err := s.GetLogs(ctx, filter)
	if err != nil {
		return nil, err
	}

	stats := &analytics.LogStats{
		RequestsByUser:     make(map[string]int64),
		RequestsByProvider: make(map[string]int64),
		CostByProvider:     make(map[string]float64),
		CostByUser:         make(map[string]float64),
	}

	for _, log := range logs {
		stats.TotalRequests++
		stats.TotalTokens += log.TotalTokens
		stats.TotalCostUSD += log.CostUSD
	}

	return stats, nil
}

func (s *testStorage) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func TestPromptOptimizer_DetectVerbosity(t *testing.T) {
	storage := newTestStorage()
	optimizer := NewPromptOptimizer(storage, DefaultPromptAnalysisConfig())

	// Create a verbose prompt log
	verbosePrompt := `Please write a function that adds two numbers together.
	I need this function to be very clear and well-documented.
	Make sure to include detailed comments explaining every step.
	The function should take two parameters and return their sum.
	Please ensure the code follows best practices and is easy to understand.
	Add error handling if needed and make it robust.`

	requestBody, _ := json.Marshal(map[string]interface{}{
		"prompt": verbosePrompt,
	})

	log := &analytics.RequestLog{
		ID:               "test-1",
		Timestamp:        time.Now(),
		UserID:           "user1",
		ProviderID:       "test",
		ModelName:        "test-model",
		PromptTokens:     150, // Verbose
		CompletionTokens: 20,  // Short completion
		TotalTokens:      170,
		CostUSD:          0.01,
		RequestBody:      string(requestBody),
	}

	storage.SaveLog(context.Background(), log)

	// Run analysis
	report, err := optimizer.AnalyzePrompts(context.Background())
	if err != nil {
		t.Fatalf("AnalyzePrompts failed: %v", err)
	}

	if report.OptimizablePrompts == 0 {
		t.Error("Expected to find optimizable prompts")
	}

	if len(report.Optimizations) == 0 {
		t.Error("Expected optimization suggestions")
	}

	// Check for verbosity optimization
	hasVerbosity := false
	for _, opt := range report.Optimizations {
		if opt.Type == "verbosity" {
			hasVerbosity = true
			if opt.TokenSavings <= 0 {
				t.Error("Expected positive token savings")
			}
			if opt.Confidence <= 0 || opt.Confidence > 1 {
				t.Errorf("Invalid confidence: %f", opt.Confidence)
			}
		}
	}

	if !hasVerbosity {
		t.Error("Expected verbosity optimization")
	}
}

func TestPromptOptimizer_DetectRepetition(t *testing.T) {
	storage := newTestStorage()
	config := DefaultPromptAnalysisConfig()
	config.MinOptimizationSaving = 0.05 // Lower threshold for test
	optimizer := NewPromptOptimizer(storage, config)

	// Create a prompt with clear repetition (meeting threshold of 3+ occurrences)
	// Each repetition is ~3 tokens, with 5 repetitions = 12 tokens savings on 100 token prompt = 12%
	repeatedPrompt := `Please write a function please write a function please write a function please write a function please write a function that adds numbers together in Go language`

	requestBody, _ := json.Marshal(map[string]interface{}{
		"prompt": repeatedPrompt,
	})

	log := &analytics.RequestLog{
		ID:               "test-2",
		Timestamp:        time.Now(),
		UserID:           "user1",
		ProviderID:       "test",
		ModelName:        "test-model",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		CostUSD:          0.01,
		RequestBody:      string(requestBody),
	}

	storage.SaveLog(context.Background(), log)

	// Run analysis
	report, err := optimizer.AnalyzePrompts(context.Background())
	if err != nil {
		t.Fatalf("AnalyzePrompts failed: %v", err)
	}

	// Check for repetition optimization
	hasRepetition := false
	for _, opt := range report.Optimizations {
		if opt.Type == "repetition" {
			hasRepetition = true
			if opt.TokenSavings <= 0 {
				t.Error("Expected positive token savings")
			}
		}
	}

	if !hasRepetition {
		t.Error("Expected repetition optimization")
	}
}

func TestPromptOptimizer_DetectUnclearInstructions(t *testing.T) {
	storage := newTestStorage()
	optimizer := NewPromptOptimizer(storage, DefaultPromptAnalysisConfig())

	// Create a prompt with unclear instructions
	unclearPrompt := `Maybe write a function that might add two numbers.
	I think it should perhaps return the sum. I'm not sure if we need error handling.
	It could be useful to sort of validate the inputs, kind of.`

	requestBody, _ := json.Marshal(map[string]interface{}{
		"prompt": unclearPrompt,
	})

	log := &analytics.RequestLog{
		ID:               "test-3",
		Timestamp:        time.Now(),
		UserID:           "user1",
		ProviderID:       "test",
		ModelName:        "test-model",
		PromptTokens:     120,
		CompletionTokens: 80,
		TotalTokens:      200,
		CostUSD:          0.015,
		RequestBody:      string(requestBody),
	}

	storage.SaveLog(context.Background(), log)

	// Run analysis
	report, err := optimizer.AnalyzePrompts(context.Background())
	if err != nil {
		t.Fatalf("AnalyzePrompts failed: %v", err)
	}

	// Check for instruction clarity optimization
	hasClarity := false
	for _, opt := range report.Optimizations {
		if opt.Type == "instruction-clarity" {
			hasClarity = true
			if opt.TokenSavings <= 0 {
				t.Error("Expected positive token savings")
			}
		}
	}

	if !hasClarity {
		t.Error("Expected instruction clarity optimization")
	}
}

func TestPromptOptimizer_ChatMessages(t *testing.T) {
	storage := newTestStorage()
	optimizer := NewPromptOptimizer(storage, DefaultPromptAnalysisConfig())

	// Create a chat-style request with messages array
	messages := []map[string]interface{}{
		{"role": "system", "content": "You are a helpful assistant who writes code."},
		{"role": "user", "content": "Please write a very detailed function maybe that might add numbers I think."},
	}

	requestBody, _ := json.Marshal(map[string]interface{}{
		"messages": messages,
	})

	log := &analytics.RequestLog{
		ID:               "test-4",
		Timestamp:        time.Now(),
		UserID:           "user1",
		ProviderID:       "test",
		ModelName:        "test-model",
		PromptTokens:     150,
		CompletionTokens: 30,
		TotalTokens:      180,
		CostUSD:          0.012,
		RequestBody:      string(requestBody),
	}

	storage.SaveLog(context.Background(), log)

	// Run analysis
	report, err := optimizer.AnalyzePrompts(context.Background())
	if err != nil {
		t.Fatalf("AnalyzePrompts failed: %v", err)
	}

	if report.TotalPrompts == 0 {
		t.Error("Expected to process chat message")
	}
}

func TestPromptOptimizer_ReportMetrics(t *testing.T) {
	storage := newTestStorage()
	optimizer := NewPromptOptimizer(storage, DefaultPromptAnalysisConfig())

	// Add multiple logs
	for i := 0; i < 5; i++ {
		verbosePrompt := `This is a very verbose prompt with lots of unnecessary words and explanations that could be simplified.`
		requestBody, _ := json.Marshal(map[string]interface{}{
			"prompt": verbosePrompt,
		})

		log := &analytics.RequestLog{
			ID:               fmt.Sprintf("test-%d", i),
			Timestamp:        time.Now(),
			UserID:           "user1",
			ProviderID:       "test",
			ModelName:        "test-model",
			PromptTokens:     150,
			CompletionTokens: 20,
			TotalTokens:      170,
			CostUSD:          0.01,
			RequestBody:      string(requestBody),
		}

		storage.SaveLog(context.Background(), log)
	}

	// Run analysis
	report, err := optimizer.AnalyzePrompts(context.Background())
	if err != nil {
		t.Fatalf("AnalyzePrompts failed: %v", err)
	}

	if report.TotalPrompts != 5 {
		t.Errorf("Expected 5 total prompts, got %d", report.TotalPrompts)
	}

	if report.TotalTokenSavings <= 0 {
		t.Error("Expected positive total token savings")
	}

	if report.TotalCostSavingsUSD <= 0 {
		t.Error("Expected positive cost savings")
	}

	if report.MonthlyProjection <= 0 {
		t.Error("Expected positive monthly projection")
	}
}

func TestPromptOptimizer_MinimumThresholds(t *testing.T) {
	config := DefaultPromptAnalysisConfig()
	config.MinPromptTokens = 200 // Set high threshold

	storage := newTestStorage()
	optimizer := NewPromptOptimizer(storage, config)

	// Create a prompt below threshold
	shortPrompt := "Add two numbers"
	requestBody, _ := json.Marshal(map[string]interface{}{
		"prompt": shortPrompt,
	})

	log := &analytics.RequestLog{
		ID:               "test-short",
		Timestamp:        time.Now(),
		UserID:           "user1",
		ProviderID:       "test",
		ModelName:        "test-model",
		PromptTokens:     50, // Below threshold
		CompletionTokens: 100,
		TotalTokens:      150,
		CostUSD:          0.01,
		RequestBody:      string(requestBody),
	}

	storage.SaveLog(context.Background(), log)

	// Run analysis
	report, err := optimizer.AnalyzePrompts(context.Background())
	if err != nil {
		t.Fatalf("AnalyzePrompts failed: %v", err)
	}

	if report.OptimizablePrompts != 0 {
		t.Error("Expected no optimizations for prompt below threshold")
	}
}
