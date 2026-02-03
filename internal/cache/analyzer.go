package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/agenticorp/internal/analytics"
)

// Analyzer analyzes request patterns to identify caching opportunities
type Analyzer struct {
	logStorage analytics.Storage
	config     *AnalysisConfig
}

// NewAnalyzer creates a new cache analyzer
func NewAnalyzer(logStorage analytics.Storage, config *AnalysisConfig) *Analyzer {
	if config == nil {
		config = DefaultAnalysisConfig()
	}

	return &Analyzer{
		logStorage: logStorage,
		config:     config,
	}
}

// Analyze performs a comprehensive cache analysis
func (a *Analyzer) Analyze(ctx context.Context) (*AnalysisReport, error) {
	now := time.Now()
	startTime := now.Add(-a.config.TimeWindow)

	// Fetch request logs
	logs, err := a.logStorage.GetLogs(ctx, &analytics.LogFilter{
		StartTime: startTime,
		EndTime:   now,
		Limit:     100000, // Analyze up to 100K requests
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	// Detect duplicates
	duplicates := a.detectDuplicates(logs)

	// Identify opportunities
	opportunities := a.identifyOpportunities(duplicates)

	// Calculate totals
	totalSavings := 0.0
	totalTokens := int64(0)
	totalLatency := int64(0)
	for _, opp := range opportunities {
		totalSavings += opp.CostSavableUSD
		totalTokens += opp.TokensSavable
		totalLatency += opp.LatencySavableMs
	}

	// Generate recommendations
	recommendations := a.generateRecommendations(opportunities)

	// Calculate duplicate percentage
	duplicateCount := int64(0)
	for _, dup := range duplicates {
		duplicateCount += int64(dup.OccurrenceCount - 1) // -1 because first occurrence isn't a duplicate
	}

	duplicatePercent := 0.0
	if len(logs) > 0 {
		duplicatePercent = float64(duplicateCount) / float64(len(logs)) * 100
	}

	// Project monthly savings
	daysInWindow := a.config.TimeWindow.Hours() / 24
	monthlyProjection := 0.0
	if daysInWindow > 0 {
		monthlyProjection = totalSavings * (30.0 / daysInWindow)
	}

	report := &AnalysisReport{
		AnalyzedAt:        now,
		TimeWindow:        a.config.TimeWindow,
		TimeWindowStart:   startTime,
		TimeWindowEnd:     now,
		TotalRequests:     int64(len(logs)),
		UniqueRequests:    int64(len(duplicates)),
		DuplicateCount:    duplicateCount,
		DuplicatePercent:  duplicatePercent,
		Opportunities:     opportunities,
		TotalSavingsUSD:   totalSavings,
		TotalTokensSaved:  totalTokens,
		TotalLatencySaved: totalLatency,
		MonthlyProjection: monthlyProjection,
		Recommendations:   recommendations,
	}

	return report, nil
}

// detectDuplicates finds duplicate requests in the logs
func (a *Analyzer) detectDuplicates(logs []*analytics.RequestLog) []*DuplicateRequest {
	// Group logs by hash (provider + model + request body)
	groups := make(map[string][]*analytics.RequestLog)

	for _, log := range logs {
		// Skip failed requests
		if log.StatusCode >= 400 {
			continue
		}

		// Generate hash
		hash := a.hashRequest(log.ProviderID, log.ModelName, log.RequestBody)
		groups[hash] = append(groups[hash], log)
	}

	// Convert to DuplicateRequest structs
	var duplicates []*DuplicateRequest
	for hash, groupLogs := range groups {
		if len(groupLogs) < a.config.MinOccurrences {
			continue
		}

		totalTokens := int64(0)
		totalCost := 0.0
		totalLatency := int64(0)
		var requestIDs []string
		var firstSeen, lastSeen time.Time

		for i, log := range groupLogs {
			if i == 0 {
				firstSeen = log.Timestamp
				lastSeen = log.Timestamp
			} else {
				if log.Timestamp.Before(firstSeen) {
					firstSeen = log.Timestamp
				}
				if log.Timestamp.After(lastSeen) {
					lastSeen = log.Timestamp
				}
			}

			totalTokens += log.TotalTokens
			totalCost += log.CostUSD
			totalLatency += log.LatencyMs
			requestIDs = append(requestIDs, log.ID)
		}

		avgLatency := int64(0)
		if len(groupLogs) > 0 {
			avgLatency = totalLatency / int64(len(groupLogs))
		}

		dup := &DuplicateRequest{
			RequestHash:     hash,
			FirstSeen:       firstSeen,
			LastSeen:        lastSeen,
			OccurrenceCount: len(groupLogs),
			ProviderID:      groupLogs[0].ProviderID,
			ModelName:       groupLogs[0].ModelName,
			TotalTokens:     totalTokens,
			TotalCost:       totalCost,
			AvgLatencyMs:    avgLatency,
			SampleRequest:   truncateString(groupLogs[0].RequestBody, 200),
			RequestIDs:      requestIDs,
		}

		duplicates = append(duplicates, dup)
	}

	// Sort by potential savings (descending)
	sort.Slice(duplicates, func(i, j int) bool {
		savingsI := a.calculateSavings(duplicates[i])
		savingsJ := a.calculateSavings(duplicates[j])
		return savingsI > savingsJ
	})

	return duplicates
}

// identifyOpportunities converts duplicates into actionable opportunities
func (a *Analyzer) identifyOpportunities(duplicates []*DuplicateRequest) []*CacheOpportunity {
	var opportunities []*CacheOpportunity

	for _, dup := range duplicates {
		// Calculate savings (all occurrences except the first would be cache hits)
		potentialHits := dup.OccurrenceCount - 1
		if potentialHits < 1 {
			continue
		}

		// Average tokens per request
		avgTokens := dup.TotalTokens / int64(dup.OccurrenceCount)
		tokensSavable := avgTokens * int64(potentialHits)

		// Average cost per request
		avgCost := dup.TotalCost / float64(dup.OccurrenceCount)
		costSavable := avgCost * float64(potentialHits)

		// Skip if below minimum savings threshold
		if costSavable < a.config.MinSavingsUSD {
			continue
		}

		// Latency savings
		latencySavable := dup.AvgLatencyMs * int64(potentialHits)

		// Calculate hit rate
		hitRate := float64(potentialHits) / float64(dup.OccurrenceCount) * 100

		// Determine TTL based on time between requests
		suggestedTTL := a.suggestTTL(dup.FirstSeen, dup.LastSeen, dup.OccurrenceCount)

		// Determine priority
		priority := a.determinePriority(costSavable, hitRate)

		// Generate recommendation
		recommendation := a.formatRecommendation(dup, costSavable, hitRate, suggestedTTL)

		// Determine if auto-enableable
		monthlyProjection := costSavable * (30.0 / (a.config.TimeWindow.Hours() / 24))
		autoEnableable := a.config.AutoEnable &&
			monthlyProjection >= a.config.AutoEnableMinUSD &&
			hitRate >= a.config.AutoEnableMinRate*100

		opp := &CacheOpportunity{
			ID:               uuid.New().String(),
			Pattern:          fmt.Sprintf("%s:%s", dup.ProviderID, dup.ModelName),
			Description:      fmt.Sprintf("Duplicate requests to %s using %s", dup.ProviderID, dup.ModelName),
			RequestCount:     dup.OccurrenceCount,
			PotentialHits:    potentialHits,
			HitRatePercent:   hitRate,
			TokensSavable:    tokensSavable,
			CostSavableUSD:   costSavable,
			LatencySavableMs: latencySavable,
			Recommendation:   recommendation,
			AutoEnableable:   autoEnableable,
			SuggestedTTL:     suggestedTTL,
			Priority:         priority,
			ProviderID:       dup.ProviderID,
			ModelName:        dup.ModelName,
		}

		opportunities = append(opportunities, opp)
	}

	return opportunities
}

// generateRecommendations creates actionable recommendations
func (a *Analyzer) generateRecommendations(opportunities []*CacheOpportunity) []string {
	var recommendations []string

	if len(opportunities) == 0 {
		return []string{"No significant caching opportunities detected. Continue monitoring."}
	}

	// Overall recommendation
	totalSavings := 0.0
	for _, opp := range opportunities {
		totalSavings += opp.CostSavableUSD
	}

	daysInWindow := a.config.TimeWindow.Hours() / 24
	monthlySavings := totalSavings * (30.0 / daysInWindow)

	recommendations = append(recommendations,
		fmt.Sprintf("Enable caching to save approximately $%.2f per month", monthlySavings))

	// Top opportunities
	if len(opportunities) > 0 && opportunities[0].Priority == "high" {
		opp := opportunities[0]
		recommendations = append(recommendations,
			fmt.Sprintf("Priority: Enable caching for %s (%.0f%% hit rate, $%.2f savings)",
				opp.Pattern, opp.HitRatePercent, opp.CostSavableUSD))
	}

	// TTL recommendations
	if len(opportunities) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Recommended TTL: %s for most patterns", opportunities[0].SuggestedTTL))
	}

	// Auto-optimization
	autoCount := 0
	for _, opp := range opportunities {
		if opp.AutoEnableable {
			autoCount++
		}
	}
	if autoCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d opportunities qualify for auto-optimization", autoCount))
	}

	return recommendations
}

// Helper functions

func (a *Analyzer) hashRequest(providerID, modelName, requestBody string) string {
	hasher := sha256.New()
	hasher.Write([]byte(providerID))
	hasher.Write([]byte(":"))
	hasher.Write([]byte(modelName))
	hasher.Write([]byte(":"))
	hasher.Write([]byte(requestBody))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (a *Analyzer) calculateSavings(dup *DuplicateRequest) float64 {
	potentialHits := dup.OccurrenceCount - 1
	if potentialHits < 1 {
		return 0
	}
	avgCost := dup.TotalCost / float64(dup.OccurrenceCount)
	return avgCost * float64(potentialHits)
}

func (a *Analyzer) suggestTTL(firstSeen, lastSeen time.Time, count int) time.Duration {
	// Calculate average time between requests
	if count <= 1 {
		return 1 * time.Hour // Default
	}

	totalDuration := lastSeen.Sub(firstSeen)
	avgInterval := totalDuration / time.Duration(count-1)

	// Set TTL to 2x the average interval (ensure we catch most duplicates)
	suggestedTTL := avgInterval * 2

	// Clamp to reasonable bounds
	if suggestedTTL < 5*time.Minute {
		return 5 * time.Minute
	}
	if suggestedTTL > 24*time.Hour {
		return 24 * time.Hour
	}

	// Round to nearest hour
	hours := int(suggestedTTL.Hours())
	if hours == 0 {
		// Round to nearest 15 minutes
		minutes := int(suggestedTTL.Minutes())
		minutes = ((minutes + 7) / 15) * 15
		return time.Duration(minutes) * time.Minute
	}

	return time.Duration(hours) * time.Hour
}

func (a *Analyzer) determinePriority(savingsUSD, hitRate float64) string {
	// High priority: >$1 savings and >70% hit rate
	if savingsUSD > 1.0 && hitRate > 70 {
		return "high"
	}

	// Medium priority: >$0.10 savings and >50% hit rate
	if savingsUSD > 0.10 && hitRate > 50 {
		return "medium"
	}

	return "low"
}

func (a *Analyzer) formatRecommendation(dup *DuplicateRequest, savingsUSD, hitRate float64, ttl time.Duration) string {
	action := "Consider enabling"
	if hitRate > 80 {
		action = "Strongly recommend enabling"
	} else if hitRate > 60 {
		action = "Recommend enabling"
	}

	return fmt.Sprintf("%s caching for %s:%s (%.0f%% hit rate, $%.2f savings, %s TTL)",
		action, dup.ProviderID, dup.ModelName, hitRate, savingsUSD, formatDuration(ttl))
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	return fmt.Sprintf("%dm", minutes)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
