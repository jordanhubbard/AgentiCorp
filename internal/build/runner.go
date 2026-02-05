package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// BuildError represents a single build error
type BuildError struct {
	File    string `json:"file"`    // File path relative to project
	Line    int    `json:"line"`    // Line number
	Column  int    `json:"column"`  // Column number (if available)
	Message string `json:"message"` // Error message
	Type    string `json:"type"`    // "error", "warning", "info"
}

// BuildResult contains the complete build result
type BuildResult struct {
	Framework string       `json:"framework"`  // "go", "npm", "make", etc.
	Success   bool         `json:"success"`    // True if build succeeded
	ExitCode  int          `json:"exit_code"`  // Process exit code
	Errors    []BuildError `json:"errors"`     // List of build errors
	Warnings  []BuildError `json:"warnings"`   // List of build warnings
	RawOutput string       `json:"raw_output"` // Full build output
	Duration  time.Duration `json:"duration"`  // Build time
	TimedOut  bool         `json:"timed_out"`  // Whether execution timed out
	Error     string       `json:"error"`      // Error message if execution failed
}

// BuildRequest defines parameters for build execution
type BuildRequest struct {
	ProjectPath  string            // Absolute path to project
	BuildCommand string            // Optional: override build command
	Framework    string            // Optional: specify build system (auto-detect if empty)
	Target       string            // Optional: specific build target
	Environment  map[string]string // Environment variables
	Timeout      time.Duration     // Max execution time
}

const (
	// DefaultBuildTimeout is the default maximum build execution time
	DefaultBuildTimeout = 10 * time.Minute
	// MaxBuildTimeout is the absolute maximum allowed timeout
	MaxBuildTimeout = 30 * time.Minute
)

// BuildRunner executes builds and parses results
type BuildRunner struct {
	workDir string
}

// NewBuildRunner creates a new BuildRunner instance
func NewBuildRunner(workDir string) *BuildRunner {
	return &BuildRunner{
		workDir: workDir,
	}
}

// Run executes build and returns structured results
func (r *BuildRunner) Run(ctx context.Context, req BuildRequest) (*BuildResult, error) {
	// Validate request
	if req.ProjectPath == "" {
		req.ProjectPath = r.workDir
	}

	// Validate timeout
	if req.Timeout == 0 {
		req.Timeout = DefaultBuildTimeout
	} else if req.Timeout > MaxBuildTimeout {
		req.Timeout = MaxBuildTimeout
	}

	// Auto-detect framework if not specified
	framework := req.Framework
	if framework == "" {
		detected, err := r.DetectFramework(req.ProjectPath)
		if err != nil {
			return nil, fmt.Errorf("failed to detect build framework: %w", err)
		}
		framework = detected
	}

	// Build command
	cmdArgs, err := r.BuildCommand(framework, req.ProjectPath, req.Target, req.BuildCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	// Execute build
	startTime := time.Now()
	output, exitCode, timedOut, err := r.executeCommand(timeoutCtx, cmdArgs, req.ProjectPath, req.Environment)
	duration := time.Since(startTime)

	// If execution failed completely, return error result
	if err != nil && !timedOut {
		return &BuildResult{
			Framework: framework,
			Success:   false,
			Duration:  duration,
			RawOutput: output,
			ExitCode:  exitCode,
			TimedOut:  false,
			Error:     err.Error(),
		}, nil
	}

	// Parse output based on framework
	result, err := r.parseOutput(framework, output, exitCode)
	if err != nil {
		// If parsing fails, return a basic result with raw output
		return &BuildResult{
			Framework: framework,
			Success:   exitCode == 0,
			Duration:  duration,
			RawOutput: output,
			ExitCode:  exitCode,
			TimedOut:  timedOut,
			Error:     fmt.Sprintf("failed to parse output: %v", err),
		}, nil
	}

	// Update result with execution details
	result.Duration = duration
	result.TimedOut = timedOut

	return result, nil
}

// DetectFramework auto-detects the build framework based on project structure
func (r *BuildRunner) DetectFramework(projectPath string) (string, error) {
	// Check for Go
	if r.fileExists(filepath.Join(projectPath, "go.mod")) {
		return "go", nil
	}
	if matches, _ := filepath.Glob(filepath.Join(projectPath, "*.go")); len(matches) > 0 {
		return "go", nil
	}

	// Check for Node.js/npm
	if r.fileExists(filepath.Join(projectPath, "package.json")) {
		return "npm", nil
	}

	// Check for Makefile
	if r.fileExists(filepath.Join(projectPath, "Makefile")) ||
		r.fileExists(filepath.Join(projectPath, "makefile")) {
		return "make", nil
	}

	// Check for Cargo (Rust)
	if r.fileExists(filepath.Join(projectPath, "Cargo.toml")) {
		return "cargo", nil
	}

	// Check for Maven (Java)
	if r.fileExists(filepath.Join(projectPath, "pom.xml")) {
		return "maven", nil
	}

	// Check for Gradle (Java)
	if r.fileExists(filepath.Join(projectPath, "build.gradle")) ||
		r.fileExists(filepath.Join(projectPath, "build.gradle.kts")) {
		return "gradle", nil
	}

	return "", fmt.Errorf("could not detect build framework in %s", projectPath)
}

// BuildCommand constructs the build command based on framework
func (r *BuildRunner) BuildCommand(framework, projectPath, target, customCommand string) ([]string, error) {
	// Use custom command if provided
	if customCommand != "" {
		return strings.Fields(customCommand), nil
	}

	switch framework {
	case "go":
		cmd := []string{"go", "build"}
		if target != "" {
			cmd = append(cmd, "-o", target)
		}
		cmd = append(cmd, "./...")
		return cmd, nil

	case "npm":
		cmd := []string{"npm", "run", "build"}
		if target != "" {
			cmd = []string{"npm", "run", target}
		}
		return cmd, nil

	case "make":
		cmd := []string{"make"}
		if target != "" {
			cmd = append(cmd, target)
		}
		return cmd, nil

	case "cargo":
		cmd := []string{"cargo", "build"}
		if target != "" {
			cmd = append(cmd, "--bin", target)
		}
		return cmd, nil

	case "maven":
		return []string{"mvn", "compile"}, nil

	case "gradle":
		return []string{"./gradlew", "build"}, nil

	default:
		return nil, fmt.Errorf("unsupported build framework: %s", framework)
	}
}

// executeCommand runs the build command and captures output
func (r *BuildRunner) executeCommand(ctx context.Context, cmdArgs []string, workDir string, env map[string]string) (output string, exitCode int, timedOut bool, err error) {
	if len(cmdArgs) == 0 {
		return "", 1, false, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = workDir

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture combined output
	outputBytes, err := cmd.CombinedOutput()
	output = string(outputBytes)

	// Check for timeout first
	if ctx.Err() == context.DeadlineExceeded {
		return output, 124, true, nil
	}

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return output, 1, false, err
		}
	}

	return output, exitCode, false, nil
}

// parseOutput parses build output based on framework
func (r *BuildRunner) parseOutput(framework, output string, exitCode int) (*BuildResult, error) {
	switch framework {
	case "go":
		return r.parseGoOutput(output, exitCode)
	case "npm":
		return r.parseNpmOutput(output, exitCode)
	case "make":
		return r.parseMakeOutput(output, exitCode)
	case "cargo":
		return r.parseCargoOutput(output, exitCode)
	default:
		return r.parseGenericOutput(output, exitCode, framework)
	}
}

// parseGoOutput parses Go build output
func (r *BuildRunner) parseGoOutput(output string, exitCode int) (*BuildResult, error) {
	result := &BuildResult{
		Framework: "go",
		Success:   exitCode == 0,
		RawOutput: output,
		ExitCode:  exitCode,
		Errors:    []BuildError{},
		Warnings:  []BuildError{},
	}

	// Go build error format: path/to/file.go:123:45: error message
	// Example: internal/foo/bar.go:10:2: undefined: someFunc
	re := regexp.MustCompile(`^(.+?\.go):(\d+):(\d+):\s+(.+)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 5 {
			file := matches[1]
			lineNum := parseInt(matches[2])
			col := parseInt(matches[3])
			message := matches[4]

			buildErr := BuildError{
				File:    file,
				Line:    lineNum,
				Column:  col,
				Message: message,
				Type:    "error",
			}
			result.Errors = append(result.Errors, buildErr)
		}
	}

	return result, nil
}

// parseNpmOutput parses npm build output
func (r *BuildRunner) parseNpmOutput(output string, exitCode int) (*BuildResult, error) {
	result := &BuildResult{
		Framework: "npm",
		Success:   exitCode == 0,
		RawOutput: output,
		ExitCode:  exitCode,
		Errors:    []BuildError{},
		Warnings:  []BuildError{},
	}

	// npm/webpack error format varies, try common patterns
	// Example: ERROR in ./src/app.js
	// Module not found: Error: Can't resolve 'foo'
	errorRe := regexp.MustCompile(`(?i)^ERROR\s+in\s+(.+?)$`)
	errorPrefixRe := regexp.MustCompile(`(?i)^(ERROR|Error):\s+`)
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		// Match "ERROR in ./file" pattern
		matches := errorRe.FindStringSubmatch(line)
		if len(matches) > 1 {
			file := matches[1]
			// Look ahead for error details
			message := line
			if i+1 < len(lines) {
				message = line + " " + lines[i+1]
			}

			buildErr := BuildError{
				File:    file,
				Line:    0,
				Column:  0,
				Message: strings.TrimSpace(message),
				Type:    "error",
			}
			result.Errors = append(result.Errors, buildErr)
			continue
		}

		// Match "ERROR: message" or "Error: message" pattern (but not "0 errors")
		if errorPrefixRe.MatchString(line) {
			buildErr := BuildError{
				File:    "",
				Line:    0,
				Column:  0,
				Message: strings.TrimSpace(line),
				Type:    "error",
			}
			result.Errors = append(result.Errors, buildErr)
		}
	}

	return result, nil
}

// parseMakeOutput parses Make output
func (r *BuildRunner) parseMakeOutput(output string, exitCode int) (*BuildResult, error) {
	result := &BuildResult{
		Framework: "make",
		Success:   exitCode == 0,
		RawOutput: output,
		ExitCode:  exitCode,
		Errors:    []BuildError{},
		Warnings:  []BuildError{},
	}

	// Make often wraps compiler output, try to parse GCC/Clang format
	// Example: file.c:10:5: error: 'foo' undeclared
	re := regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(error|warning):\s+(.+)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 6 {
			file := matches[1]
			lineNum := parseInt(matches[2])
			col := parseInt(matches[3])
			errType := matches[4]
			message := matches[5]

			buildErr := BuildError{
				File:    file,
				Line:    lineNum,
				Column:  col,
				Message: message,
				Type:    errType,
			}

			if errType == "error" {
				result.Errors = append(result.Errors, buildErr)
			} else {
				result.Warnings = append(result.Warnings, buildErr)
			}
		}
	}

	return result, nil
}

// parseCargoOutput parses Cargo (Rust) build output
func (r *BuildRunner) parseCargoOutput(output string, exitCode int) (*BuildResult, error) {
	result := &BuildResult{
		Framework: "cargo",
		Success:   exitCode == 0,
		RawOutput: output,
		ExitCode:  exitCode,
		Errors:    []BuildError{},
		Warnings:  []BuildError{},
	}

	// Cargo error format: error[E0XXX]: message
	//   --> src/main.rs:10:5
	re := regexp.MustCompile(`^\s*-->\s+(.+?):(\d+):(\d+)`)
	errorMsgRe := regexp.MustCompile(`^(error|warning)(?:\[[\w\d]+\])?\s*:\s*(.+)`)

	lines := strings.Split(output, "\n")
	var currentError *BuildError

	for _, line := range lines {
		// Check for error/warning message
		msgMatches := errorMsgRe.FindStringSubmatch(line)
		if len(msgMatches) == 3 {
			currentError = &BuildError{
				Type:    msgMatches[1],
				Message: msgMatches[2],
			}
			continue
		}

		// Check for location
		locMatches := re.FindStringSubmatch(line)
		if len(locMatches) == 4 && currentError != nil {
			currentError.File = locMatches[1]
			currentError.Line = parseInt(locMatches[2])
			currentError.Column = parseInt(locMatches[3])

			if currentError.Type == "error" {
				result.Errors = append(result.Errors, *currentError)
			} else {
				result.Warnings = append(result.Warnings, *currentError)
			}
			currentError = nil
		}
	}

	return result, nil
}

// parseGenericOutput provides fallback parsing for unknown build systems
func (r *BuildRunner) parseGenericOutput(output string, exitCode int, framework string) (*BuildResult, error) {
	result := &BuildResult{
		Framework: framework,
		Success:   exitCode == 0,
		RawOutput: output,
		ExitCode:  exitCode,
		Errors:    []BuildError{},
		Warnings:  []BuildError{},
	}

	// Try to parse common error patterns
	// Pattern: file:line:col: error: message
	re := regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(error|warning):\s+(.+)`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 6 {
			buildErr := BuildError{
				File:    matches[1],
				Line:    parseInt(matches[2]),
				Column:  parseInt(matches[3]),
				Type:    matches[4],
				Message: matches[5],
			}

			if buildErr.Type == "error" {
				result.Errors = append(result.Errors, buildErr)
			} else {
				result.Warnings = append(result.Warnings, buildErr)
			}
		}
	}

	return result, nil
}

// fileExists checks if a file exists
func (r *BuildRunner) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// parseInt safely parses an integer from string
func parseInt(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}
