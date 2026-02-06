package actions

import (
	"context"
	"time"

	"github.com/jordanhubbard/loom/internal/build"
)

// BuildRunnerAdapter adapts the BuildRunner to the actions interface
type BuildRunnerAdapter struct {
	runner *build.BuildRunner
}

// NewBuildRunnerAdapter creates a new adapter
func NewBuildRunnerAdapter(projectPath string) *BuildRunnerAdapter {
	return &BuildRunnerAdapter{
		runner: build.NewBuildRunner(projectPath),
	}
}

// Run executes the build and returns results as a map
func (a *BuildRunnerAdapter) Run(ctx context.Context, projectPath, buildTarget, buildCommand, framework string, timeoutSeconds int) (map[string]interface{}, error) {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeoutSeconds == 0 {
		timeout = build.DefaultBuildTimeout
	}

	req := build.BuildRequest{
		ProjectPath:  projectPath,
		BuildCommand: buildCommand,
		Framework:    framework,
		Target:       buildTarget,
		Timeout:      timeout,
		Environment:  make(map[string]string),
	}

	result, err := a.runner.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert BuildResult to map[string]interface{}
	return map[string]interface{}{
		"framework":   result.Framework,
		"success":     result.Success,
		"exit_code":   result.ExitCode,
		"errors":      convertBuildErrors(result.Errors),
		"warnings":    convertBuildErrors(result.Warnings),
		"raw_output":  result.RawOutput,
		"duration":    result.Duration.String(),
		"timed_out":   result.TimedOut,
		"error":       result.Error,
		"error_count": len(result.Errors),
	}, nil
}

// convertBuildErrors converts []build.BuildError to []map[string]interface{}
func convertBuildErrors(errors []build.BuildError) []map[string]interface{} {
	result := make([]map[string]interface{}, len(errors))
	for i, e := range errors {
		result[i] = map[string]interface{}{
			"file":    e.File,
			"line":    e.Line,
			"column":  e.Column,
			"message": e.Message,
			"type":    e.Type,
		}
	}
	return result
}
