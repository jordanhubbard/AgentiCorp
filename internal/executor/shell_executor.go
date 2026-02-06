package executor

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/loom/pkg/models"
)

// allowedCommands is the allowlist of permitted commands for security
var allowedCommands = map[string]bool{
	// Build tools
	"go":    true,
	"make":  true,
	"cmake": true,

	// Package managers
	"npm":  true,
	"yarn": true,
	"pip":  true,
	"pip3": true,

	// Version control
	"git": true,
	"bd":  true,

	// Testing
	"pytest":   true,
	"jest":     true,
	"mocha":    true,
	"go test":  true, // Special case handled in parsing

	// Common utilities (read-only operations)
	"ls":   true,
	"cat":  true,
	"grep": true,
	"find": true,
	"echo": true,
	"pwd":  true,
	"date": true,
	"wc":   true,
	"head": true,
	"tail": true,
	"diff": true,
	"tree": true,

	// Docker
	"docker": true,

	// Language tools
	"node":   true,
	"python": true,
	"python3": true,
	"ruby":   true,
	"java":   true,
	"javac":  true,
	"rustc":  true,
	"cargo":  true,
}

// ShellExecutor provides shell command execution with persistent logging
type ShellExecutor struct {
	db *sql.DB
}

// NewShellExecutor creates a new shell executor
func NewShellExecutor(db *sql.DB) *ShellExecutor {
	return &ShellExecutor{
		db: db,
	}
}

// validateCommand checks if a command is allowed and returns the parsed command parts
func validateCommand(command string) ([]string, bool, error) {
	// Empty command check
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return nil, false, fmt.Errorf("empty command")
	}

	// Check for shell metacharacters that require shell interpretation
	shellMetachars := []string{"|", "&&", "||", ";", ">", "<", "&", "`", "$(", "\"", "'", "\\"}
	requiresShell := false
	for _, meta := range shellMetachars {
		if strings.Contains(command, meta) {
			requiresShell = true
			break
		}
	}

	// Parse command into fields
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return nil, false, fmt.Errorf("invalid command")
	}

	// Extract binary name (base of first part, handles paths like /usr/bin/go)
	binary := filepath.Base(parts[0])

	// Special case: "go test" is two words but a single command
	if binary == "go" && len(parts) > 1 && parts[1] == "test" {
		binary = "go test"
	}

	// Check allowlist
	if !allowedCommands[binary] {
		return nil, false, fmt.Errorf("command not allowed: %s (use one of: go, npm, git, pytest, make, docker, or common utilities)", binary)
	}

	// If requires shell, return original command as single part
	if requiresShell {
		log.Printf("[ShellExecutor] Command requires shell interpretation: %s", command)
		return []string{command}, true, nil
	}

	// Return parsed parts for direct execution
	return parts, false, nil
}

// ExecuteCommandRequest represents a shell command execution request
type ExecuteCommandRequest struct {
	AgentID    string                 `json:"agent_id"`
	BeadID     string                 `json:"bead_id"`
	ProjectID  string                 `json:"project_id"`
	Command    string                 `json:"command"`
	WorkingDir string                 `json:"working_dir"`
	Timeout    int                    `json:"timeout_seconds"` // Optional timeout in seconds (default: 300)
	Context    map[string]interface{} `json:"context"`
}

// ExecuteCommandResult represents the result of a shell command execution
type ExecuteCommandResult struct {
	ID          string    `json:"id"`
	Command     string    `json:"command"`
	ExitCode    int       `json:"exit_code"`
	Stdout      string    `json:"stdout"`
	Stderr      string    `json:"stderr"`
	Duration    int64     `json:"duration_ms"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
}

// ExecuteCommand executes a shell command and logs it to the database
func (e *ShellExecutor) ExecuteCommand(ctx context.Context, req ExecuteCommandRequest) (*ExecuteCommandResult, error) {
	if req.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Validate command against allowlist
	parts, requiresShell, err := validateCommand(req.Command)
	if err != nil {
		return nil, fmt.Errorf("command validation failed: %w", err)
	}

	// Set default timeout if not specified
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 300 // 5 minutes default
	}

	// Set default working directory
	workingDir := req.WorkingDir
	if workingDir == "" {
		workingDir = "/app/src"
	}

	// Create command log entry
	cmdLog := &models.CommandLog{
		ID:         fmt.Sprintf("cmd-%s", uuid.New().String()[:8]),
		AgentID:    req.AgentID,
		BeadID:     req.BeadID,
		ProjectID:  req.ProjectID,
		Command:    req.Command,
		WorkingDir: workingDir,
		Context:    req.Context,
		StartedAt:  time.Now(),
		CreatedAt:  time.Now(),
	}

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Execute command
	log.Printf("[ShellExecutor] Executing command for agent=%s bead=%s: %s", req.AgentID, req.BeadID, req.Command)

	var cmd *exec.Cmd
	if requiresShell {
		// Complex command requires shell interpretation (piping, redirection, etc.)
		log.Printf("[ShellExecutor] Using shell for complex command")
		cmd = exec.CommandContext(cmdCtx, "/bin/sh", "-c", parts[0])
	} else {
		// Simple command - execute directly without shell for security
		log.Printf("[ShellExecutor] Direct execution (no shell)")
		cmd = exec.CommandContext(cmdCtx, parts[0], parts[1:]...)
	}
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err = cmd.Run()
	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	// Update command log with results
	cmdLog.Stdout = stdout.String()
	cmdLog.Stderr = stderr.String()
	cmdLog.CompletedAt = endTime
	cmdLog.Duration = duration

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			cmdLog.ExitCode = exitErr.ExitCode()
		} else {
			cmdLog.ExitCode = -1
		}
	} else {
		cmdLog.ExitCode = 0
	}

	// Save to database
	contextJSON := ""
	if cmdLog.Context != nil {
		if b, err := json.Marshal(cmdLog.Context); err == nil {
			contextJSON = string(b)
		}
	}

	insertQuery := `
		INSERT INTO command_logs (id, agent_id, bead_id, project_id, command, working_dir, 
			exit_code, stdout, stderr, duration_ms, started_at, completed_at, context, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, dbErr := e.db.Exec(insertQuery,
		cmdLog.ID, cmdLog.AgentID, cmdLog.BeadID, cmdLog.ProjectID, cmdLog.Command,
		cmdLog.WorkingDir, cmdLog.ExitCode, cmdLog.Stdout, cmdLog.Stderr, cmdLog.Duration,
		cmdLog.StartedAt, cmdLog.CompletedAt, contextJSON, cmdLog.CreatedAt,
	)
	if dbErr != nil {
		log.Printf("[ShellExecutor] Warning: Failed to save command log: %v", dbErr)
	}

	// Build result
	result := &ExecuteCommandResult{
		ID:          cmdLog.ID,
		Command:     req.Command,
		ExitCode:    cmdLog.ExitCode,
		Stdout:      cmdLog.Stdout,
		Stderr:      cmdLog.Stderr,
		Duration:    duration,
		StartedAt:   startTime,
		CompletedAt: endTime,
		Success:     cmdLog.ExitCode == 0,
	}

	if err != nil {
		result.Error = err.Error()
	}

	log.Printf("[ShellExecutor] Command completed: exit_code=%d duration=%dms", cmdLog.ExitCode, duration)

	return result, nil
}

// GetCommandLogs retrieves command logs with optional filters
func (e *ShellExecutor) GetCommandLogs(filters map[string]interface{}, limit int) ([]*models.CommandLog, error) {
	var logs []*models.CommandLog

	query := "SELECT * FROM command_logs WHERE 1=1"
	args := []interface{}{}

	if agentID, ok := filters["agent_id"].(string); ok && agentID != "" {
		query += " AND agent_id = ?"
		args = append(args, agentID)
	}
	if beadID, ok := filters["bead_id"].(string); ok && beadID != "" {
		query += " AND bead_id = ?"
		args = append(args, beadID)
	}
	if projectID, ok := filters["project_id"].(string); ok && projectID != "" {
		query += " AND project_id = ?"
		args = append(args, projectID)
	}

	if limit <= 0 {
		limit = 100
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cmdLog models.CommandLog
		var contextJSON sql.NullString

		err := rows.Scan(
			&cmdLog.ID, &cmdLog.AgentID, &cmdLog.BeadID, &cmdLog.ProjectID,
			&cmdLog.Command, &cmdLog.WorkingDir, &cmdLog.ExitCode,
			&cmdLog.Stdout, &cmdLog.Stderr, &cmdLog.Duration,
			&cmdLog.StartedAt, &cmdLog.CompletedAt, &contextJSON, &cmdLog.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if contextJSON.Valid {
			if err := json.Unmarshal([]byte(contextJSON.String), &cmdLog.Context); err != nil {
				// Log unmarshal error but continue
				cmdLog.Context = nil
			}
		}

		logs = append(logs, &cmdLog)
	}

	return logs, nil
}

// GetCommandLog retrieves a single command log by ID
func (e *ShellExecutor) GetCommandLog(id string) (*models.CommandLog, error) {
	query := "SELECT * FROM command_logs WHERE id = ?"

	var cmdLog models.CommandLog
	var contextJSON sql.NullString

	err := e.db.QueryRow(query, id).Scan(
		&cmdLog.ID, &cmdLog.AgentID, &cmdLog.BeadID, &cmdLog.ProjectID,
		&cmdLog.Command, &cmdLog.WorkingDir, &cmdLog.ExitCode,
		&cmdLog.Stdout, &cmdLog.Stderr, &cmdLog.Duration,
		&cmdLog.StartedAt, &cmdLog.CompletedAt, &contextJSON, &cmdLog.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if contextJSON.Valid {
		if err := json.Unmarshal([]byte(contextJSON.String), &cmdLog.Context); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	return &cmdLog, nil
}
