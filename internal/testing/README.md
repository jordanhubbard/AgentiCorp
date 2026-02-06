# Testing Package

The `internal/testing` package provides a robust test execution framework for Loom agents. It enables agents to run tests across multiple frameworks, parse results, and iterate based on feedback.

## Features

- **Multi-Framework Support**: Go, Jest, npm, pytest (extensible)
- **Auto-Detection**: Automatically detects test framework from project structure
- **Timeout Protection**: Configurable timeouts with graceful termination
- **Output Streaming**: Real-time test output via SSE (Server-Sent Events)
- **Structured Results**: Parse test output into consistent JSON format
- **Safe Execution**: Sandbox execution with resource limits

## Quick Start

```go
import (
    "context"
    "time"
    "github.com/jordanhubbard/loom/internal/testing"
)

// Create a test runner
runner := testing.NewTestRunner("/path/to/project")

// Execute tests
result, err := runner.Run(context.Background(), testing.TestRequest{
    ProjectPath: "/path/to/project",
    Timeout:     5 * time.Minute,
})

if err != nil {
    // Handle execution error
}

// Check results
if result.Success {
    fmt.Printf("All tests passed! (%d/%d)\n", result.Summary.Passed, result.Summary.Total)
} else {
    fmt.Printf("Tests failed: %d failures out of %d tests\n",
        result.Summary.Failed, result.Summary.Total)
}
```

## Core Types

### TestRunner

The main service for executing tests.

```go
type TestRunner struct {
    // Internal fields
}

func NewTestRunner(workDir string) *TestRunner
func (r *TestRunner) Run(ctx context.Context, req TestRequest) (*TestResult, error)
func (r *TestRunner) DetectFramework(projectPath string) (string, error)
func (r *TestRunner) BuildCommand(framework, projectPath, pattern, customCommand string) ([]string, error)
```

### TestRequest

Defines parameters for test execution.

```go
type TestRequest struct {
    ProjectPath  string            // Absolute path to project
    TestCommand  string            // Optional: override test command
    Framework    string            // Optional: specify framework (auto-detect if empty)
    TestPattern  string            // Optional: run specific tests (e.g., "TestFoo*")
    Environment  map[string]string // Environment variables
    Timeout      time.Duration     // Max execution time
    StreamOutput bool              // Whether to stream output in real-time
}
```

### TestResult

Contains structured test execution results.

```go
type TestResult struct {
    Framework string        `json:"framework"`  // "go", "jest", "pytest", etc.
    Success   bool          `json:"success"`    // Overall pass/fail
    Duration  time.Duration `json:"duration"`   // Total execution time
    Tests     []TestCase    `json:"tests"`      // Individual test results
    Summary   TestSummary   `json:"summary"`    // Aggregate statistics
    RawOutput string        `json:"raw_output"` // Full command output
    ExitCode  int           `json:"exit_code"`  // Process exit code
    TimedOut  bool          `json:"timed_out"`  // Whether execution timed out
    Error     string        `json:"error"`      // Error message if execution failed
}

type TestSummary struct {
    Total   int `json:"total"`
    Passed  int `json:"passed"`
    Failed  int `json:"failed"`
    Skipped int `json:"skipped"`
}

type TestCase struct {
    Name       string        `json:"name"`
    Package    string        `json:"package"`
    Status     TestStatus    `json:"status"`
    Duration   time.Duration `json:"duration"`
    Output     string        `json:"output"`
    Error      string        `json:"error"`
    StackTrace string        `json:"stack_trace"`
}
```

## Supported Frameworks

### Go

**Auto-Detection:**
- Presence of `go.mod` file
- `*_test.go` files in project

**Command:**
```bash
go test -json ./...
```

**Example:**
```go
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/go/project",
    Framework:   "go",           // Optional - auto-detected
    TestPattern: "TestFoo",      // Run only tests matching pattern
    Timeout:     5 * time.Minute,
})
```

### Jest (JavaScript/TypeScript)

**Auto-Detection:**
- `package.json` with `jest` in dependencies

**Command:**
```bash
npm test -- --json
```

**Example:**
```go
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/js/project",
    Framework:   "jest",
    TestPattern: "should handle",
    Timeout:     5 * time.Minute,
})
```

### npm (Generic Node.js)

**Auto-Detection:**
- `package.json` without specific framework detection

**Command:**
```bash
npm test
```

### pytest (Python)

**Auto-Detection:**
- `pytest.ini`, `pyproject.toml`, or `setup.cfg`
- `test_*.py` files in project

**Command:**
```bash
pytest --json-report --json-report-file=/dev/stdout
```

**Example:**
```go
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/python/project",
    Framework:   "pytest",
    TestPattern: "test_basic",
    Timeout:     5 * time.Minute,
})
```

## Advanced Features

### Custom Test Commands

Override the default test command:

```go
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
    TestCommand: "make test",  // Custom command
    Timeout:     5 * time.Minute,
})
```

### Environment Variables

Pass environment variables to test execution:

```go
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
    Environment: map[string]string{
        "GO111MODULE": "on",
        "CGO_ENABLED": "0",
        "TEST_ENV":    "ci",
    },
    Timeout: 5 * time.Minute,
})
```

### Output Streaming

Stream test output in real-time:

```go
// Implement OutputStreamer interface
type MyStreamer struct {}

func (s *MyStreamer) Write(line string) error {
    fmt.Println("TEST:", line)
    return nil
}

func (s *MyStreamer) Close() error {
    return nil
}

// Use streamer
runner := testing.NewTestRunner("/path/to/project")
runner.SetOutputStreamer(&MyStreamer{})

result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
    Timeout:     5 * time.Minute,
})
```

### Timeout Configuration

```go
const (
    DefaultTestTimeout = 10 * time.Minute
    MaxTestTimeout     = 30 * time.Minute
)

// Use default timeout (10 minutes)
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
    // Timeout not specified - uses DefaultTestTimeout
})

// Custom timeout
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
    Timeout:     2 * time.Minute,
})

// Timeout is automatically capped at MaxTestTimeout
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
    Timeout:     60 * time.Minute,  // Will be capped at 30 minutes
})
```

## Error Handling

The TestRunner is designed to be resilient and always returns a TestResult when possible:

```go
result, err := runner.Run(ctx, req)

// Critical execution error (command not found, etc.)
if err != nil {
    log.Fatalf("Test execution failed: %v", err)
}

// Test execution completed (may have failures)
if result.Error != "" {
    log.Printf("Test execution warning: %s", result.Error)
}

if result.TimedOut {
    log.Printf("Tests timed out after %v", result.Duration)
}

if !result.Success {
    log.Printf("%d tests failed:", result.Summary.Failed)
    for _, test := range result.Tests {
        if test.Status == testing.TestFail {
            log.Printf("  - %s: %s", test.Name, test.Error)
        }
    }
}
```

## Testing

### Unit Tests

```bash
# Run all unit tests
go test ./internal/testing

# Run specific test
go test ./internal/testing -run TestTestRunner_DetectFramework
```

### Integration Tests

Integration tests create real test projects and execute them:

```bash
# Run integration tests
go test ./internal/testing -run TestIntegration

# Skip integration tests (short mode)
go test ./internal/testing -short
```

## Examples

### Example 1: Basic Test Execution

```go
package main

import (
    "context"
    "fmt"
    "github.com/jordanhubbard/loom/internal/testing"
    "time"
)

func main() {
    runner := testing.NewTestRunner(".")

    result, err := runner.Run(context.Background(), testing.TestRequest{
        ProjectPath: ".",
        Timeout:     5 * time.Minute,
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Framework: %s\n", result.Framework)
    fmt.Printf("Success: %v\n", result.Success)
    fmt.Printf("Tests: %d passed, %d failed, %d skipped\n",
        result.Summary.Passed,
        result.Summary.Failed,
        result.Summary.Skipped)
}
```

### Example 2: Handling Failures

```go
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
})

if err != nil {
    return fmt.Errorf("test execution error: %w", err)
}

if !result.Success {
    var failedTests []string
    for _, test := range result.Tests {
        if test.Status == testing.TestFail {
            failedTests = append(failedTests,
                fmt.Sprintf("%s: %s", test.Name, test.Error))
        }
    }

    return fmt.Errorf("tests failed:\n%s",
        strings.Join(failedTests, "\n"))
}
```

### Example 3: Running Specific Tests

```go
// Run only database tests
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
    TestPattern: "TestDatabase",
    Timeout:     2 * time.Minute,
})

// Run tests with custom environment
result, err := runner.Run(ctx, testing.TestRequest{
    ProjectPath: "/path/to/project",
    Environment: map[string]string{
        "DATABASE_URL": "postgres://localhost/test",
        "REDIS_URL":    "redis://localhost:6379",
    },
})
```

## Architecture

The TestRunner follows a modular architecture:

1. **Framework Detection**: Auto-detect test framework from project structure
2. **Command Building**: Construct appropriate test command for framework
3. **Execution**: Run command with timeout and resource limits
4. **Output Parsing**: Parse framework-specific output into structured format
5. **Result Aggregation**: Compile individual test results into summary

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       v
┌─────────────────┐
│   TestRunner    │
└────────┬────────┘
         │
         ├──> DetectFramework
         ├──> BuildCommand
         ├──> executeCommand ──> Process Execution
         └──> parseOutput ────> Framework Parser
                                 (Go/Jest/pytest)
```

## Future Enhancements

See `docs/TEST_EXECUTION_DESIGN.md` for planned enhancements:

- Coverage reporting
- Performance regression detection
- Parallel test execution
- Selective test running (only affected tests)
- Advanced failure analysis with actionable suggestions

## Related Documentation

- [Test Execution Design](../../docs/TEST_EXECUTION_DESIGN.md) - Complete architecture document
- [Worker Integration](../worker/README.md) - How tests integrate with agent workflow
- [Conversation Architecture](../../docs/CONVERSATION_ARCHITECTURE.md) - Multi-turn test-fix-retest loops

## Contributing

When adding support for a new test framework:

1. Add detection logic to `DetectFramework()`
2. Add command building to `BuildCommand()`
3. Implement framework-specific parser in `parseOutput()`
4. Add unit tests for the new framework
5. Add integration tests with a real project
6. Update this README with examples

## License

See [LICENSE](../../LICENSE) for details.
