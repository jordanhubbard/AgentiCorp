package actions

import (
	"context"

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

// CreatePR creates a pull request
func (a *GitServiceAdapter) CreatePR(ctx context.Context, beadID, title, body, base, branch string, reviewers []string, draft bool) (map[string]interface{}, error) {
	req := git.CreatePRRequest{
		BeadID:    beadID,
		Title:     title,
		Body:      body,
		Base:      base,
		Branch:    branch,
		Reviewers: reviewers,
		Draft:     draft,
	}

	result, err := a.service.CreatePR(ctx, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"pr_number": result.Number,
		"pr_url":    result.URL,
		"branch":    result.Branch,
		"base":      result.Base,
	}, nil
}
