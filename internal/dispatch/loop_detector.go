package dispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// ActionRecord represents a single action taken by an agent
type ActionRecord struct {
	Timestamp   time.Time              `json:"timestamp"`
	AgentID     string                 `json:"agent_id"`
	ActionType  string                 `json:"action_type"`   // e.g., "read_file", "run_tests", "edit_file"
	ActionData  map[string]interface{} `json:"action_data"`   // Specific details
	ResultHash  string                 `json:"result_hash"`   // Hash of action result
	ProgressKey string                 `json:"progress_key"`  // Key identifying the action pattern
}

// ProgressMetrics tracks progress indicators for a bead
type ProgressMetrics struct {
	FilesRead        int       `json:"files_read"`
	FilesModified    int       `json:"files_modified"`
	TestsRun         int       `json:"tests_run"`
	CommandsExecuted int       `json:"commands_executed"`
	LastProgress     time.Time `json:"last_progress"`
}

// LoopDetector detects stuck loops vs. productive investigation
type LoopDetector struct {
	repeatThreshold int // Number of identical action sequences before flagging as loop
}

// NewLoopDetector creates a new loop detector with default settings
func NewLoopDetector() *LoopDetector {
	return &LoopDetector{
		repeatThreshold: 3, // Flag as loop after 3 identical sequences
	}
}

// SetRepeatThreshold configures how many repeats before flagging as a loop
func (ld *LoopDetector) SetRepeatThreshold(threshold int) {
	if threshold < 2 {
		threshold = 2
	}
	ld.repeatThreshold = threshold
}

// RecordAction adds an action to the bead's dispatch history
func (ld *LoopDetector) RecordAction(bead *models.Bead, action ActionRecord) error {
	if bead.Context == nil {
		bead.Context = make(map[string]string)
	}

	// Generate a progress key for this action type
	action.ProgressKey = ld.generateProgressKey(action)

	// Get existing action history
	history, err := ld.getActionHistory(bead)
	if err != nil {
		log.Printf("[LoopDetector] Failed to parse action history for bead %s: %v", bead.ID, err)
		history = []ActionRecord{}
	}

	// Append new action
	history = append(history, action)

	// Keep only recent history (last 50 actions)
	if len(history) > 50 {
		history = history[len(history)-50:]
	}

	// Store back in bead context
	historyJSON, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal action history: %w", err)
	}
	bead.Context["action_history"] = string(historyJSON)

	// Update progress metrics
	ld.updateProgressMetrics(bead, action)

	return nil
}

// IsStuckInLoop checks if the bead is stuck in a non-productive loop
func (ld *LoopDetector) IsStuckInLoop(bead *models.Bead) (bool, string) {
	history, err := ld.getActionHistory(bead)
	if err != nil || len(history) < ld.repeatThreshold*2 {
		// Not enough history to detect a loop
		return false, ""
	}

	// Check for progress in recent history
	if ld.hasRecentProgress(bead) {
		// Making progress, not stuck
		return false, ""
	}

	// Look for repeated action patterns
	pattern, count := ld.findRepeatedPattern(history)
	if count >= ld.repeatThreshold {
		reason := fmt.Sprintf("Repeated action pattern %d times without progress: %s", count, pattern)
		return true, reason
	}

	return false, ""
}

// getActionHistory retrieves the action history from bead context
func (ld *LoopDetector) getActionHistory(bead *models.Bead) ([]ActionRecord, error) {
	if bead.Context == nil {
		return []ActionRecord{}, nil
	}

	historyJSON := bead.Context["action_history"]
	if historyJSON == "" {
		return []ActionRecord{}, nil
	}

	var history []ActionRecord
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		return nil, err
	}

	return history, nil
}

// generateProgressKey creates a key that identifies the action pattern
func (ld *LoopDetector) generateProgressKey(action ActionRecord) string {
	// Create a signature for this action type and key data
	// This allows us to detect when the same action is repeated
	keyData := fmt.Sprintf("%s:%v", action.ActionType, action.ActionData)

	// For file operations, include the file path
	if filePath, ok := action.ActionData["file_path"].(string); ok {
		keyData = fmt.Sprintf("%s:%s", action.ActionType, filePath)
	}

	// For commands, include the command
	if command, ok := action.ActionData["command"].(string); ok {
		keyData = fmt.Sprintf("%s:%s", action.ActionType, command)
	}

	// Hash to keep it short
	hash := sha256.Sum256([]byte(keyData))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes (16 hex chars)
}

// findRepeatedPattern looks for repeated action sequences
func (ld *LoopDetector) findRepeatedPattern(history []ActionRecord) (string, int) {
	if len(history) < ld.repeatThreshold {
		return "", 0
	}

	// Look at recent history (last 15 actions)
	recent := history
	if len(recent) > 15 {
		recent = recent[len(recent)-15:]
	}

	// Count consecutive identical progress keys
	patternCounts := make(map[string]int)
	var lastKey string
	consecutiveCount := 0

	for _, action := range recent {
		if action.ProgressKey == lastKey {
			consecutiveCount++
		} else {
			if consecutiveCount >= ld.repeatThreshold {
				patternCounts[lastKey] = consecutiveCount
			}
			lastKey = action.ProgressKey
			consecutiveCount = 1
		}
	}

	// Check last sequence
	if consecutiveCount >= ld.repeatThreshold {
		patternCounts[lastKey] = consecutiveCount
	}

	// Find the pattern with highest repeat count
	maxCount := 0
	maxPattern := ""
	for pattern, count := range patternCounts {
		if count > maxCount {
			maxCount = count
			maxPattern = pattern
		}
	}

	return maxPattern, maxCount
}

// hasRecentProgress checks if there has been any progress recently
func (ld *LoopDetector) hasRecentProgress(bead *models.Bead) bool {
	if bead.Context == nil {
		return false
	}

	metricsJSON := bead.Context["progress_metrics"]
	if metricsJSON == "" {
		return false
	}

	var metrics ProgressMetrics
	if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
		return false
	}

	// Consider it progress only if metrics have increased recently (within last 5 minutes)
	// The LastProgress timestamp indicates when the last meaningful action was taken
	if metrics.LastProgress.IsZero() {
		return false
	}

	timeSinceProgress := time.Since(metrics.LastProgress)
	return timeSinceProgress < 5*time.Minute
}

// updateProgressMetrics updates progress tracking based on action
func (ld *LoopDetector) updateProgressMetrics(bead *models.Bead, action ActionRecord) {
	if bead.Context == nil {
		bead.Context = make(map[string]string)
	}

	// Get existing metrics
	var metrics ProgressMetrics
	if metricsJSON := bead.Context["progress_metrics"]; metricsJSON != "" {
		_ = json.Unmarshal([]byte(metricsJSON), &metrics)
	}

	// Update metrics based on action type
	progressMade := false
	switch action.ActionType {
	case "read_file", "glob", "grep":
		metrics.FilesRead++
		progressMade = true
	case "edit_file", "write_file":
		metrics.FilesModified++
		progressMade = true
	case "run_tests", "test":
		metrics.TestsRun++
		progressMade = true
	case "bash", "execute":
		metrics.CommandsExecuted++
		progressMade = true
	}

	if progressMade {
		metrics.LastProgress = time.Now()
	}

	// Store updated metrics
	metricsJSON, err := json.Marshal(metrics)
	if err == nil {
		bead.Context["progress_metrics"] = string(metricsJSON)
	}
}

// GetProgressSummary returns a human-readable progress summary
func (ld *LoopDetector) GetProgressSummary(bead *models.Bead) string {
	if bead.Context == nil {
		return "No progress data"
	}

	metricsJSON := bead.Context["progress_metrics"]
	if metricsJSON == "" {
		return "No progress data"
	}

	var metrics ProgressMetrics
	if err := json.Unmarshal([]byte(metricsJSON), &metrics); err != nil {
		return "Invalid progress data"
	}

	timeSince := "never"
	if !metrics.LastProgress.IsZero() {
		timeSince = time.Since(metrics.LastProgress).Round(time.Second).String() + " ago"
	}

	return fmt.Sprintf("Files read: %d, modified: %d, tests: %d, commands: %d (last: %s)",
		metrics.FilesRead, metrics.FilesModified, metrics.TestsRun,
		metrics.CommandsExecuted, timeSince)
}

// ResetProgress clears progress tracking for a bead
func (ld *LoopDetector) ResetProgress(bead *models.Bead) {
	if bead.Context != nil {
		delete(bead.Context, "action_history")
		delete(bead.Context, "progress_metrics")
	}
}
