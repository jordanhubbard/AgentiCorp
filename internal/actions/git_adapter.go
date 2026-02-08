package actions

import (
	"context"

	"github.com/jordanhubbard/loom/internal/git"
)

// GitServiceAdapter adapts git.GitService to the actions.GitOperator interface
type GitServiceAdapter struct {
	service   *git.GitService
	projectID string
}

// NewGitServiceAdapter creates a new adapter.
// projectKeyDir is optional — if empty, the git.GitService default is used.
func NewGitServiceAdapter(projectPath, projectID string, projectKeyDir ...string) (*GitServiceAdapter, error) {
	service, err := git.NewGitService(projectPath, projectID, projectKeyDir...)
	if err != nil {
		return nil, err
	}

	return &GitServiceAdapter{
		service:   service,
		projectID: projectID,
	}, nil
}

// --- Existing operations ---

// Status returns git status for a project (delegates to adapter's project)
func (a *GitServiceAdapter) Status(_ context.Context, _ string) (string, error) {
	return a.service.GetStatus(context.Background())
}

// Diff returns git diff for a project
func (a *GitServiceAdapter) Diff(_ context.Context, _ string) (string, error) {
	return a.service.GetDiff(context.Background(), false)
}

// CreateBranch creates a new agent branch
func (a *GitServiceAdapter) CreateBranch(ctx context.Context, beadID, description, baseBranch string) (map[string]interface{}, error) {
	result, err := a.service.CreateBranch(ctx, git.CreateBranchRequest{
		BeadID:      beadID,
		Description: description,
		BaseBranch:  baseBranch,
	})
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
	result, err := a.service.Commit(ctx, git.CommitRequest{
		BeadID:   beadID,
		AgentID:  agentID,
		Message:  message,
		Files:    files,
		AllowAll: allowAll,
	})
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
	result, err := a.service.Push(ctx, git.PushRequest{
		BeadID:      beadID,
		Branch:      branch,
		SetUpstream: setUpstream,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"branch":  result.Branch,
		"remote":  result.Remote,
		"success": result.Success,
	}, nil
}

// GetStatus returns git status as structured response
func (a *GitServiceAdapter) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	status, err := a.service.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": status}, nil
}

// GetDiff returns git diff as structured response
func (a *GitServiceAdapter) GetDiff(ctx context.Context, staged bool) (map[string]interface{}, error) {
	diff, err := a.service.GetDiff(ctx, staged)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"diff": diff, "staged": staged}, nil
}

// CreatePR creates a pull request
func (a *GitServiceAdapter) CreatePR(ctx context.Context, beadID, title, body, base, branch string, reviewers []string, draft bool) (map[string]interface{}, error) {
	result, err := a.service.CreatePR(ctx, git.CreatePRRequest{
		BeadID:    beadID,
		Title:     title,
		Body:      body,
		Base:      base,
		Branch:    branch,
		Reviewers: reviewers,
		Draft:     draft,
	})
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

// --- New extended operations ---

// Merge merges a branch into current with optional --no-ff
func (a *GitServiceAdapter) Merge(ctx context.Context, beadID, sourceBranch, message string, noFF bool) (map[string]interface{}, error) {
	result, err := a.service.Merge(ctx, git.MergeRequest{
		SourceBranch: sourceBranch,
		Message:      message,
		NoFF:         noFF,
		BeadID:       beadID,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"merged_branch": result.MergedBranch,
		"commit_sha":    result.CommitSHA,
		"success":       result.Success,
	}, nil
}

// Revert reverts specific commits
func (a *GitServiceAdapter) Revert(ctx context.Context, beadID string, commitSHAs []string, reason string) (map[string]interface{}, error) {
	result, err := a.service.Revert(ctx, git.RevertRequest{
		CommitSHAs: commitSHAs,
		BeadID:     beadID,
		Reason:     reason,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"reverted_shas":  result.RevertedSHAs,
		"new_commit_sha": result.NewCommitSHA,
		"success":        result.Success,
	}, nil
}

// DeleteBranch deletes a local (and optionally remote) branch
func (a *GitServiceAdapter) DeleteBranch(ctx context.Context, branch string, deleteRemote bool) (map[string]interface{}, error) {
	result, err := a.service.DeleteBranch(ctx, git.DeleteBranchRequest{
		Branch:       branch,
		DeleteRemote: deleteRemote,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"branch":         result.Branch,
		"deleted_local":  result.DeletedLocal,
		"deleted_remote": result.DeletedRemote,
	}, nil
}

// Checkout switches to a different branch
func (a *GitServiceAdapter) Checkout(ctx context.Context, branch string) (map[string]interface{}, error) {
	result, err := a.service.Checkout(ctx, git.CheckoutRequest{
		Branch: branch,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"branch":          result.Branch,
		"previous_branch": result.PreviousBranch,
	}, nil
}

// Log returns structured commit history
func (a *GitServiceAdapter) Log(ctx context.Context, branch string, maxCount int) (map[string]interface{}, error) {
	entries, err := a.service.Log(ctx, git.LogRequest{
		Branch:   branch,
		MaxCount: maxCount,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	}, nil
}

// Fetch fetches remote refs
func (a *GitServiceAdapter) Fetch(ctx context.Context) (map[string]interface{}, error) {
	if err := a.service.Fetch(ctx); err != nil {
		return nil, err
	}
	return map[string]interface{}{"success": true}, nil
}

// ListBranches lists all local and remote branches
func (a *GitServiceAdapter) ListBranches(ctx context.Context) (map[string]interface{}, error) {
	branches, err := a.service.ListBranches(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"branches": branches,
		"count":    len(branches),
	}, nil
}

// DiffBranches returns cross-branch diff
func (a *GitServiceAdapter) DiffBranches(ctx context.Context, branch1, branch2 string) (map[string]interface{}, error) {
	diff, err := a.service.DiffBranches(ctx, git.DiffBranchesRequest{
		Branch1: branch1,
		Branch2: branch2,
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"diff":    diff,
		"branch1": branch1,
		"branch2": branch2,
	}, nil
}

// GetBeadCommits returns all commits for a bead ID
func (a *GitServiceAdapter) GetBeadCommits(ctx context.Context, beadID string) (map[string]interface{}, error) {
	commits, err := a.service.GetBeadCommits(ctx, beadID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"commits": commits,
		"count":   len(commits),
		"bead_id": beadID,
	}, nil
}
