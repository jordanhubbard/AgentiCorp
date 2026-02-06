# Feedback Loops

This document describes the feedback loop orchestration system for Loom agents. The feedback orchestrator automatically runs build → lint → test cycles after code changes, providing immediate verification feedback.

## Overview

The Feedback Orchestrator coordinates three verification phases in sequence:

1. **Build**: Verify code compiles (detects syntax errors, type errors, undefined references)
2. **Lint**: Check code quality and style (detects unused variables, formatting issues, potential bugs)
3. **Test**: Run test suite (verifies correctness, catches regressions)

The orchestrator aggregates results from all phases and provides a unified feedback report with actionable error messages.

## Architecture

### Components

**Location:** `internal/feedback/orchestrator.go`

```go
type Orchestrator struct {
    buildRunner  *build.BuildRunner
    lintRunner   *linter.LinterRunner
    testRunner   *testing.TestRunner
    projectPath  string
}
```

The orchestrator coordinates three runners:
- **BuildRunner**: Executes builds (go, npm, make, cargo, maven, gradle)
- **LinterRunner**: Runs linters (golangci-lint, eslint, pylint)
- **TestRunner**: Executes tests (go test, jest, pytest, npm test)

### Feedback Result

```go
type FeedbackResult struct {
    Success     bool          // True if all checks passed
    Duration    time.Duration // Total execution time
    Build       *BuildCheck   // Build results
    Lint        *LintCheck    // Linter results
    Test        *TestCheck    // Test results
    Summary     string        // Human-readable summary
    FailedPhase string        // Which phase failed (if any)
}
```

## Usage

### Basic Usage

```go
orch := feedback.NewOrchestrator("/path/to/project")

// Run full feedback loop
result, err := orch.FullCheck(context.Background())
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Summary)
```

### Convenience Methods

```go
// Full check: build + lint + test
result, err := orch.FullCheck(ctx)

// Quick check: lint + test (skip build for speed)
result, err := orch.QuickCheck(ctx)

// Build only: verify compilation
result, err := orch.BuildOnly(ctx)
```

### Custom Configuration

```go
config := feedback.OrchestratorConfig{
    ProjectPath: "/path/to/project",

    // Phase control
    RunBuild: true,
    RunLint:  true,
    RunTests: true,

    // Stop on failure
    StopOnBuildFailure: true,  // Don't lint/test broken code
    StopOnLintFailure:  false, // Continue to tests even if lint fails

    // Framework overrides (auto-detect by default)
    BuildFramework: "go",
    LintFramework:  "golangci-lint",
    TestFramework:  "go",

    // Filters
    LintFiles:   []string{"internal/*.go"},
    TestPattern: "TestFoo*",

    // Timeouts
    BuildTimeout: 10 * time.Minute,
    LintTimeout:  5 * time.Minute,
    TestTimeout:  5 * time.Minute,
}

result, err := orch.Run(ctx, config)
```

## Execution Flow

### Default Flow (StopOnBuildFailure=true)

```
Phase 1: Build
  ├─ Success → Continue to Phase 2
  └─ Failure → Skip lint & test, return early

Phase 2: Lint
  ├─ Success → Continue to Phase 3
  └─ Failure → Continue to Phase 3 (by default)

Phase 3: Test
  ├─ Success → Report overall success
  └─ Failure → Report test failures
```

### Stop on Lint Failure

```
config.StopOnLintFailure = true

Phase 1: Build
  └─ Success → Continue

Phase 2: Lint
  ├─ Success → Continue to Phase 3
  └─ Failure → Skip tests, return early

Phase 3: Test
  └─ Only runs if build & lint passed
```

## Feedback Result Examples

### All Phases Success

```
✓ All checks passed (build: 1.2s, lint: 0.5s, test: 2.3s)
```

### Build Failure

```
✗ Feedback loop failed at build phase
  ✗ Build failed with 2 error(s) (1.5s)
    - internal/foo.go:10:2: undefined: someFunc
    - internal/bar.go:25:5: syntax error: unexpected newline
  ⊘ Lint skipped (build failed)
  ⊘ Tests skipped (build failed)
```

### Lint Failure (Tests Continue)

```
✗ Feedback loop failed at lint phase
  ✓ Build passed (1.2s)
  ✗ Lint failed with 3 violation(s) (0.8s)
    - main.go:5: [unused] unused variable 'x'
    - utils.go:15: [golint] exported func Foo should have comment
    - parser.go:30: [errcheck] error return value not checked
  ✓ Tests passed (10/10) (2.1s)
```

### Test Failure

```
✗ Feedback loop failed at test phase
  ✓ Build passed (1.3s)
  ✓ Lint passed (0.6s)
  ✗ Tests failed (8/10 passed) (2.5s)
    - TestCalculate: Expected 100, got 99
    - TestValidate: nil pointer dereference
```

## Integration with Actions

The feedback orchestrator integrates with the agent action system to provide automatic feedback after code changes.

### Automatic Trigger Points

Feedback loops are automatically triggered after:

1. **apply_patch**: After applying code patches
2. **write_file**: After writing new files
3. **edit_code**: After editing existing files

### Action Integration

```json
{
  "actions": [
    {"type": "write_file", "path": "src/foo.go", "content": "..."},
    {"type": "feedback_check"}
  ]
}
```

**Automatic Response:**

```json
{
  "action_type": "feedback_check",
  "status": "executed",
  "metadata": {
    "success": false,
    "failed_phase": "build",
    "build": {
      "success": false,
      "error_count": 1,
      "errors": [
        {
          "file": "src/foo.go",
          "line": 10,
          "message": "undefined: bar"
        }
      ]
    }
  }
}
```

## Performance Considerations

### Phase Timing

Typical execution times (Loom project):
- **Build**: 1-3 seconds (Go build)
- **Lint**: 0.5-2 seconds (golangci-lint)
- **Test**: 2-5 seconds (unit tests)

**Total**: ~4-10 seconds for full feedback loop

### Optimization Strategies

**1. Quick Check for Rapid Iteration**

Skip build for faster feedback during development:

```go
result, err := orch.QuickCheck(ctx)  // lint + test only (~3s)
```

**2. Targeted Linting**

Lint only changed files:

```go
config.LintFiles = []string{"internal/parser.go", "internal/lexer.go"}
```

**3. Filtered Tests**

Run specific test patterns:

```go
config.TestPattern = "TestParser*"  // Only run parser tests
```

**4. Parallel Execution**

Future enhancement: Run lint and test in parallel after build succeeds.

### Timeout Configuration

```go
const (
    DefaultBuildTimeout = 10 * time.Minute
    DefaultLintTimeout  = 5 * time.Minute
    DefaultTestTimeout  = 5 * time.Minute
)
```

Adjust timeouts based on project size:
- **Small projects** (<1000 LOC): Use defaults
- **Medium projects** (1000-10000 LOC): 2-3x defaults
- **Large projects** (>10000 LOC): 5-10x defaults

## Best Practices

### 1. Always Run After Code Changes

```go
// After any code modification
if err := applyPatch(patch); err != nil {
    return err
}

result, err := orch.FullCheck(ctx)
if !result.Success {
    // Report failures to agent
    return fmt.Errorf("feedback failed: %s", result.Summary)
}
```

### 2. Stop on Build Failure

Don't waste time linting/testing code that doesn't compile:

```go
config.StopOnBuildFailure = true  // Default behavior
```

### 3. Continue on Lint Failure

Allow tests to run even if linting fails (tests may pass):

```go
config.StopOnLintFailure = false  // Default behavior
```

### 4. Use QuickCheck for Iteration

During active development, use QuickCheck for faster feedback:

```go
// Development: fast iteration
result, _ := orch.QuickCheck(ctx)

// Pre-commit: thorough verification
result, _ := orch.FullCheck(ctx)
```

### 5. Parse Errors for Automated Fixes

```go
if !result.Build.Success {
    for _, err := range result.Build.Errors {
        if strings.Contains(err.Message, "undefined:") {
            // Extract undefined symbol and suggest fix
            symbol := extractSymbol(err.Message)
            suggestImport(err.File, symbol)
        }
    }
}
```

## Error Handling

### Build Errors

```go
type BuildError struct {
    File    string  // "internal/foo.go"
    Line    int     // 10
    Column  int     // 2
    Message string  // "undefined: someFunc"
    Type    string  // "error" or "warning"
}
```

**Common Build Errors:**
- Syntax errors: Fix typos, missing punctuation
- Type errors: Add type conversions, fix signatures
- Undefined references: Add imports, fix names
- Missing dependencies: Run `go mod tidy`, `npm install`

### Lint Violations

```go
type Violation struct {
    File     string  // "main.go"
    Line     int     // 5
    Column   int     // 10
    Rule     string  // "unused"
    Severity string  // "error", "warning", "info"
    Message  string  // "unused variable 'x'"
    Linter   string  // "unused"
}
```

**Common Violations:**
- Unused code: Remove or use the code
- Style issues: Reformat code
- Potential bugs: Fix logic errors
- Documentation: Add comments

### Test Failures

```go
type TestCase struct {
    Name     string  // "TestCalculate"
    Status   string  // "fail"
    Error    string  // "Expected 100, got 99"
    Duration time.Duration
}
```

**Common Test Failures:**
- Logic errors: Fix implementation
- Edge cases: Add handling
- Race conditions: Add synchronization
- Flaky tests: Fix test assumptions

## Related Documentation

- [Build Verification](BUILD_INTEGRATION.md) - Build runner architecture
- [Linter Integration](LINTER_INTEGRATION.md) - Linter runner details
- [Test Execution](TEST_EXECUTION_DESIGN.md) - Test runner design
- [Agent Actions](AGENT_ACTIONS.md) - Complete action reference

## Implementation

The feedback orchestrator is implemented in:

- `internal/feedback/orchestrator.go` - Core orchestration logic
- `internal/feedback/orchestrator_test.go` - Unit and integration tests
- `internal/build/runner.go` - Build verification
- `internal/linter/runner.go` - Linter execution
- `internal/testing/runner.go` - Test execution

## Future Enhancements

### Phase 1 (Current)

- ✅ Sequential build → lint → test execution
- ✅ Configurable phase control
- ✅ Stop on failure options
- ✅ Aggregated feedback reporting

### Phase 2

- **Parallel execution**: Run lint and test in parallel after build
- **Incremental feedback**: Stream results as phases complete
- **Smart retries**: Auto-retry flaky tests
- **Failure analysis**: ML-powered error classification

### Phase 3

- **Automated fixes**: Suggest and apply fixes for common errors
- **Historical tracking**: Track feedback metrics over time
- **Performance profiling**: Identify slow tests and builds
- **CI/CD integration**: GitHub Actions, GitLab CI integration

## Contributing

When extending the feedback orchestrator:

1. Add new runners to the `Orchestrator` struct
2. Implement `Run()` method for new checks
3. Update `FeedbackResult` with new check results
4. Add to `buildSummary()` for user-facing output
5. Add unit tests for new functionality
6. Update this documentation

## License

See [LICENSE](../LICENSE) for details.
