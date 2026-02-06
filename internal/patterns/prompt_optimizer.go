package patterns

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/loom/internal/analytics"
)

// PromptOptimizer analyzes prompts and suggests optimizations
type PromptOptimizer struct {
	storage analytics.Storage
	config  *PromptAnalysisConfig
}

// PromptAnalysisConfig configures prompt analysis behavior
type PromptAnalysisConfig struct {
	TimeWindow            time.Duration
	MinPromptTokens       int64   // Minimum tokens to consider for optimization
	VerbosityThreshold    float64 // Ratio of prompt to completion tokens indicating verbosity
	RepetitionThreshold   int     // Number of repeated words to flag
	MinOptimizationSaving float64 // Minimum token reduction percentage (0.0-1.0)
}

// DefaultPromptAnalysisConfig returns sensible defaults
func DefaultPromptAnalysisConfig() *PromptAnalysisConfig {
	return &PromptAnalysisConfig{
		TimeWindow:            7 * 24 * time.Hour, // 7 days
		MinPromptTokens:       100,                // Only analyze prompts with 100+ tokens
		VerbosityThreshold:    5.0,                // Prompt is 5x longer than completion
		RepetitionThreshold:   3,                  // 3+ occurrences of same phrase
		MinOptimizationSaving: 0.10,               // 10% minimum token reduction
	}
}

// PromptOptimization represents a suggested prompt optimization
type PromptOptimization struct {
	ID                    string    `json:"id"`
	Type                  string    `json:"type"` // "verbosity", "repetition", "instruction-clarity"
	OriginalPrompt        string    `json:"original_prompt"`
	OptimizedPrompt       string    `json:"optimized_prompt"`
	OriginalTokens        int64     `json:"original_tokens"`
	EstimatedTokens       int64     `json:"estimated_tokens"`
	TokenSavings          int64     `json:"token_savings"`
	TokenSavingsPercent   float64   `json:"token_savings_percent"`
	CostSavingsUSD        float64   `json:"cost_savings_usd"`
	MonthlyCostSavingsUSD float64   `json:"monthly_cost_savings_usd"`
	Recommendation        string    `json:"recommendation"`
	QualityImpact         string    `json:"quality_impact"` // "minimal", "low", "moderate"
	Confidence            float64   `json:"confidence"`     // 0.0-1.0
	RequestCount          int       `json:"request_count"`
	DetectedAt            time.Time `json:"detected_at"`
}

// PromptAnalysisReport contains the results of prompt analysis
type PromptAnalysisReport struct {
	AnalyzedAt          time.Time             `json:"analyzed_at"`
	TimeWindow          time.Duration         `json:"time_window"`
	TotalPrompts        int                   `json:"total_prompts"`
	OptimizablePrompts  int                   `json:"optimizable_prompts"`
	Optimizations       []*PromptOptimization `json:"optimizations"`
	TotalTokenSavings   int64                 `json:"total_token_savings"`
	TotalCostSavingsUSD float64               `json:"total_cost_savings_usd"`
	MonthlyProjection   float64               `json:"monthly_projection_usd"`
}

// NewPromptOptimizer creates a new prompt optimizer
func NewPromptOptimizer(storage analytics.Storage, config *PromptAnalysisConfig) *PromptOptimizer {
	if config == nil {
		config = DefaultPromptAnalysisConfig()
	}
	return &PromptOptimizer{
		storage: storage,
		config:  config,
	}
}

// AnalyzePrompts analyzes recent prompts and generates optimization suggestions
func (p *PromptOptimizer) AnalyzePrompts(ctx context.Context) (*PromptAnalysisReport, error) {
	// Fetch logs within time window
	startTime := time.Now().Add(-p.config.TimeWindow)
	filter := &analytics.LogFilter{
		StartTime: startTime,
		EndTime:   time.Now(),
		Limit:     10000, // Analyze up to 10K requests
	}

	logs, err := p.storage.GetLogs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	var optimizations []*PromptOptimization
	optimizableCount := 0
	var totalTokenSavings int64
	var totalCostSavings float64

	// Analyze each log for optimization opportunities
	for _, log := range logs {
		if log.PromptTokens < p.config.MinPromptTokens {
			continue // Skip short prompts
		}

		// Extract prompt from request body
		prompt := p.extractPrompt(log.RequestBody)
		if prompt == "" {
			continue
		}

		// Check for verbosity
		if opt := p.detectVerbosity(log, prompt); opt != nil {
			optimizations = append(optimizations, opt)
			optimizableCount++
			totalTokenSavings += opt.TokenSavings
			totalCostSavings += opt.CostSavingsUSD
		}

		// Check for repetition
		if opt := p.detectRepetition(log, prompt); opt != nil {
			optimizations = append(optimizations, opt)
			optimizableCount++
			totalTokenSavings += opt.TokenSavings
			totalCostSavings += opt.CostSavingsUSD
		}

		// Check for unclear instructions
		if opt := p.detectUnclearInstructions(log, prompt); opt != nil {
			optimizations = append(optimizations, opt)
			optimizableCount++
			totalTokenSavings += opt.TokenSavings
			totalCostSavings += opt.CostSavingsUSD
		}
	}

	// Sort by cost savings descending
	sort.Slice(optimizations, func(i, j int) bool {
		return optimizations[i].MonthlyCostSavingsUSD > optimizations[j].MonthlyCostSavingsUSD
	})

	// Calculate monthly projection
	daysInWindow := p.config.TimeWindow.Hours() / 24
	monthlyProjection := totalCostSavings * 30 / daysInWindow

	return &PromptAnalysisReport{
		AnalyzedAt:          time.Now(),
		TimeWindow:          p.config.TimeWindow,
		TotalPrompts:        len(logs),
		OptimizablePrompts:  optimizableCount,
		Optimizations:       optimizations,
		TotalTokenSavings:   totalTokenSavings,
		TotalCostSavingsUSD: totalCostSavings,
		MonthlyProjection:   monthlyProjection,
	}, nil
}

// extractPrompt extracts the prompt text from request body JSON
func (p *PromptOptimizer) extractPrompt(requestBody string) string {
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(requestBody), &body); err != nil {
		return ""
	}

	// Try different common fields
	if prompt, ok := body["prompt"].(string); ok {
		return prompt
	}

	// Check for messages array (chat format)
	if messages, ok := body["messages"].([]interface{}); ok {
		var combinedPrompt strings.Builder
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if content, ok := msgMap["content"].(string); ok {
					combinedPrompt.WriteString(content)
					combinedPrompt.WriteString("\n")
				}
			}
		}
		return combinedPrompt.String()
	}

	return ""
}

// detectVerbosity identifies overly verbose prompts
func (p *PromptOptimizer) detectVerbosity(log *analytics.RequestLog, prompt string) *PromptOptimization {
	if log.CompletionTokens == 0 {
		return nil
	}

	ratio := float64(log.PromptTokens) / float64(log.CompletionTokens)
	if ratio < p.config.VerbosityThreshold {
		return nil
	}

	// Estimate optimization: reduce by 30% for verbose prompts
	estimatedTokens := int64(float64(log.PromptTokens) * 0.7)
	tokenSavings := log.PromptTokens - estimatedTokens
	savingsPercent := float64(tokenSavings) / float64(log.PromptTokens)

	if savingsPercent < p.config.MinOptimizationSaving {
		return nil
	}

	// Estimate cost savings
	avgCostPerToken := log.CostUSD / float64(log.TotalTokens)
	costSavings := float64(tokenSavings) * avgCostPerToken
	monthlySavings := costSavings * 30 * 7 / p.config.TimeWindow.Hours() * 24

	// Generate optimized version (truncated for display)
	optimizedPrompt := p.generateOptimizedPrompt(prompt, "Remove verbose explanations and focus on essential instructions.")

	return &PromptOptimization{
		ID:                    uuid.New().String(),
		Type:                  "verbosity",
		OriginalPrompt:        truncateForDisplay(prompt, 200),
		OptimizedPrompt:       truncateForDisplay(optimizedPrompt, 200),
		OriginalTokens:        log.PromptTokens,
		EstimatedTokens:       estimatedTokens,
		TokenSavings:          tokenSavings,
		TokenSavingsPercent:   savingsPercent,
		CostSavingsUSD:        costSavings,
		MonthlyCostSavingsUSD: monthlySavings,
		Recommendation:        fmt.Sprintf("Prompt is %.1fx longer than completion. Reduce verbose explanations and focus on essential instructions.", ratio),
		QualityImpact:         "minimal",
		Confidence:            0.7,
		RequestCount:          1,
		DetectedAt:            time.Now(),
	}
}

// detectRepetition identifies repeated phrases or instructions
func (p *PromptOptimizer) detectRepetition(log *analytics.RequestLog, prompt string) *PromptOptimization {
	// Look for repeated phrases (3+ words)
	words := strings.Fields(strings.ToLower(prompt))
	if len(words) < 9 { // Need at least 3 phrases of 3 words
		return nil
	}

	// Count trigram occurrences
	trigrams := make(map[string]int)
	for i := 0; i <= len(words)-3; i++ {
		trigram := strings.Join(words[i:i+3], " ")
		trigrams[trigram]++
	}

	// Find most repeated
	maxRepeat := 0
	var mostRepeated string
	for trigram, count := range trigrams {
		if count > maxRepeat {
			maxRepeat = count
			mostRepeated = trigram
		}
	}

	if maxRepeat < p.config.RepetitionThreshold {
		return nil
	}

	// Estimate token savings from removing repetition
	// Each repeat wastes ~3 tokens
	tokenSavings := int64((maxRepeat - 1) * 3)
	savingsPercent := float64(tokenSavings) / float64(log.PromptTokens)

	if savingsPercent < p.config.MinOptimizationSaving {
		return nil
	}

	avgCostPerToken := log.CostUSD / float64(log.TotalTokens)
	costSavings := float64(tokenSavings) * avgCostPerToken
	monthlySavings := costSavings * 30 * 7 / p.config.TimeWindow.Hours() * 24

	optimizedPrompt := p.generateOptimizedPrompt(prompt, fmt.Sprintf("Remove repeated phrase: '%s'", mostRepeated))

	return &PromptOptimization{
		ID:                    uuid.New().String(),
		Type:                  "repetition",
		OriginalPrompt:        truncateForDisplay(prompt, 200),
		OptimizedPrompt:       truncateForDisplay(optimizedPrompt, 200),
		OriginalTokens:        log.PromptTokens,
		EstimatedTokens:       log.PromptTokens - tokenSavings,
		TokenSavings:          tokenSavings,
		TokenSavingsPercent:   savingsPercent,
		CostSavingsUSD:        costSavings,
		MonthlyCostSavingsUSD: monthlySavings,
		Recommendation:        fmt.Sprintf("Detected repeated phrase '%s' (%d times). Remove redundant repetitions.", mostRepeated, maxRepeat),
		QualityImpact:         "minimal",
		Confidence:            0.8,
		RequestCount:          1,
		DetectedAt:            time.Now(),
	}
}

// detectUnclearInstructions identifies prompts that may benefit from clarification
func (p *PromptOptimizer) detectUnclearInstructions(log *analytics.RequestLog, prompt string) *PromptOptimization {
	// Check for indicators of unclear instructions
	unclearIndicators := []string{
		"maybe", "perhaps", "might", "could be", "not sure",
		"i think", "i guess", "kind of", "sort of",
	}

	lowerPrompt := strings.ToLower(prompt)
	unclearCount := 0
	for _, indicator := range unclearIndicators {
		if strings.Contains(lowerPrompt, indicator) {
			unclearCount++
		}
	}

	if unclearCount == 0 {
		return nil
	}

	// Unclear instructions may cause longer completions
	// Estimate 15% token savings from clarifying
	tokenSavings := int64(float64(log.PromptTokens) * 0.15)
	savingsPercent := 0.15

	if savingsPercent < p.config.MinOptimizationSaving {
		return nil
	}

	avgCostPerToken := log.CostUSD / float64(log.TotalTokens)
	costSavings := float64(tokenSavings) * avgCostPerToken
	monthlySavings := costSavings * 30 * 7 / p.config.TimeWindow.Hours() * 24

	optimizedPrompt := p.generateOptimizedPrompt(prompt, "Replace uncertain language with clear, direct instructions.")

	return &PromptOptimization{
		ID:                    uuid.New().String(),
		Type:                  "instruction-clarity",
		OriginalPrompt:        truncateForDisplay(prompt, 200),
		OptimizedPrompt:       truncateForDisplay(optimizedPrompt, 200),
		OriginalTokens:        log.PromptTokens,
		EstimatedTokens:       log.PromptTokens - tokenSavings,
		TokenSavings:          tokenSavings,
		TokenSavingsPercent:   savingsPercent,
		CostSavingsUSD:        costSavings,
		MonthlyCostSavingsUSD: monthlySavings,
		Recommendation:        fmt.Sprintf("Detected %d unclear indicators. Use direct, specific instructions instead of uncertain language.", unclearCount),
		QualityImpact:         "low",
		Confidence:            0.6,
		RequestCount:          1,
		DetectedAt:            time.Now(),
	}
}

// generateOptimizedPrompt creates a suggested optimized version
func (p *PromptOptimizer) generateOptimizedPrompt(original, suggestion string) string {
	// This is a simplified version - a real implementation would use
	// an LLM to actually optimize the prompt
	return fmt.Sprintf("[OPTIMIZED: %s]\n\n%s", suggestion, truncateForDisplay(original, 150))
}

// truncateForDisplay truncates text for display purposes
func truncateForDisplay(text string, maxLen int) string {
	// Remove excessive whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// GetTopOptimizations returns the top N optimization opportunities by savings
func (p *PromptOptimizer) GetTopOptimizations(ctx context.Context, limit int) ([]*PromptOptimization, error) {
	report, err := p.AnalyzePrompts(ctx)
	if err != nil {
		return nil, err
	}

	if len(report.Optimizations) <= limit {
		return report.Optimizations, nil
	}

	return report.Optimizations[:limit], nil
}

// EstimateQualityImpact estimates the quality impact of an optimization
func EstimateQualityImpact(opt *PromptOptimization) string {
	switch {
	case opt.TokenSavingsPercent < 0.20:
		return "minimal"
	case opt.TokenSavingsPercent < 0.40:
		return "low"
	default:
		return "moderate"
	}
}

// CalculateConfidence calculates confidence in the optimization
func CalculateConfidence(opt *PromptOptimization) float64 {
	baseConfidence := 0.5

	// Higher confidence for repetition fixes
	if opt.Type == "repetition" {
		baseConfidence = 0.8
	}

	// Higher confidence for larger savings
	if opt.TokenSavingsPercent > 0.30 {
		baseConfidence += 0.1
	}

	return math.Min(baseConfidence, 1.0)
}
