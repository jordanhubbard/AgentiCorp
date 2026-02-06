# Test Execution Framework Design

**Status:** Draft
**Version:** 1.0
**Created:** 2026-02-04
**Epic:** ac-r1m - Feedback Loops & Test Integration

## Overview

This document defines the architecture for enabling agents to execute tests, parse results, and iterate based on feedback. The framework provides a safe, structured way for agents to run tests across multiple languages and frameworks, analyze failures, and automatically implement fixes.

## Goals

1. **Multi-Framework Support**: Enable testing across Go, JavaScript/TypeScript, Python, and other languages
2. **Structured Output**: Parse test results into consistent JSON format for agent consumption
3. **Safe Execution**: Sandbox test execution with timeouts and resource limits
4. **Intelligent Feedback**: Provide actionable failure analysis to guide agent fixes
5. **Iterative Workflow**: Support test-fix-retest cycles within conversation sessions
6. **Integration**: Seamlessly integrate with existing Worker/Dispatcher architecture

## Architecture Components

### 1. TestRunner Service

**Location:** `internal/testing/runner.go`

The TestRunner is responsible for detecting test frameworks, executing tests, and capturing output.

```go
type TestRunner struct {
    workDir     string
    timeout     time.Duration
    sandbox     *Sandbox
    parsers     map[string]OutputParser
}

type TestRequest struct {
    ProjectPath  string            // Absolute path to project
    TestCommand  string            // Optional: override test command
    Framework    string            // Optional: specify framework (auto-detect if empty)
    TestPattern  string            // Optional: run specific tests (e.g., "TestFoo*")
    Environment  map[string]string // Environment variables
    Timeout      time.Duration     // Max execution time
}

type TestResult struct {
    Framework    string        `json:"framework"`     // "go", "jest", "pytest", etc.
    Success      bool          `json:"success"`       // Overall pass/fail
    Duration     time.Duration `json:"duration"`      // Total execution time
    Tests        []TestCase    `json:"tests"`         // Individual test results
    Summary      TestSummary   `json:"summary"`       // Aggregate statistics
    RawOutput    string        `json:"raw_output"`    // Full command output
    ExitCode     int           `json:"exit_code"`     // Process exit code
    TimedOut     bool          `json:"timed_out"`     // Whether execution timed out
}

type TestCase struct {
    Name       string        `json:"name"`          // Test name/identifier
    Package    string        `json:"package"`       // Package/file path
    Status     TestStatus    `json:"status"`        // pass/fail/skip
    Duration   time.Duration `json:"duration"`      // Individual test time
    Output     string        `json:"output"`        // Test-specific output
    Error      string        `json:"error"`         // Error message if failed
    StackTrace string        `json:"stack_trace"`   // Stack trace if available
}

type TestStatus string

const (
    TestPass TestStatus = "pass"
    TestFail TestStatus = "fail"
    TestSkip TestStatus = "skip"
)

type TestSummary struct {
    Total   int `json:"total"`
    Passed  int `json:"passed"`
    Failed  int `json:"failed"`
    Skipped int `json:"skipped"`
}
```

**Key Methods:**

- `Run(ctx context.Context, req TestRequest) (*TestResult, error)` - Execute tests
- `DetectFramework(projectPath string) (string, error)` - Auto-detect test framework
- `BuildCommand(framework, projectPath, pattern string) ([]string, error)` - Construct test command

### 2. Output Parsers

**Location:** `internal/testing/parsers/`

Each framework has a dedicated parser to convert raw output into structured TestResult.

```go
type OutputParser interface {
    // Parse converts raw test output into structured TestResult
    Parse(output string, exitCode int) (*TestResult, error)

    // CanHandle checks if this parser can handle the given output
    CanHandle(output string) bool
}

// Implementations:
// - GoTestParser (internal/testing/parsers/go.go)
// - JestParser (internal/testing/parsers/jest.go)
// - PytestParser (internal/testing/parsers/pytest.go)
// - GenericParser (internal/testing/parsers/generic.go) - fallback
```

**Parser Detection Priority:**
1. Explicit framework specified in TestRequest
2. Project structure detection (go.mod, package.json, pytest.ini)
3. Output pattern matching (last resort)

### 3. Failure Analyzer

**Location:** `internal/testing/analyzer.go`

The FailureAnalyzer examines failed tests and provides actionable insights.

```go
type FailureAnalyzer struct {
    // Dependencies for analysis
}

type FailureAnalysis struct {
    FailedTests    []TestCase         `json:"failed_tests"`
    ErrorPatterns  []ErrorPattern     `json:"error_patterns"`
    Suggestions    []string           `json:"suggestions"`
    AffectedFiles  []string           `json:"affected_files"`
    RootCause      string             `json:"root_cause"`
}

type ErrorPattern struct {
    Type        string   `json:"type"`         // "assertion", "panic", "timeout", etc.
    Message     string   `json:"message"`      // Error message pattern
    Frequency   int      `json:"frequency"`    // How many tests failed with this
    TestNames   []string `json:"test_names"`   // Which tests failed
}

// Analyze examines test failures and provides structured feedback
func (a *FailureAnalyzer) Analyze(result *TestResult) (*FailureAnalysis, error)
```

**Analysis Strategies:**

1. **Error Pattern Detection**
   - Assertion failures (expected vs actual)
   - Nil pointer dereferences
   - Type mismatches
   - Timeout errors
   - Import/dependency errors

2. **Root Cause Inference**
   - Single function causing multiple test failures
   - Configuration issues affecting all tests
   - Environment-specific failures

3. **Actionable Suggestions**
   - "Fix assertion in function X at line Y"
   - "Check nil pointer access in method Z"
   - "Update import path for package A"

### 4. Sandbox Execution

**Location:** `internal/testing/sandbox.go`

Ensures safe test execution with resource limits.

```go
type Sandbox struct {
    maxMemory    int64         // Max memory in bytes
    maxCPU       float64       // Max CPU percentage
    allowNetwork bool          // Whether to allow network access
    timeout      time.Duration // Max execution time
}

type SandboxConfig struct {
    MaxMemory    int64
    MaxCPU       float64
    AllowNetwork bool
    Timeout      time.Duration
    WorkDir      string
    Environment  map[string]string
}

// Execute runs a command in sandboxed environment
func (s *Sandbox) Execute(ctx context.Context, cmd []string, cfg SandboxConfig) (*ExecResult, error)
```

**Security Measures:**

- Process isolation (separate process group)
- Resource limits via `ulimit` or cgroups
- Network restrictions (configurable)
- Filesystem restrictions (chroot/jail when available)
- Timeout enforcement with SIGKILL fallback

## Framework Support

### Go Testing

**Detection:**
- Presence of `go.mod` or `*.go` files
- Test files matching `*_test.go`

**Execution:**
```bash
go test -json ./...
```

**Output Format:**
Go's native JSON test output (`-json` flag) provides structured results:
```json
{"Time":"2026-02-04T...","Action":"run","Package":"github.com/user/pkg","Test":"TestFoo"}
{"Time":"2026-02-04T...","Action":"output","Package":"github.com/user/pkg","Test":"TestFoo","Output":"..."}
{"Time":"2026-02-04T...","Action":"pass","Package":"github.com/user/pkg","Test":"TestFoo","Elapsed":0.01}
```

**Parser:** `GoTestParser` converts event stream into TestResult structure.

### JavaScript/TypeScript (Jest)

**Detection:**
- `package.json` with `jest` in dependencies
- `jest.config.js` or `jest.config.ts`

**Execution:**
```bash
npm test -- --json --outputFile=results.json
# or
jest --json --outputFile=results.json
```

**Output Format:**
Jest's `--json` flag produces:
```json
{
  "success": false,
  "testResults": [
    {
      "name": "test/foo.test.js",
      "status": "failed",
      "assertionResults": [
        {
          "fullName": "Foo should bar",
          "status": "failed",
          "failureMessages": ["Expected 2 but got 3"]
        }
      ]
    }
  ]
}
```

### Python (pytest)

**Detection:**
- `pytest.ini`, `pyproject.toml`, or `setup.cfg` with pytest config
- `tests/` or `test_*.py` files

**Execution:**
```bash
pytest --json-report --json-report-file=results.json
```

**Output Format:**
With `pytest-json-report` plugin:
```json
{
  "tests": [
    {
      "nodeid": "test_foo.py::test_bar",
      "outcome": "failed",
      "duration": 0.01,
      "call": {
        "longrepr": "AssertionError: assert 2 == 3"
      }
    }
  ]
}
```

### Generic/Fallback

For frameworks without structured output, parse stdout/stderr:
- Exit code 0 = success
- Exit code != 0 = failure
- Parse common patterns ("PASS", "FAIL", "ERROR")
- Provide raw output to agent for manual interpretation

## Integration with Worker/Dispatcher

### Worker Integration

The Worker executes test requests as part of agent actions:

```go
// In internal/worker/worker.go

func (w *Worker) ExecuteTask(ctx context.Context, task *models.Task) error {
    // ... existing code ...

    // If agent requests test execution
    if action.Type == "run_tests" {
        testRunner := testing.NewTestRunner(w.workDir)
        result, err := testRunner.Run(ctx, testing.TestRequest{
            ProjectPath: task.ProjectPath,
            TestPattern: action.TestPattern,
            Timeout:     5 * time.Minute,
        })

        if err != nil {
            return fmt.Errorf("test execution failed: %w", err)
        }

        // Add test results to conversation context
        w.addTestResultToContext(result)

        // If tests failed, run failure analysis
        if !result.Success {
            analyzer := testing.NewFailureAnalyzer()
            analysis, _ := analyzer.Analyze(result)
            w.addAnalysisToContext(analysis)
        }
    }
}
```

### Agent Action Schema

New action type for test execution:

```json
{
  "action": "run_tests",
  "parameters": {
    "test_pattern": "TestFoo*",
    "framework": "go",
    "timeout_seconds": 300
  }
}
```

Response format in conversation context:

```json
{
  "test_result": {
    "success": false,
    "summary": {
      "total": 10,
      "passed": 8,
      "failed": 2,
      "skipped": 0
    },
    "failed_tests": [
      {
        "name": "TestCalculateTotal",
        "error": "Expected 100, got 99",
        "file": "calculator_test.go",
        "line": 45
      }
    ],
    "analysis": {
      "root_cause": "Off-by-one error in Calculate function",
      "suggestions": [
        "Check loop boundary in Calculate at calculator.go:23",
        "Verify edge case handling for empty input"
      ],
      "affected_files": ["internal/calculator.go"]
    }
  }
}
```

## Iterative Feedback Loop

### Workflow

1. **Agent writes/modifies code**
2. **Agent triggers test execution** via `run_tests` action
3. **TestRunner executes tests** in sandboxed environment
4. **Output parser** converts results to structured format
5. **Failure analyzer** examines failures (if any)
6. **Results added to conversation** for agent to process
7. **Agent analyzes feedback** and implements fixes
8. **Repeat** until tests pass or max iterations reached

### Conversation Context Enhancement

Test results are added to the conversation session:

```go
message := models.ChatMessage{
    Role: "system",
    Content: fmt.Sprintf("Test execution completed:\n%s", formatTestResult(result)),
    TokenCount: estimateTokens(result),
}
session.AddMessage(message.Role, message.Content, message.TokenCount)
```

**Formatted Output Example:**

```
Test execution completed:
Framework: go
Status: FAILED (8/10 passed)
Duration: 1.2s

Failed Tests:
1. TestCalculateTotal (calculator_test.go:45)
   Error: Expected 100, got 99
   Suggestion: Check loop boundary in Calculate at calculator.go:23

2. TestValidateInput (validator_test.go:30)
   Error: panic: runtime error: invalid memory address or nil pointer dereference
   Suggestion: Add nil check in Validate function
```

## Timeout and Resource Limits

### Default Limits

```go
const (
    DefaultTestTimeout = 5 * time.Minute
    MaxTestTimeout     = 15 * time.Minute
    DefaultMaxMemory   = 2 * 1024 * 1024 * 1024 // 2GB
    DefaultMaxCPU      = 2.0                     // 2 CPU cores
)
```

### Per-Framework Timeouts

Different frameworks may need different timeouts:

- **Go tests**: 5 minutes (default)
- **JavaScript (Jest)**: 5 minutes (default)
- **Python (pytest)**: 5 minutes (default)
- **Integration tests**: 15 minutes (configurable)

### Timeout Handling

```go
ctx, cancel := context.WithTimeout(context.Background(), req.Timeout)
defer cancel()

result, err := runner.Run(ctx, req)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        return &TestResult{
            Success:  false,
            TimedOut: true,
            Summary: TestSummary{Total: 0, Failed: 1},
            RawOutput: fmt.Sprintf("Test execution timed out after %s", req.Timeout),
        }, nil
    }
    return nil, err
}
```

## Error Handling

### Execution Errors

- **Command not found**: Framework not installed
- **Timeout**: Tests ran too long
- **Out of memory**: Resource limits exceeded
- **Permission denied**: Sandbox restrictions
- **Parse error**: Could not parse test output

**Strategy:** Always return TestResult with error details, never propagate errors that would crash the agent.

### Partial Results

If tests are interrupted (timeout, crash), return partial results:

```go
result := &TestResult{
    Success: false,
    Tests:   parsedTests, // Tests completed before interruption
    Summary: calculateSummary(parsedTests),
    RawOutput: output,
    Error: "Test execution interrupted: timeout",
}
```

## Configuration

### Project-Level Config

Support `.loom/testing.yaml` for project-specific settings:

```yaml
testing:
  framework: go
  command: "go test -v ./..."
  timeout: 300s
  patterns:
    - "Test*"
  exclude:
    - "TestIntegration*"  # Skip integration tests by default
  environment:
    GO111MODULE: "on"
    CGO_ENABLED: "0"
  sandbox:
    allow_network: false
    max_memory: 1GB
```

### Global Defaults

System-wide defaults in `pkg/config/config.go`:

```go
type TestingConfig struct {
    DefaultTimeout   time.Duration     `yaml:"default_timeout"`
    MaxTimeout       time.Duration     `yaml:"max_timeout"`
    AllowNetwork     bool              `yaml:"allow_network"`
    MaxMemory        int64             `yaml:"max_memory"`
    SandboxEnabled   bool              `yaml:"sandbox_enabled"`
    ParserTimeout    time.Duration     `yaml:"parser_timeout"`
}
```

## Performance Considerations

### Caching

- **Test binary caching**: Cache compiled test binaries (Go)
- **Dependency caching**: Reuse node_modules, venv
- **Result caching**: Cache results for unchanged code (future optimization)

### Parallel Execution

Support parallel test execution when framework allows:

```go
type TestRequest struct {
    // ... existing fields ...
    Parallel     bool // Enable parallel test execution
    MaxParallel  int  // Max parallel test processes
}
```

Go example:
```bash
go test -parallel 4 ./...
```

### Streaming Results

For long-running test suites, stream results as they complete:

```go
type ResultStream chan TestCase

func (r *TestRunner) RunStreaming(ctx context.Context, req TestRequest) (ResultStream, error)
```

## Failure Analysis Strategies

### Pattern-Based Analysis

Common failure patterns and their signatures:

| Pattern | Signature | Suggestion |
|---------|-----------|------------|
| Assertion failure | `expected X but got Y` | Compare expected vs actual values |
| Nil pointer | `nil pointer dereference` | Add nil checks before dereferencing |
| Index out of range | `index out of range` | Verify array/slice bounds |
| Type mismatch | `cannot use X as Y` | Check type conversions |
| Import error | `could not import` | Verify import paths |
| Timeout | `test timed out` | Optimize slow operations or increase timeout |

### Statistical Analysis

- **Flaky test detection**: Test passes sometimes, fails sometimes
- **Consistent failures**: Same test always fails
- **Cascading failures**: One failure causes many others

### Context-Aware Suggestions

Use recent code changes to provide targeted suggestions:

```go
type FailureContext struct {
    RecentChanges  []string // Files modified in last commit
    RelatedFiles   []string // Files imported by failed test
    ConversationID string   // Link to conversation session
}

func (a *FailureAnalyzer) AnalyzeWithContext(
    result *TestResult,
    context FailureContext,
) (*FailureAnalysis, error)
```

## Security and Sandboxing

### Threat Model

Protect against:
- Malicious test code
- Resource exhaustion (CPU, memory, disk)
- Network attacks
- File system tampering
- Privilege escalation

### Mitigation Strategies

1. **Process Isolation**
   ```go
   cmd := exec.CommandContext(ctx, "go", "test", "./...")
   cmd.SysProcAttr = &syscall.SysProcAttr{
       Setpgid: true, // Create new process group
   }
   ```

2. **Resource Limits (Linux)**
   ```go
   import "syscall"

   cmd.SysProcAttr = &syscall.SysProcAttr{
       // Set memory limit
       // Set CPU limit
       // Set file descriptor limit
   }
   ```

3. **Timeout Enforcement**
   - Context-based timeout (graceful)
   - SIGTERM after timeout
   - SIGKILL after timeout + grace period

4. **Network Isolation** (optional)
   - Disable network for unit tests
   - Allow network for integration tests (with approval)

5. **Read-Only Filesystem** (where possible)
   - Mount test directory as read-only
   - Provide writable temp directory for test artifacts

## Future Enhancements

### Phase 2

- **Coverage reporting**: Track test coverage changes
- **Performance regression**: Detect slow tests
- **Parallel test execution**: Run tests concurrently
- **Selective test running**: Only run tests affected by changes
- **Test generation**: Agent creates tests for untested code

### Phase 3

- **Visual diff rendering**: Show test output diffs in UI
- **Test result history**: Track test health over time
- **Flaky test detection**: Identify unreliable tests
- **Mutation testing**: Verify test effectiveness

## Example Workflows

### Workflow 1: Agent Fixes Bug

1. Agent receives bug report
2. Agent runs existing tests: `run_tests`
3. Tests fail, showing the bug
4. Agent analyzes failure, identifies root cause
5. Agent modifies code to fix bug
6. Agent runs tests again: `run_tests`
7. Tests pass, bug confirmed fixed

### Workflow 2: Test-Driven Development

1. Agent creates failing test for new feature
2. Runs tests: `run_tests` → fails as expected
3. Implements feature code
4. Runs tests: `run_tests` → still fails
5. Analyzes failure, refines implementation
6. Runs tests: `run_tests` → passes
7. Refactors code while keeping tests green

### Workflow 3: Debugging

1. Tests fail in CI
2. Agent retrieves test results
3. Agent adds debug logging to suspected code
4. Runs tests locally: `run_tests`
5. Analyzes detailed output
6. Identifies issue
7. Fixes and verifies: `run_tests`

## Implementation Plan

### Phase 1: Core Framework (ac-r1m.2)

- [ ] Implement TestRunner service
- [ ] Implement GoTestParser
- [ ] Basic Sandbox execution
- [ ] Integration with Worker

### Phase 2: Multi-Framework (ac-r1m.3, ac-r1m.4)

- [ ] Implement JestParser
- [ ] Implement PytestParser
- [ ] Add ActionRunTests to agent schema
- [ ] Implement linter integration

### Phase 3: Analysis & Feedback (ac-r1m.5, ac-r1m.6)

- [ ] Implement FailureAnalyzer
- [ ] Pattern-based error detection
- [ ] Suggestion generation
- [ ] Feedback loop orchestration

## Design Review Checklist

- [x] Multi-framework support (Go, Jest, pytest)
- [x] Structured output parsing (JSON)
- [x] Failure analysis strategy
- [x] Timeout and sandboxing requirements
- [x] Integration with Worker/Dispatcher
- [x] Security considerations
- [x] Error handling strategy
- [x] Configuration options
- [x] Example workflows
- [x] Implementation roadmap

## References

- Go Testing: https://pkg.go.dev/testing
- Jest JSON Reporter: https://jestjs.io/docs/cli#--json
- Pytest JSON Report: https://github.com/numirias/pytest-json-report
- Process Sandboxing: Linux namespaces, cgroups
- Loom Worker Architecture: `internal/worker/worker.go`
- Conversation Context: `docs/CONVERSATION_ARCHITECTURE.md`
