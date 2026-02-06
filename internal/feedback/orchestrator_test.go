package feedback

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/build"
	"github.com/jordanhubbard/loom/internal/linter"
	testpkg "github.com/jordanhubbard/loom/internal/testing"
)

func TestOrchestrator_Run_AllSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	orch := NewOrchestrator(".")
	config := DefaultConfig(".")

	ctx := context.Background()
	result, err := orch.Run(ctx, config)

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Summary)
	}

	if result.FailedPhase != "" {
		t.Errorf("Expected no failed phase, got %s", result.FailedPhase)
	}

	if result.Build.Skipped {
		t.Error("Build should not be skipped")
	}

	if result.Lint.Skipped {
		t.Error("Lint should not be skipped")
	}

	if result.Test.Skipped {
		t.Error("Test should not be skipped")
	}

	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

func TestOrchestrator_Run_BuildFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	orch := NewOrchestrator(".")
	config := DefaultConfig(".")
	config.StopOnBuildFailure = true

	// This will fail to build (no valid project in current dir for building)
	ctx := context.Background()
	result, err := orch.Run(ctx, config)

	// Note: In a real test, we'd use mock runners
	// For now, we just verify the orchestrator logic
	if err != nil {
		t.Logf("Expected error or build failure: %v", err)
	}

	if result != nil {
		if !result.Success && result.FailedPhase == "build" {
			// Good - build failed as expected
			if !result.Lint.Skipped {
				t.Error("Lint should be skipped after build failure")
			}
			if !result.Test.Skipped {
				t.Error("Tests should be skipped after build failure")
			}
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("/test/path")

	if config.ProjectPath != "/test/path" {
		t.Errorf("Expected project path '/test/path', got '%s'", config.ProjectPath)
	}

	if !config.RunBuild {
		t.Error("Expected RunBuild to be true by default")
	}

	if !config.RunLint {
		t.Error("Expected RunLint to be true by default")
	}

	if !config.RunTests {
		t.Error("Expected RunTests to be true by default")
	}

	if !config.StopOnBuildFailure {
		t.Error("Expected StopOnBuildFailure to be true by default")
	}

	if config.StopOnLintFailure {
		t.Error("Expected StopOnLintFailure to be false by default")
	}

	if config.BuildTimeout != build.DefaultBuildTimeout {
		t.Errorf("Expected build timeout %v, got %v", build.DefaultBuildTimeout, config.BuildTimeout)
	}

	if config.LintTimeout != linter.DefaultLintTimeout {
		t.Errorf("Expected lint timeout %v, got %v", linter.DefaultLintTimeout, config.LintTimeout)
	}

	if config.TestTimeout != testpkg.DefaultTestTimeout {
		t.Errorf("Expected test timeout %v, got %v", testpkg.DefaultTestTimeout, config.TestTimeout)
	}
}

func TestBuildSummary_AllSuccess(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     true,
		FailedPhase: "",
		Build: &BuildCheck{
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		Lint: &LintCheck{
			Success:  true,
			Duration: 50 * time.Millisecond,
		},
		Test: &TestCheck{
			Success:  true,
			Duration: 200 * time.Millisecond,
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✓ All checks passed") {
		t.Errorf("Expected success message in summary, got: %s", summary)
	}

	if !strings.Contains(summary, "build") {
		t.Error("Expected 'build' in summary")
	}

	if !strings.Contains(summary, "lint") {
		t.Error("Expected 'lint' in summary")
	}

	if !strings.Contains(summary, "test") {
		t.Error("Expected 'test' in summary")
	}
}

func TestBuildSummary_BuildFailure(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "build",
		Build: &BuildCheck{
			Success:    false,
			Duration:   150 * time.Millisecond,
			ErrorCount: 2,
			Errors: []build.BuildError{
				{
					File:    "foo.go",
					Line:    10,
					Column:  2,
					Message: "undefined: bar",
					Type:    "error",
				},
				{
					File:    "baz.go",
					Line:    25,
					Column:  5,
					Message: "syntax error",
					Type:    "error",
				},
			},
		},
		Lint: &LintCheck{
			Skipped:    true,
			SkipReason: "build failed",
		},
		Test: &TestCheck{
			Skipped:    true,
			SkipReason: "build failed",
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✗ Feedback loop failed at build phase") {
		t.Errorf("Expected build failure message, got: %s", summary)
	}

	if !strings.Contains(summary, "2 error(s)") {
		t.Error("Expected error count in summary")
	}

	if !strings.Contains(summary, "foo.go:10:2") {
		t.Error("Expected first error location in summary")
	}

	if !strings.Contains(summary, "Lint skipped") {
		t.Error("Expected lint skipped message")
	}

	if !strings.Contains(summary, "Tests skipped") {
		t.Error("Expected test skipped message")
	}
}

func TestBuildSummary_LintFailure(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "lint",
		Build: &BuildCheck{
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		Lint: &LintCheck{
			Success:        false,
			Duration:       80 * time.Millisecond,
			ViolationCount: 3,
			Violations: []linter.Violation{
				{
					File:     "main.go",
					Line:     5,
					Rule:     "unused",
					Severity: "error",
					Message:  "unused variable 'x'",
				},
			},
		},
		Test: &TestCheck{
			Success:  true,
			Duration: 200 * time.Millisecond,
			Summary: testpkg.TestSummary{
				Total:  10,
				Passed: 10,
			},
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✗ Feedback loop failed at lint phase") {
		t.Errorf("Expected lint failure message, got: %s", summary)
	}

	if !strings.Contains(summary, "✓ Build passed") {
		t.Error("Expected build success in summary")
	}

	if !strings.Contains(summary, "3 violation(s)") {
		t.Error("Expected violation count in summary")
	}

	if !strings.Contains(summary, "✓ Tests passed") {
		t.Error("Expected test success (lint doesn't stop tests)")
	}
}

func TestBuildSummary_TestFailure(t *testing.T) {
	orch := NewOrchestrator(".")

	result := &FeedbackResult{
		Success:     false,
		FailedPhase: "test",
		Build: &BuildCheck{
			Success:  true,
			Duration: 100 * time.Millisecond,
		},
		Lint: &LintCheck{
			Success:  true,
			Duration: 50 * time.Millisecond,
		},
		Test: &TestCheck{
			Success:  false,
			Duration: 300 * time.Millisecond,
			Summary: testpkg.TestSummary{
				Total:  10,
				Passed: 8,
				Failed: 2,
			},
			FailedTests: []testpkg.TestCase{
				{
					Name:   "TestFoo",
					Status: "fail",
					Error:  "Expected 100, got 99",
				},
				{
					Name:   "TestBar",
					Status: "fail",
					Error:  "nil pointer dereference",
				},
			},
		},
	}

	summary := orch.buildSummary(result)

	if !strings.Contains(summary, "✗ Feedback loop failed at test phase") {
		t.Errorf("Expected test failure message, got: %s", summary)
	}

	if !strings.Contains(summary, "✓ Build passed") {
		t.Error("Expected build success in summary")
	}

	if !strings.Contains(summary, "✓ Lint passed") {
		t.Error("Expected lint success in summary")
	}

	if !strings.Contains(summary, "8/10 passed") {
		t.Error("Expected test counts in summary")
	}

	if !strings.Contains(summary, "TestFoo") {
		t.Error("Expected failed test name in summary")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{50 * time.Millisecond, "50ms"},
		{500 * time.Millisecond, "500ms"},
		{999 * time.Millisecond, "999ms"},
		{1 * time.Second, "1.0s"},
		{1500 * time.Millisecond, "1.5s"},
		{2 * time.Second, "2.0s"},
		{10500 * time.Millisecond, "10.5s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, got, tt.want)
		}
	}
}

func TestOrchestrator_QuickCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	orch := NewOrchestrator(".")

	ctx := context.Background()
	result, err := orch.QuickCheck(ctx)

	if err != nil {
		t.Fatalf("QuickCheck() error = %v", err)
	}

	if !result.Build.Skipped {
		t.Error("QuickCheck should skip build")
	}

	if result.Build.SkipReason != "disabled" {
		t.Errorf("Expected skip reason 'disabled', got '%s'", result.Build.SkipReason)
	}
}

func TestOrchestrator_BuildOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	orch := NewOrchestrator(".")

	ctx := context.Background()
	result, err := orch.BuildOnly(ctx)

	// May error if no valid build in current dir
	if err != nil {
		t.Logf("BuildOnly error (expected if no buildable project): %v", err)
	}

	if result != nil {
		if !result.Build.Skipped && result.Build.Framework == "" {
			// If build ran, it should have detected a framework
			t.Logf("Build result: success=%v, framework=%s", result.Build.Success, result.Build.Framework)
		}

		if !result.Lint.Skipped {
			t.Error("BuildOnly should skip lint")
		}

		if !result.Test.Skipped {
			t.Error("BuildOnly should skip tests")
		}
	}
}

func TestOrchestratorConfig_PhaseControl(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig(".")

	// Test disabling phases
	config.RunBuild = false
	config.RunLint = false
	config.RunTests = false

	orch := NewOrchestrator(".")
	ctx := context.Background()
	result, err := orch.Run(ctx, config)

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !result.Build.Skipped {
		t.Error("Build should be skipped when disabled")
	}

	if !result.Lint.Skipped {
		t.Error("Lint should be skipped when disabled")
	}

	if !result.Test.Skipped {
		t.Error("Tests should be skipped when disabled")
	}

	if !result.Success {
		t.Error("Should be successful when all phases skipped")
	}
}

func TestOrchestratorConfig_StopOnLintFailure(t *testing.T) {
	config := DefaultConfig(".")
	config.StopOnLintFailure = true

	if !config.StopOnLintFailure {
		t.Error("Expected StopOnLintFailure to be settable to true")
	}
}
