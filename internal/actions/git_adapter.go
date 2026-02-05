package actions

import (
	"context"
	"fmt"

	"github.com/jordanhubbard/agenticorp/internal/git"
)

// GitService interface for dependency injection
type GitService interface {
	CreateBranch(ctx context.Context, req git.CreateBranchRequest) (*git.CreateBranchResult, error)
	Commit(ctx context.Context, req git.CommitRequest) (*git.CommitResult, error)
	Push(ctx context.Context, req git.PushRequest) (*git.PushResult, error)
	GetStatus(ctx context.Context) (string, error)
	GetDiff(ctx context.Context, staged bool) (string, error)
}

// GitServiceAdapter adapts GitService to actions interface
type GitServiceAdapter struct {
	service *git.GitService
}

// NewGitServiceAdapter creates a new adapter
func NewGitServiceAdapter(projectPath, projectID string) (*GitServiceAdapter, error) {
	service, err := git.NewGitService(projectPath, projectID)
	if err != nil {
		return nil, err
	}

	return &GitServiceAdapter{
		service: service,
	}, nil
}

// CreateBranch creates a new agent branch
func (a *GitServiceAdapter) CreateBranch(ctx context.Context, beadID, description, baseBranch string) (map[string]interface{}, error) {
	req := git.CreateBranchRequest{
		BeadID:      beadID,
		Description: description,
		BaseBranch:  baseBranch,
	}

	result, err := a.service.CreateBranch(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"branch_name": result.BranchName,
		"created":     result.Created,
		"existed":     result.Existed,
	}, nil
}

// Commit creates a new commit with attribution
func (a *GitServiceAdapter) Commit(ctx context.Context, beadID, agentID, message string, files []string, allowAll bool) (map[string]interface{}, error) {
	req := git.CommitRequest{
		BeadID:   beadID,
		AgentID:  agentID,
		Message:  message,
		Files:    files,
		AllowAll: allowAll,
	}

	result, err := a.service.Commit(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"commit_sha":    result.CommitSHA,
		"files_changed": result.FilesChanged,
		"insertions":    result.Insertions,
		"deletions":     result.Deletions,
		"files":         result.Files,
	}, nil
}

// Push pushes commits to remote
func (a *GitServiceAdapter) Push(ctx context.Context, beadID, branch string, setUpstream bool) (map[string]interface{}, error) {
	req := git.PushRequest{
		BeadID:      beadID,
		Branch:      branch,
		SetUpstream: setUpstream,
	}

	result, err := a.service.Push(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"branch":  result.Branch,
		"remote":  result.Remote,
		"success": result.Success,
	}, nil
}

// GetStatus returns git status
func (a *GitServiceAdapter) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	status, err := a.service.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": status,
	}, nil
}

// GetDiff returns git diff
func (a *GitServiceAdapter) GetDiff(ctx context.Context, staged bool) (map[string]interface{}, error) {
	diff, err := a.service.GetDiff(ctx, staged)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"diff":   diff,
		"staged": staged,
	}, nil
}

// generateCommitMessage generates a commit message from bead context
func generateCommitMessage(beadID, agentID, title, description string) string {
	// Default format: feat/fix/refactor based on title keywords
	commitType := "feat"
	titleLower := ""
	if title != "" {
		titleLower = title
		if containsAny(titleLower, "fix", "bug", "error") {
			commitType = "fix"
		} else if containsAny(titleLower, "refactor", "cleanup", "reorganize") {
			commitType = "refactor"
		} else if containsAny(titleLower, "test") {
			commitType = "test"
		} else if containsAny(titleLower, "doc") {
			commitType = "docs"
		}
	}

	message := fmt.Sprintf("%s: %s\n\n", commitType, title)

	if description != "" {
		message += description + "\n\n"
	}

	message += fmt.Sprintf("Bead: %s\n", beadID)
	message += fmt.Sprintf("Agent: %s\n", agentID)
	message += "Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>\n"

	return message
}

// containsAny checks if string contains any of the substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			// Simple case-insensitive contains
			for i := 0; i <= len(s)-len(substr); i++ {
				match := true
				for j := 0; j < len(substr); j++ {
					c1 := s[i+j]
					c2 := substr[j]
					// Simple lowercase comparison
					if c1 >= 'A' && c1 <= 'Z' {
						c1 = c1 + 32
					}
					if c2 >= 'A' && c2 <= 'Z' {
						c2 = c2 + 32
					}
					if c1 != c2 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}
