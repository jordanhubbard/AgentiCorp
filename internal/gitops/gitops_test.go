package gitops

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir, filepath.Join(tmpDir, "keys"))
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.baseWorkDir != tmpDir {
		t.Errorf("Expected baseWorkDir %s, got %s", tmpDir, mgr.baseWorkDir)
	}

	// Verify directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Base work directory was not created")
	}
}

func TestGetProjectWorkDir(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir, filepath.Join(tmpDir, "keys"))

	projectID := "test-project"
	expected := filepath.Join(tmpDir, projectID)
	actual := mgr.GetProjectWorkDir(projectID)

	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestCloneProject(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir, filepath.Join(tmpDir, "keys"))

	project := &models.Project{
		ID:            "test-clone",
		Name:          "Test Clone",
		GitRepo:       "https://github.com/jordanhubbard/loom.git",
		Branch:        "main",
		BeadsPath:     ".beads",
		GitAuthMethod: models.GitAuthNone,
	}

	ctx := context.Background()
	err := mgr.CloneProject(ctx, project)
	if err != nil {
		t.Fatalf("CloneProject failed: %v", err)
	}

	// Verify the repo was cloned
	workDir := mgr.GetProjectWorkDir(project.ID)
	if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
		t.Error("Git repository was not cloned")
	}

	// Verify metadata was updated
	if project.WorkDir != workDir {
		t.Errorf("WorkDir not set correctly: got %s, expected %s", project.WorkDir, workDir)
	}

	if project.LastSyncAt == nil {
		t.Error("LastSyncAt was not set")
	}

	if project.LastCommitHash == "" {
		t.Error("LastCommitHash was not set")
	}
}

func TestCommitChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir, filepath.Join(tmpDir, "keys"))

	// First clone a repo
	project := &models.Project{
		ID:            "test-commit",
		Name:          "Test Commit",
		GitRepo:       "https://github.com/jordanhubbard/loom.git",
		Branch:        "main",
		BeadsPath:     ".beads",
		GitAuthMethod: models.GitAuthNone,
	}

	ctx := context.Background()
	if err := mgr.CloneProject(ctx, project); err != nil {
		t.Fatalf("CloneProject failed: %v", err)
	}

	// Get hash before making changes
	workDir := mgr.GetProjectWorkDir(project.ID)
	oldHash, err := mgr.GetCurrentCommit(workDir)
	if err != nil {
		t.Fatalf("GetCurrentCommit (before) failed: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(workDir, "test-gitops.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Commit the change
	err = mgr.CommitChanges(ctx, project, "Test commit from gitops", "Test Agent", "agent@test.local")
	if err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Check that commit hash changed
	newHash, err := mgr.GetCurrentCommit(workDir)
	if err != nil {
		t.Fatalf("GetCurrentCommit (after) failed: %v", err)
	}

	if newHash == oldHash {
		t.Errorf("Commit hash did not change after commit (old: %s, new: %s)", oldHash, newHash)
	}

	// Verify project.LastCommitHash was updated
	if project.LastCommitHash != newHash {
		t.Errorf("Project.LastCommitHash not updated (expected: %s, got: %s)", newHash, project.LastCommitHash)
	}
}

func TestCommitChangesNoChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir, filepath.Join(tmpDir, "keys"))

	project := &models.Project{
		ID:            "test-no-changes",
		Name:          "Test No Changes",
		GitRepo:       "https://github.com/jordanhubbard/loom.git",
		Branch:        "main",
		BeadsPath:     ".beads",
		GitAuthMethod: models.GitAuthNone,
	}

	ctx := context.Background()
	if err := mgr.CloneProject(ctx, project); err != nil {
		t.Fatalf("CloneProject failed: %v", err)
	}

	// Try to commit with no changes - should succeed without error
	err := mgr.CommitChanges(ctx, project, "Empty commit", "Test Agent", "agent@test.local")
	if err != nil {
		t.Errorf("CommitChanges should succeed with no changes: %v", err)
	}
}
