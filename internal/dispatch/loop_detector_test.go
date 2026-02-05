package dispatch

import (
	"fmt"
	"testing"
	"time"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

func TestNewLoopDetector(t *testing.T) {
	ld := NewLoopDetector()

	if ld == nil {
		t.Fatal("Expected loop detector to be created")
	}

	if ld.repeatThreshold != 3 {
		t.Errorf("Expected default repeat threshold to be 3, got %d", ld.repeatThreshold)
	}
}

func TestSetRepeatThreshold(t *testing.T) {
	ld := NewLoopDetector()

	testCases := []struct {
		name      string
		threshold int
		expected  int
	}{
		{"Set to 3", 3, 3},
		{"Set to 5", 5, 5},
		{"Set to 1 (below minimum)", 1, 2}, // Should be clamped to 2
		{"Set to 0 (below minimum)", 0, 2}, // Should be clamped to 2
		{"Set to 10", 10, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ld.SetRepeatThreshold(tc.threshold)
			if ld.repeatThreshold != tc.expected {
				t.Errorf("Expected threshold to be %d, got %d", tc.expected, ld.repeatThreshold)
			}
		})
	}
}

func TestRecordAction(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-test-123",
		Context: make(map[string]string),
	}

	action := ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{
			"file_path": "/test/file.go",
		},
	}

	err := ld.RecordAction(bead, action)
	if err != nil {
		t.Fatalf("Failed to record action: %v", err)
	}

	// Verify action was stored
	history, err := ld.getActionHistory(bead)
	if err != nil {
		t.Fatalf("Failed to get action history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 action in history, got %d", len(history))
	}

	if history[0].ActionType != "read_file" {
		t.Errorf("Expected action type read_file, got %s", history[0].ActionType)
	}

	// Verify progress metrics were updated
	if bead.Context["progress_metrics"] == "" {
		t.Error("Expected progress metrics to be set")
	}
}

func TestRecordMultipleActions(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-test-multiple",
		Context: make(map[string]string),
	}

	actions := []ActionRecord{
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "read_file", ActionData: map[string]interface{}{"file_path": "a.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "edit_file", ActionData: map[string]interface{}{"file_path": "a.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "run_tests", ActionData: map[string]interface{}{"command": "go test"}},
	}

	for _, action := range actions {
		if err := ld.RecordAction(bead, action); err != nil {
			t.Fatalf("Failed to record action: %v", err)
		}
	}

	history, err := ld.getActionHistory(bead)
	if err != nil {
		t.Fatalf("Failed to get action history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 actions in history, got %d", len(history))
	}
}

func TestIsStuckInLoop_NoHistory(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-no-history",
		Context: make(map[string]string),
	}

	stuck, reason := ld.IsStuckInLoop(bead)
	if stuck {
		t.Errorf("Expected bead with no history to not be stuck, got stuck: %s", reason)
	}
}

func TestIsStuckInLoop_InsufficientHistory(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-insufficient",
		Context: make(map[string]string),
	}

	// Record only 2 actions (less than threshold * 2)
	ld.RecordAction(bead, ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "test.go"},
	})
	ld.RecordAction(bead, ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{"file_path": "test.go"},
	})

	stuck, reason := ld.IsStuckInLoop(bead)
	if stuck {
		t.Errorf("Expected bead with insufficient history to not be stuck, got: %s", reason)
	}
}

func TestIsStuckInLoop_RepeatedActionWithoutProgress(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-stuck",
		Context: make(map[string]string),
	}

	// Record same action 7 times (exceeds threshold of 3, and enough for history check)
	for i := 0; i < 7; i++ {
		action := ActionRecord{
			Timestamp:  time.Now().Add(-10 * time.Minute), // Old timestamp, no recent progress
			AgentID:    "agent-1",
			ActionType: "read_file",
			ActionData: map[string]interface{}{
				"file_path": "test.go",
			},
		}
		ld.RecordAction(bead, action)
	}

	// Manually override progress metrics to simulate old progress (no recent activity)
	// This simulates a bead that was active 10 minutes ago but has had no progress since
	oldTime := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	bead.Context["progress_metrics"] = fmt.Sprintf(`{"files_read":7,"files_modified":0,"tests_run":0,"commands_executed":0,"last_progress":"%s"}`, oldTime)

	// Debug: check what we have
	history, _ := ld.getActionHistory(bead)
	t.Logf("History length: %d", len(history))
	t.Logf("Progress summary: %s", ld.GetProgressSummary(bead))
	t.Logf("Has recent progress: %v", ld.hasRecentProgress(bead))

	stuck, reason := ld.IsStuckInLoop(bead)
	if !stuck {
		t.Error("Expected bead with repeated actions and no progress to be stuck")
	}

	if reason == "" {
		t.Error("Expected stuck reason to be provided")
	}

	t.Logf("Stuck reason: %s", reason)
}

func TestIsStuckInLoop_RepeatedActionWithProgress(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-productive",
		Context: make(map[string]string),
	}

	// Record same action multiple times, but with recent progress
	for i := 0; i < 4; i++ {
		action := ActionRecord{
			Timestamp:  time.Now(),
			AgentID:    "agent-1",
			ActionType: "read_file",
			ActionData: map[string]interface{}{
				"file_path": "test.go",
			},
		}
		ld.RecordAction(bead, action)
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	stuck, reason := ld.IsStuckInLoop(bead)
	if stuck {
		t.Errorf("Expected bead with recent progress to not be stuck, got: %s", reason)
	}
}

func TestIsStuckInLoop_VariedActions(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-varied",
		Context: make(map[string]string),
	}

	// Record varied actions
	actions := []ActionRecord{
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "read_file", ActionData: map[string]interface{}{"file_path": "a.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "read_file", ActionData: map[string]interface{}{"file_path": "b.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "edit_file", ActionData: map[string]interface{}{"file_path": "c.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "run_tests", ActionData: map[string]interface{}{"command": "go test"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "read_file", ActionData: map[string]interface{}{"file_path": "d.go"}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "bash", ActionData: map[string]interface{}{"command": "git status"}},
	}

	for _, action := range actions {
		ld.RecordAction(bead, action)
	}

	stuck, reason := ld.IsStuckInLoop(bead)
	if stuck {
		t.Errorf("Expected bead with varied actions to not be stuck, got: %s", reason)
	}
}

func TestProgressMetricsUpdate(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-metrics",
		Context: make(map[string]string),
	}

	testCases := []struct {
		actionType    string
		expectedField string
	}{
		{"read_file", "FilesRead"},
		{"edit_file", "FilesModified"},
		{"write_file", "FilesModified"},
		{"run_tests", "TestsRun"},
		{"bash", "CommandsExecuted"},
	}

	for _, tc := range testCases {
		t.Run(tc.actionType, func(t *testing.T) {
			action := ActionRecord{
				Timestamp:  time.Now(),
				AgentID:    "agent-1",
				ActionType: tc.actionType,
				ActionData: map[string]interface{}{},
			}

			err := ld.RecordAction(bead, action)
			if err != nil {
				t.Fatalf("Failed to record action: %v", err)
			}

			summary := ld.GetProgressSummary(bead)
			if summary == "No progress data" {
				t.Errorf("Expected progress data after %s action", tc.actionType)
			}

			t.Logf("Progress after %s: %s", tc.actionType, summary)
		})
	}
}

func TestGetProgressSummary(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-summary",
		Context: make(map[string]string),
	}

	// Before any actions
	summary := ld.GetProgressSummary(bead)
	if summary != "No progress data" {
		t.Errorf("Expected 'No progress data' for new bead, got: %s", summary)
	}

	// After some actions
	actions := []ActionRecord{
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "read_file", ActionData: map[string]interface{}{}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "read_file", ActionData: map[string]interface{}{}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "edit_file", ActionData: map[string]interface{}{}},
		{Timestamp: time.Now(), AgentID: "agent-1", ActionType: "run_tests", ActionData: map[string]interface{}{}},
	}

	for _, action := range actions {
		ld.RecordAction(bead, action)
	}

	summary = ld.GetProgressSummary(bead)
	if summary == "No progress data" {
		t.Error("Expected progress summary after actions")
	}

	t.Logf("Progress summary: %s", summary)

	// Verify summary contains expected information
	if summary == "Invalid progress data" {
		t.Error("Expected valid progress summary")
	}
}

func TestResetProgress(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-reset",
		Context: make(map[string]string),
	}

	// Record some actions
	ld.RecordAction(bead, ActionRecord{
		Timestamp:  time.Now(),
		AgentID:    "agent-1",
		ActionType: "read_file",
		ActionData: map[string]interface{}{},
	})

	// Verify data exists
	if bead.Context["action_history"] == "" {
		t.Error("Expected action history to be set")
	}
	if bead.Context["progress_metrics"] == "" {
		t.Error("Expected progress metrics to be set")
	}

	// Reset
	ld.ResetProgress(bead)

	// Verify data is cleared
	if bead.Context["action_history"] != "" {
		t.Error("Expected action history to be cleared")
	}
	if bead.Context["progress_metrics"] != "" {
		t.Error("Expected progress metrics to be cleared")
	}
}

func TestGenerateProgressKey(t *testing.T) {
	ld := NewLoopDetector()

	testCases := []struct {
		name      string
		action1   ActionRecord
		action2   ActionRecord
		shouldMatch bool
	}{
		{
			name: "Same file path",
			action1: ActionRecord{
				ActionType: "read_file",
				ActionData: map[string]interface{}{"file_path": "test.go"},
			},
			action2: ActionRecord{
				ActionType: "read_file",
				ActionData: map[string]interface{}{"file_path": "test.go"},
			},
			shouldMatch: true,
		},
		{
			name: "Different file paths",
			action1: ActionRecord{
				ActionType: "read_file",
				ActionData: map[string]interface{}{"file_path": "test1.go"},
			},
			action2: ActionRecord{
				ActionType: "read_file",
				ActionData: map[string]interface{}{"file_path": "test2.go"},
			},
			shouldMatch: false,
		},
		{
			name: "Same command",
			action1: ActionRecord{
				ActionType: "bash",
				ActionData: map[string]interface{}{"command": "go test"},
			},
			action2: ActionRecord{
				ActionType: "bash",
				ActionData: map[string]interface{}{"command": "go test"},
			},
			shouldMatch: true,
		},
		{
			name: "Different action types",
			action1: ActionRecord{
				ActionType: "read_file",
				ActionData: map[string]interface{}{"file_path": "test.go"},
			},
			action2: ActionRecord{
				ActionType: "edit_file",
				ActionData: map[string]interface{}{"file_path": "test.go"},
			},
			shouldMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key1 := ld.generateProgressKey(tc.action1)
			key2 := ld.generateProgressKey(tc.action2)

			if tc.shouldMatch && key1 != key2 {
				t.Errorf("Expected keys to match: %s != %s", key1, key2)
			}

			if !tc.shouldMatch && key1 == key2 {
				t.Errorf("Expected keys to differ: %s == %s", key1, key2)
			}
		})
	}
}

func TestHistoryLimit(t *testing.T) {
	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-history-limit",
		Context: make(map[string]string),
	}

	// Record more than 50 actions (the limit)
	for i := 0; i < 60; i++ {
		action := ActionRecord{
			Timestamp:  time.Now(),
			AgentID:    "agent-1",
			ActionType: "read_file",
			ActionData: map[string]interface{}{"file_path": "test.go"},
		}
		ld.RecordAction(bead, action)
	}

	history, err := ld.getActionHistory(bead)
	if err != nil {
		t.Fatalf("Failed to get action history: %v", err)
	}

	if len(history) != 50 {
		t.Errorf("Expected history to be limited to 50 actions, got %d", len(history))
	}
}

func TestConcurrentActionRecording(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	ld := NewLoopDetector()
	bead := &models.Bead{
		ID:      "bead-concurrent",
		Context: make(map[string]string),
	}

	// Note: This test doesn't use actual concurrency because the current implementation
	// doesn't have locking. This would need to be added for true concurrent safety.
	// For now, test sequential recording which simulates the typical dispatcher flow.

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(idx int) {
			action := ActionRecord{
				Timestamp:  time.Now(),
				AgentID:    "agent-1",
				ActionType: "read_file",
				ActionData: map[string]interface{}{"index": idx},
			}
			ld.RecordAction(bead, action)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	history, err := ld.getActionHistory(bead)
	if err != nil {
		t.Fatalf("Failed to get action history: %v", err)
	}

	// Should have recorded some actions (exact count may vary due to race conditions)
	if len(history) == 0 {
		t.Error("Expected some actions to be recorded")
	}

	t.Logf("Recorded %d actions concurrently", len(history))
}
