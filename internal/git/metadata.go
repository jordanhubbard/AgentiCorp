package git

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// CommitMetadata contains Loom-specific metadata extracted from commit trailers
type CommitMetadata struct {
	SHA       string         `json:"sha"`
	BeadID    string         `json:"bead_id"`
	AgentID   string         `json:"agent_id"`
	ProjectID string         `json:"project_id"`
	Dispatch  int            `json:"dispatch"`
	Progress  map[string]int `json:"progress,omitempty"`
	Subject   string         `json:"subject"`
	Timestamp time.Time      `json:"timestamp"`
}

// ParseCommitMetadata extracts Loom metadata from a commit message body.
// Expects trailers like:
//
//	Bead: loom-abc123
//	Agent: agent-456
//	Project: myapp
//	Dispatch: 5
//	Progress: files_modified=3, tests_run=2
func ParseCommitMetadata(commitMsg string) *CommitMetadata {
	meta := &CommitMetadata{}
	lines := strings.Split(commitMsg, "\n")

	if len(lines) > 0 {
		meta.Subject = lines[0]
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if kv := extractTrailer(line, "Bead:"); kv != "" {
			meta.BeadID = kv
		} else if kv := extractTrailer(line, "Agent:"); kv != "" {
			meta.AgentID = kv
		} else if kv := extractTrailer(line, "Project:"); kv != "" {
			meta.ProjectID = kv
		} else if kv := extractTrailer(line, "Dispatch:"); kv != "" {
			if n, err := strconv.Atoi(kv); err == nil {
				meta.Dispatch = n
			}
		} else if kv := extractTrailer(line, "Progress:"); kv != "" {
			meta.Progress = parseProgressTrailer(kv)
		}
	}

	return meta
}

// GetBeadCommits returns all commits for a specific bead ID by scanning git log trailers.
func (s *GitService) GetBeadCommits(ctx context.Context, beadID string) ([]CommitMetadata, error) {
	// Search for commits containing the bead ID in their message
	args := []string{"log", "--all", "--max-count=50",
		"--format=%H|%aI|%B%x00",
		fmt.Sprintf("--grep=Bead: %s", beadID)}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = s.projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log search failed: %w", err)
	}

	var commits []CommitMetadata
	// Split by null byte separator
	entries := strings.Split(string(output), "\x00")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// First line contains SHA|timestamp, rest is body
		parts := strings.SplitN(entry, "|", 3)
		if len(parts) < 3 {
			continue
		}

		sha := parts[0]
		timestamp, _ := time.Parse(time.RFC3339, parts[1])
		body := parts[2]

		meta := ParseCommitMetadata(body)
		meta.SHA = sha
		meta.Timestamp = timestamp

		// Only include commits that actually match this bead
		if meta.BeadID == beadID {
			commits = append(commits, *meta)
		}
	}

	s.auditLogger.LogOperation("get_bead_commits", beadID, fmt.Sprintf("found=%d", len(commits)), true, nil)
	return commits, nil
}

// extractTrailer extracts the value for a trailer key from a line.
// Returns empty string if the line does not match.
func extractTrailer(line, key string) string {
	if !strings.HasPrefix(line, key) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(line, key))
}

// parseProgressTrailer parses "key1=val1, key2=val2" into a map
func parseProgressTrailer(s string) map[string]int {
	result := make(map[string]int)
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			if n, err := strconv.Atoi(strings.TrimSpace(kv[1])); err == nil {
				result[strings.TrimSpace(kv[0])] = n
			}
		}
	}
	return result
}
