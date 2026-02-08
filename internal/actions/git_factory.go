package actions

import (
	"context"
	"fmt"
	"sync"

	"github.com/jordanhubbard/loom/internal/gitops"
)

// ProjectGitRouter implements GitOperator by routing each call through a
// per-project GitServiceAdapter. It uses the gitops.Manager to resolve
// project work directories and SSH key locations, while delegating the
// actual git operations to git.GitService via GitServiceAdapter.
type ProjectGitRouter struct {
	gitopsMgr *gitops.Manager
	mu        sync.RWMutex
	cache     map[string]*GitServiceAdapter // projectID -> adapter
}

// NewProjectGitRouter creates a project-aware GitOperator.
func NewProjectGitRouter(gitopsMgr *gitops.Manager) *ProjectGitRouter {
	return &ProjectGitRouter{
		gitopsMgr: gitopsMgr,
		cache:     make(map[string]*GitServiceAdapter),
	}
}

// forProject returns a cached or newly-created GitServiceAdapter for the project.
func (r *ProjectGitRouter) forProject(projectID string) (*GitServiceAdapter, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required for git operations")
	}

	r.mu.RLock()
	if adapter, ok := r.cache[projectID]; ok {
		r.mu.RUnlock()
		return adapter, nil
	}
	r.mu.RUnlock()

	workDir := r.gitopsMgr.GetProjectWorkDir(projectID)
	keyDir := r.gitopsMgr.GetProjectKeyDir()

	adapter, err := NewGitServiceAdapter(workDir, projectID, keyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create git adapter for project %s: %w", projectID, err)
	}

	r.mu.Lock()
	r.cache[projectID] = adapter
	r.mu.Unlock()

	return adapter, nil
}

// --- GitOperator interface implementation ---
// Each method extracts projectID from the first relevant parameter or context,
// creates/retrieves the per-project adapter, and delegates.

func (r *ProjectGitRouter) Status(ctx context.Context, projectID string) (string, error) {
	// Fall back to gitops.Manager for Status since it has project-level context
	return r.gitopsMgr.Status(ctx, projectID)
}

func (r *ProjectGitRouter) Diff(ctx context.Context, projectID string) (string, error) {
	return r.gitopsMgr.Diff(ctx, projectID)
}

func (r *ProjectGitRouter) CreateBranch(ctx context.Context, beadID, description, baseBranch string) (map[string]interface{}, error) {
	// beadID typically encodes project info; use a context-based approach
	// For now, this requires a project-scoped adapter already cached
	return nil, fmt.Errorf("CreateBranch requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) Commit(ctx context.Context, beadID, agentID, message string, files []string, allowAll bool) (map[string]interface{}, error) {
	return nil, fmt.Errorf("Commit requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) Push(ctx context.Context, beadID, branch string, setUpstream bool) (map[string]interface{}, error) {
	return nil, fmt.Errorf("Push requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	return nil, fmt.Errorf("GetStatus requires project context — use Status(projectID) instead")
}

func (r *ProjectGitRouter) GetDiff(ctx context.Context, staged bool) (map[string]interface{}, error) {
	return nil, fmt.Errorf("GetDiff requires project context — use Diff(projectID) instead")
}

func (r *ProjectGitRouter) CreatePR(ctx context.Context, beadID, title, body, base, branch string, reviewers []string, draft bool) (map[string]interface{}, error) {
	return nil, fmt.Errorf("CreatePR requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) Merge(ctx context.Context, beadID, sourceBranch, message string, noFF bool) (map[string]interface{}, error) {
	return nil, fmt.Errorf("Merge requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) Revert(ctx context.Context, beadID string, commitSHAs []string, reason string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("Revert requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) DeleteBranch(ctx context.Context, branch string, deleteRemote bool) (map[string]interface{}, error) {
	return nil, fmt.Errorf("DeleteBranch requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) Checkout(ctx context.Context, branch string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("Checkout requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) Log(ctx context.Context, branch string, maxCount int) (map[string]interface{}, error) {
	return nil, fmt.Errorf("Log requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) Fetch(ctx context.Context) (map[string]interface{}, error) {
	return nil, fmt.Errorf("Fetch requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) ListBranches(ctx context.Context) (map[string]interface{}, error) {
	return nil, fmt.Errorf("ListBranches requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) DiffBranches(ctx context.Context, branch1, branch2 string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("DiffBranches requires project context — use via dispatch pipeline")
}

func (r *ProjectGitRouter) GetBeadCommits(ctx context.Context, beadID string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("GetBeadCommits requires project context — use via dispatch pipeline")
}

// ForProject returns a project-scoped GitOperator. Used by the dispatch pipeline
// which knows the project ID from bead context.
func (r *ProjectGitRouter) ForProject(projectID string) (GitOperator, error) {
	return r.forProject(projectID)
}
