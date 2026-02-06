package feedback

import (
	"context"
	"fmt"
	"time"

	"github.com/jordanhubbard/loom/internal/build"
	"github.com/jordanhubbard/loom/internal/linter"
	"github.com/jordanhubbard/loom/internal/testing"
)

// FeedbackResult contains aggregated feedback from all checks
type FeedbackResult struct {
	Success     bool          `json:"success"`      // True if all checks passed
	Duration    time.Duration `json:"duration"`     // Total execution time
	Build       *BuildCheck   `json:"build"`        // Build results
	Lint        *LintCheck    `json:"lint"`         // Linter results
	Test        *TestCheck    `json:"test"`         // Test results
	Summary     string        `json:"summary"`      // Human-readable summary
	FailedPhase string        `json:"failed_phase"` // Which phase failed (if any)
}

// BuildCheck contains build verification results
type BuildCheck struct {
	Success    bool                `json:"success"`
	Duration   time.Duration       `json:"duration"`
	Framework  string              `json:"framework"`
	ErrorCount int                 `json:"error_count"`
	Errors     []build.BuildError  `json:"errors,omitempty"`
	Warnings   []build.BuildError  `json:"warnings,omitempty"`
	Skipped    bool                `json:"skipped"`     // If build was skipped
	SkipReason string              `json:"skip_reason"` // Why build was skipped
}

// LintCheck contains linter results
type LintCheck struct {
	Success        bool               `json:"success"`
	Duration       time.Duration      `json:"duration"`
	Framework      string             `json:"framework"`
	ViolationCount int                `json:"violation_count"`
	Violations     []linter.Violation `json:"violations,omitempty"`
	Skipped        bool               `json:"skipped"`
	SkipReason     string             `json:"skip_reason"`
}

// TestCheck contains test execution results
type TestCheck struct {
	Success     bool                 `json:"success"`
	Duration    time.Duration        `json:"duration"`
	Framework   string               `json:"framework"`
	Summary     testing.TestSummary  `json:"summary"`
	FailedTests []testing.TestCase   `json:"failed_tests,omitempty"`
	Skipped     bool                 `json:"skipped"`
	SkipReason  string               `json:"skip_reason"`
}

// OrchestratorConfig configures feedback orchestration
type OrchestratorConfig struct {
	ProjectPath string // Project root path

	// Phase control
	RunBuild bool // Run build verification (default: true)
	RunLint  bool // Run linter (default: true)
	RunTests bool // Run tests (default: true)

	// Stop on failure
	StopOnBuildFailure bool // Stop if build fails (default: true)
	StopOnLintFailure  bool // Stop if linter fails (default: false)

	// Framework overrides
	BuildFramework string // Override auto-detection
	LintFramework  string // Override auto-detection
	TestFramework  string // Override auto-detection

	// Filters
	LintFiles   []string // Specific files to lint
	TestPattern string   // Test pattern filter

	// Timeouts
	BuildTimeout time.Duration // Build timeout (default: 10m)
	LintTimeout  time.Duration // Lint timeout (default: 5m)
	TestTimeout  time.Duration // Test timeout (default: 5m)
}

// DefaultConfig returns default orchestrator configuration
func DefaultConfig(projectPath string) OrchestratorConfig {
	return OrchestratorConfig{
		ProjectPath:        projectPath,
		RunBuild:           true,
		RunLint:            true,
		RunTests:           true,
		StopOnBuildFailure: true,
		StopOnLintFailure:  false,
		BuildTimeout:       build.DefaultBuildTimeout,
		LintTimeout:        linter.DefaultLintTimeout,
		TestTimeout:        testing.DefaultTestTimeout,
	}
}

// Orchestrator coordinates feedback loops
type Orchestrator struct {
	buildRunner  *build.BuildRunner
	lintRunner   *linter.LinterRunner
	testRunner   *testing.TestRunner
	projectPath  string
}

// NewOrchestrator creates a new feedback orchestrator
func NewOrchestrator(projectPath string) *Orchestrator {
	return &Orchestrator{
		buildRunner: build.NewBuildRunner(projectPath),
		lintRunner:  linter.NewLinterRunner(projectPath),
		testRunner:  testing.NewTestRunner(projectPath),
		projectPath: projectPath,
	}
}

// Run executes the complete feedback loop
func (o *Orchestrator) Run(ctx context.Context, config OrchestratorConfig) (*FeedbackResult, error) {
	startTime := time.Now()

	result := &FeedbackResult{
		Success: true,
		Build:   &BuildCheck{},
		Lint:    &LintCheck{},
		Test:    &TestCheck{},
	}

	// Phase 1: Build
	if config.RunBuild {
		buildResult, err := o.runBuild(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("build phase error: %w", err)
		}
		result.Build = buildResult

		if !buildResult.Success {
			result.Success = false
			result.FailedPhase = "build"
			result.Summary = o.buildSummary(result)
			result.Duration = time.Since(startTime)

			if config.StopOnBuildFailure {
				// Skip linter and tests if build failed
				result.Lint.Skipped = true
				result.Lint.SkipReason = "build failed"
				result.Test.Skipped = true
				result.Test.SkipReason = "build failed"
				return result, nil
			}
		}
	} else {
		result.Build.Skipped = true
		result.Build.SkipReason = "disabled"
	}

	// Phase 2: Lint
	if config.RunLint {
		lintResult, err := o.runLint(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("lint phase error: %w", err)
		}
		result.Lint = lintResult

		if !lintResult.Success {
			result.Success = false
			if result.FailedPhase == "" {
				result.FailedPhase = "lint"
			}

			if config.StopOnLintFailure {
				// Skip tests if linter failed
				result.Test.Skipped = true
				result.Test.SkipReason = "linter failed"
				result.Summary = o.buildSummary(result)
				result.Duration = time.Since(startTime)
				return result, nil
			}
		}
	} else {
		result.Lint.Skipped = true
		result.Lint.SkipReason = "disabled"
	}

	// Phase 3: Test
	if config.RunTests {
		testResult, err := o.runTests(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("test phase error: %w", err)
		}
		result.Test = testResult

		if !testResult.Success {
			result.Success = false
			if result.FailedPhase == "" {
				result.FailedPhase = "test"
			}
		}
	} else {
		result.Test.Skipped = true
		result.Test.SkipReason = "disabled"
	}

	result.Duration = time.Since(startTime)
	result.Summary = o.buildSummary(result)

	return result, nil
}

// runBuild executes build verification
func (o *Orchestrator) runBuild(ctx context.Context, config OrchestratorConfig) (*BuildCheck, error) {
	startTime := time.Now()

	req := build.BuildRequest{
		ProjectPath: config.ProjectPath,
		Framework:   config.BuildFramework,
		Timeout:     config.BuildTimeout,
		Environment: make(map[string]string),
	}

	buildResult, err := o.buildRunner.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	return &BuildCheck{
		Success:    buildResult.Success,
		Duration:   time.Since(startTime),
		Framework:  buildResult.Framework,
		ErrorCount: len(buildResult.Errors),
		Errors:     buildResult.Errors,
		Warnings:   buildResult.Warnings,
	}, nil
}

// runLint executes linter
func (o *Orchestrator) runLint(ctx context.Context, config OrchestratorConfig) (*LintCheck, error) {
	startTime := time.Now()

	req := linter.LintRequest{
		ProjectPath: config.ProjectPath,
		Framework:   config.LintFramework,
		Files:       config.LintFiles,
		Timeout:     config.LintTimeout,
		Environment: make(map[string]string),
	}

	lintResult, err := o.lintRunner.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	return &LintCheck{
		Success:        lintResult.Success,
		Duration:       time.Since(startTime),
		Framework:      lintResult.Framework,
		ViolationCount: len(lintResult.Violations),
		Violations:     lintResult.Violations,
	}, nil
}

// runTests executes test suite
func (o *Orchestrator) runTests(ctx context.Context, config OrchestratorConfig) (*TestCheck, error) {
	startTime := time.Now()

	req := testing.TestRequest{
		ProjectPath: config.ProjectPath,
		Framework:   config.TestFramework,
		TestPattern: config.TestPattern,
		Timeout:     config.TestTimeout,
		Environment: make(map[string]string),
	}

	testResult, err := o.testRunner.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	// Extract failed tests
	var failedTests []testing.TestCase
	for _, test := range testResult.Tests {
		if test.Status == testing.TestFail {
			failedTests = append(failedTests, test)
		}
	}

	return &TestCheck{
		Success:     testResult.Success,
		Duration:    time.Since(startTime),
		Framework:   testResult.Framework,
		Summary:     testResult.Summary,
		FailedTests: failedTests,
	}, nil
}

// buildSummary creates a human-readable summary
func (o *Orchestrator) buildSummary(result *FeedbackResult) string {
	if result.Success {
		return fmt.Sprintf("✓ All checks passed (build: %s, lint: %s, test: %s)",
			formatDuration(result.Build.Duration),
			formatDuration(result.Lint.Duration),
			formatDuration(result.Test.Duration))
	}

	summary := fmt.Sprintf("✗ Feedback loop failed at %s phase\n", result.FailedPhase)

	// Build details
	if !result.Build.Skipped {
		if result.Build.Success {
			summary += fmt.Sprintf("  ✓ Build passed (%s)\n", formatDuration(result.Build.Duration))
		} else {
			summary += fmt.Sprintf("  ✗ Build failed with %d error(s) (%s)\n",
				result.Build.ErrorCount, formatDuration(result.Build.Duration))
			// Show first few errors
			for i, err := range result.Build.Errors {
				if i >= 3 {
					summary += fmt.Sprintf("    ... and %d more error(s)\n", len(result.Build.Errors)-3)
					break
				}
				summary += fmt.Sprintf("    - %s:%d:%d: %s\n", err.File, err.Line, err.Column, err.Message)
			}
		}
	}

	// Lint details
	if !result.Lint.Skipped {
		if result.Lint.Success {
			summary += fmt.Sprintf("  ✓ Lint passed (%s)\n", formatDuration(result.Lint.Duration))
		} else {
			summary += fmt.Sprintf("  ✗ Lint failed with %d violation(s) (%s)\n",
				result.Lint.ViolationCount, formatDuration(result.Lint.Duration))
			// Show first few violations
			for i, v := range result.Lint.Violations {
				if i >= 3 {
					summary += fmt.Sprintf("    ... and %d more violation(s)\n", len(result.Lint.Violations)-3)
					break
				}
				summary += fmt.Sprintf("    - %s:%d: [%s] %s\n", v.File, v.Line, v.Rule, v.Message)
			}
		}
	} else if result.Lint.Skipped {
		summary += fmt.Sprintf("  ⊘ Lint skipped (%s)\n", result.Lint.SkipReason)
	}

	// Test details
	if !result.Test.Skipped {
		if result.Test.Success {
			summary += fmt.Sprintf("  ✓ Tests passed (%d/%d) (%s)\n",
				result.Test.Summary.Passed, result.Test.Summary.Total, formatDuration(result.Test.Duration))
		} else {
			summary += fmt.Sprintf("  ✗ Tests failed (%d/%d passed) (%s)\n",
				result.Test.Summary.Passed, result.Test.Summary.Total, formatDuration(result.Test.Duration))
			// Show first few failed tests
			for i, t := range result.Test.FailedTests {
				if i >= 3 {
					summary += fmt.Sprintf("    ... and %d more failed test(s)\n", len(result.Test.FailedTests)-3)
					break
				}
				summary += fmt.Sprintf("    - %s: %s\n", t.Name, t.Error)
			}
		}
	} else if result.Test.Skipped {
		summary += fmt.Sprintf("  ⊘ Tests skipped (%s)\n", result.Test.SkipReason)
	}

	return summary
}

// formatDuration formats duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// QuickCheck runs a fast feedback loop (lint + test, skip build)
func (o *Orchestrator) QuickCheck(ctx context.Context) (*FeedbackResult, error) {
	config := DefaultConfig(o.projectPath)
	config.RunBuild = false // Skip build for speed
	return o.Run(ctx, config)
}

// FullCheck runs complete feedback loop (build + lint + test)
func (o *Orchestrator) FullCheck(ctx context.Context) (*FeedbackResult, error) {
	config := DefaultConfig(o.projectPath)
	return o.Run(ctx, config)
}

// BuildOnly runs only build verification
func (o *Orchestrator) BuildOnly(ctx context.Context) (*FeedbackResult, error) {
	config := DefaultConfig(o.projectPath)
	config.RunLint = false
	config.RunTests = false
	return o.Run(ctx, config)
}
