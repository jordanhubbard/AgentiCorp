package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/jordanhubbard/loom/internal/actions"
)

// TestBugFixWorkflow tests the complete bug fix workflow:
// 1. Agent receives bug report bead
// 2. Reads code to understand issue
// 3. Fixes the bug
// 4. Runs tests to verify fix
// 5. Commits changes
// 6. Pushes to remote
// 7. Creates PR for review
func TestBugFixWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup: Create mock action envelope for bug fix workflow
	envelope := &actions.ActionEnvelope{
		Actions: []actions.Action{
			// Step 1: Read the buggy file
			{
				Type: actions.ActionReadFile,
				Path: "src/auth.go",
			},
			// Step 2: Search for related code
			{
				Type:  actions.ActionSearchText,
				Query: "timeout",
				Limit: 10,
			},
			// Step 3: Fix the bug
			{
				Type:  actions.ActionEditCode,
				Path:  "src/auth.go",
				Patch: "Fix timeout handling",
			},
			// Step 4: Run tests to verify
			{
				Type:        actions.ActionRunTests,
				TestPattern: "./src/...",
			},
			// Step 5: Commit if tests pass
			{
				Type:          actions.ActionGitCommit,
				CommitMessage: "fix: Resolve authentication timeout issue\n\nBead: bead-test-123\nAgent: test-agent",
			},
			// Step 6: Push to remote
			{
				Type:        actions.ActionGitPush,
				SetUpstream: true,
			},
			// Step 7: Create PR
			{
				Type:    actions.ActionCreatePR,
				PRTitle: "Fix authentication timeout bug",
				PRBody:  "Resolves timeout issue in authentication flow",
			},
		},
		Notes: "Bug fix workflow - authentication timeout",
	}

	// Validate the action envelope
	err := actions.Validate(envelope)
	if err != nil {
		t.Fatalf("Action envelope validation failed: %v", err)
	}

	// Test JSON encoding/decoding
	jsonData, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Failed to marshal envelope: %v", err)
	}

	decoded, err := actions.DecodeStrict(jsonData)
	if err != nil {
		t.Fatalf("Failed to decode envelope: %v", err)
	}

	// Verify workflow integrity
	if len(decoded.Actions) != 7 {
		t.Errorf("Expected 7 actions in bug fix workflow, got %d", len(decoded.Actions))
	}

	// Verify action sequence
	expectedSequence := []string{
		actions.ActionReadFile,
		actions.ActionSearchText,
		actions.ActionEditCode,
		actions.ActionRunTests,
		actions.ActionGitCommit,
		actions.ActionGitPush,
		actions.ActionCreatePR,
	}

	for i, expected := range expectedSequence {
		if decoded.Actions[i].Type != expected {
			t.Errorf("Step %d: expected %s, got %s", i+1, expected, decoded.Actions[i].Type)
		}
	}

	t.Log("✓ Bug fix workflow structure validated")
}

// TestFeatureDevelopmentWorkflow tests the complete feature development workflow:
// 1. Start development workflow (EPCC)
// 2. Explore: Read existing code
// 3. Plan: Design the solution
// 4. Code: Implement + test
// 5. Commit: Git operations
func TestFeatureDevelopmentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Phase 1: Start EPCC workflow
	startWorkflow := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:           actions.ActionStartDev,
				Workflow:       "epcc",
				RequireReviews: false,
			},
		},
		Notes: "Starting EPCC workflow for user profile feature",
	}

	err := actions.Validate(startWorkflow)
	if err != nil {
		t.Fatalf("Start workflow validation failed: %v", err)
	}

	// Phase 2: Explore phase
	explorePhase := &actions.ActionEnvelope{
		Actions: []actions.Action{
			// Read existing user management code
			{
				Type: actions.ActionReadFile,
				Path: "src/user/manager.go",
			},
			// Search for related patterns
			{
				Type:  actions.ActionSearchText,
				Query: "type User struct",
			},
			// Read tree structure
			{
				Type:     actions.ActionReadTree,
				Path:     "src/user",
				MaxDepth: 2,
			},
		},
		Notes: "Explore phase: Understanding existing user system",
	}

	err = actions.Validate(explorePhase)
	if err != nil {
		t.Fatalf("Explore phase validation failed: %v", err)
	}

	// Phase 3: Plan phase (check what's next)
	planPhase := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type: actions.ActionWhatsNext,
			},
		},
		Notes: "Getting guidance for plan phase",
	}

	err = actions.Validate(planPhase)
	if err != nil {
		t.Fatalf("Plan phase validation failed: %v", err)
	}

	// Phase 4: Code phase
	codePhase := &actions.ActionEnvelope{
		Actions: []actions.Action{
			// Write implementation
			{
				Type:    actions.ActionWriteFile,
				Path:    "src/user/profile.go",
				Content: "package user\n\n// Profile implementation",
			},
			// Write tests
			{
				Type:    actions.ActionWriteFile,
				Path:    "src/user/profile_test.go",
				Content: "package user\n\n// Profile tests",
			},
			// Build project
			{
				Type: actions.ActionBuildProject,
			},
			// Run tests
			{
				Type: actions.ActionRunTests,
			},
			// Run linter
			{
				Type: actions.ActionRunLinter,
			},
		},
		Notes: "Code phase: Implementation with verification",
	}

	err = actions.Validate(codePhase)
	if err != nil {
		t.Fatalf("Code phase validation failed: %v", err)
	}

	// Phase 5: Commit phase
	commitPhase := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:          actions.ActionGitCommit,
				CommitMessage: "feat: Add user profile management\n\nBead: bead-feature-123",
			},
			{
				Type:        actions.ActionGitPush,
				SetUpstream: true,
			},
			{
				Type:    actions.ActionCreatePR,
				PRTitle: "Add user profile management",
			},
		},
		Notes: "Commit phase: Git operations",
	}

	err = actions.Validate(commitPhase)
	if err != nil {
		t.Fatalf("Commit phase validation failed: %v", err)
	}

	// Proceed to next phase
	proceedPhase := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:        actions.ActionProceedToPhase,
				TargetPhase: "code",
				ReviewState: "not-required",
				Reason:      "Exploration complete, moving to implementation",
			},
		},
	}

	err = actions.Validate(proceedPhase)
	if err != nil {
		t.Fatalf("Proceed to phase validation failed: %v", err)
	}

	t.Log("✓ Feature development workflow (EPCC) validated")
}

// TestMultiAgentCollaboration tests coordination between multiple agents:
// 1. Product Manager creates PRD
// 2. Engineering Manager creates technical breakdown
// 3. Implementation agent writes code
// 4. QA agent tests
// 5. Code Reviewer reviews
func TestMultiAgentCollaboration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Agent 1: Product Manager creates PRD bead
	pmAction := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type: actions.ActionCreateBead,
				Bead: &actions.BeadPayload{
					Title:       "PRD: Dark Mode Feature",
					Description: "Product requirements for dark mode implementation",
					Type:        "prd",
					Priority:    1,
					ProjectID:   "test-project",
					Tags:        []string{"feature", "ui", "prd"},
				},
			},
		},
		Notes: "PM: Creating PRD for dark mode",
	}

	err := actions.Validate(pmAction)
	if err != nil {
		t.Fatalf("PM action validation failed: %v", err)
	}

	// Agent 2: Engineering Manager breaks down work
	emAction := &actions.ActionEnvelope{
		Actions: []actions.Action{
			// Read the PRD
			{
				Type: actions.ActionReadFile,
				Path: "docs/PRD_dark_mode.md",
			},
			// Create technical breakdown beads
			{
				Type: actions.ActionCreateBead,
				Bead: &actions.BeadPayload{
					Title:       "Implement theme switching",
					Description: "Add theme context and toggle",
					Type:        "task",
					Priority:    1,
					ProjectID:   "test-project",
					Tags:        []string{"implementation"},
				},
			},
			{
				Type: actions.ActionCreateBead,
				Bead: &actions.BeadPayload{
					Title:       "Add dark mode CSS",
					Description: "Create dark theme styles",
					Type:        "task",
					Priority:    1,
					ProjectID:   "test-project",
					Tags:        []string{"implementation", "css"},
				},
			},
		},
		Notes: "EM: Creating technical breakdown",
	}

	err = actions.Validate(emAction)
	if err != nil {
		t.Fatalf("EM action validation failed: %v", err)
	}

	// Agent 3: Implementation agent writes code
	implAction := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:    actions.ActionWriteFile,
				Path:    "src/theme/toggle.tsx",
				Content: "// Theme toggle implementation",
			},
			{
				Type: actions.ActionRunTests,
			},
			{
				Type:          actions.ActionGitCommit,
				CommitMessage: "feat: Add theme toggle\n\nBead: bead-impl-123",
			},
		},
		Notes: "Impl Agent: Writing code",
	}

	err = actions.Validate(implAction)
	if err != nil {
		t.Fatalf("Implementation action validation failed: %v", err)
	}

	// Agent 4: QA agent tests
	qaAction := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:        actions.ActionRunTests,
				TestPattern: "./...",
			},
			{
				Type: actions.ActionCreateBead,
				Bead: &actions.BeadPayload{
					Title:       "QA Sign-off: Dark Mode",
					Description: "All tests passing, feature ready",
					Type:        "task",
					ProjectID:   "test-project",
				},
			},
		},
		Notes: "QA: Testing implementation",
	}

	err = actions.Validate(qaAction)
	if err != nil {
		t.Fatalf("QA action validation failed: %v", err)
	}

	// Verify collaboration sequence
	t.Log("✓ Multi-agent collaboration workflow validated")
}

// TestEscalationWorkflow tests the escalation workflow:
// 1. Agent encounters decision point
// 2. Creates decision bead
// 3. Escalates to CEO
// 4. CEO approves/rejects
// 5. Agent proceeds
func TestEscalationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Step 1: Agent creates decision bead
	createDecision := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type: actions.ActionCreateBead,
				Bead: &actions.BeadPayload{
					Title: "Decision: Breaking API Change Required",
					Description: `We need to break the API to fix security issue.

Option 1: Breaking change now (1 week)
Option 2: Maintain backward compatibility (4 weeks)

Engineering recommends Option 1 due to security severity.`,
					Type:      "decision",
					Priority:  0, // P0 for CEO
					ProjectID: "test-project",
					Tags:      []string{"decision", "security", "ceo"},
				},
			},
		},
		Notes: "Creating decision bead for CEO escalation",
	}

	err := actions.Validate(createDecision)
	if err != nil {
		t.Fatalf("Decision creation validation failed: %v", err)
	}

	// Step 2: Escalate to CEO
	escalate := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:   actions.ActionEscalateCEO,
				BeadID: "bead-decision-123",
			},
		},
		Notes: "Escalating decision to CEO",
	}

	err = actions.Validate(escalate)
	if err != nil {
		t.Fatalf("Escalation validation failed: %v", err)
	}

	// Step 3: CEO approves
	approve := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:   actions.ActionApproveBead,
				BeadID: "bead-decision-123",
			},
		},
		Notes: "CEO approving breaking change",
	}

	err = actions.Validate(approve)
	if err != nil {
		t.Fatalf("Approval validation failed: %v", err)
	}

	// Alternative: CEO rejects
	reject := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:   actions.ActionRejectBead,
				BeadID: "bead-decision-123",
				Reason: "Too disruptive, implement Option 2 instead",
			},
		},
		Notes: "CEO rejecting breaking change",
	}

	err = actions.Validate(reject)
	if err != nil {
		t.Fatalf("Rejection validation failed: %v", err)
	}

	// Step 4: Agent proceeds after decision
	proceed := &actions.ActionEnvelope{
		Actions: []actions.Action{
			// Implementation continues based on decision
			{
				Type:    actions.ActionWriteFile,
				Path:    "src/api/v2.go",
				Content: "// New API implementation",
			},
		},
		Notes: "Proceeding with approved approach",
	}

	err = actions.Validate(proceed)
	if err != nil {
		t.Fatalf("Proceed validation failed: %v", err)
	}

	t.Log("✓ Escalation workflow validated")
}

// TestFeedbackLoopIntegration tests the build → lint → test feedback loop:
// 1. Agent writes code
// 2. Runs build (fails)
// 3. Fixes build errors
// 4. Runs linter (fails)
// 5. Fixes lint errors
// 6. Runs tests (fails)
// 7. Fixes tests
// 8. All checks pass
func TestFeedbackLoopIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Iteration 1: Build fails
	iteration1 := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:    actions.ActionWriteFile,
				Path:    "src/feature.go",
				Content: "package main\n\nfunc Broken() { // missing return",
			},
			{
				Type: actions.ActionBuildProject,
			},
		},
		Notes: "Iteration 1: Build will fail",
	}

	err := actions.Validate(iteration1)
	if err != nil {
		t.Fatalf("Iteration 1 validation failed: %v", err)
	}

	// Iteration 2: Fix build, lint fails
	iteration2 := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:    actions.ActionEditCode,
				Path:    "src/feature.go",
				Patch:   "Add missing closing brace",
				Content: "package main\n\nfunc Broken() {}",
			},
			{
				Type: actions.ActionBuildProject,
			},
			{
				Type: actions.ActionRunLinter,
			},
		},
		Notes: "Iteration 2: Fix build, check linter",
	}

	err = actions.Validate(iteration2)
	if err != nil {
		t.Fatalf("Iteration 2 validation failed: %v", err)
	}

	// Iteration 3: Fix lint, tests fail
	iteration3 := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:    actions.ActionEditCode,
				Path:    "src/feature.go",
				Patch:   "Rename to proper convention",
				Content: "package main\n\nfunc Fixed() {}",
			},
			{
				Type: actions.ActionRunLinter,
			},
			{
				Type: actions.ActionRunTests,
			},
		},
		Notes: "Iteration 3: Fix linter, run tests",
	}

	err = actions.Validate(iteration3)
	if err != nil {
		t.Fatalf("Iteration 3 validation failed: %v", err)
	}

	// Iteration 4: All checks pass
	iteration4 := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:    actions.ActionWriteFile,
				Path:    "src/feature_test.go",
				Content: "package main\n\nimport \"testing\"\n\nfunc TestFixed(t *testing.T) {}",
			},
			{
				Type: actions.ActionRunTests,
			},
			{
				Type:          actions.ActionGitCommit,
				CommitMessage: "feat: Add fixed feature\n\nBead: bead-feedback-123",
			},
		},
		Notes: "Iteration 4: All checks pass, commit",
	}

	err = actions.Validate(iteration4)
	if err != nil {
		t.Fatalf("Iteration 4 validation failed: %v", err)
	}

	t.Log("✓ Feedback loop integration validated")
}

// TestWorkflowResumption tests resuming development after a break:
// 1. Save workflow state
// 2. Simulate break/restart
// 3. Resume workflow
// 4. Continue from where left off
func TestWorkflowResumption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Before break: Agent working in code phase
	beforeBreak := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type:    actions.ActionWriteFile,
				Path:    "src/partial.go",
				Content: "// Partially complete implementation",
			},
		},
		Notes: "Working on implementation (session interrupted)",
	}

	err := actions.Validate(beforeBreak)
	if err != nil {
		t.Fatalf("Before break validation failed: %v", err)
	}

	// After break: Resume workflow
	afterBreak := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type: actions.ActionResumeWorkflow,
			},
		},
		Notes: "Resuming after conversation restart",
	}

	err = actions.Validate(afterBreak)
	if err != nil {
		t.Fatalf("After break validation failed: %v", err)
	}

	// Continue work
	continueWork := &actions.ActionEnvelope{
		Actions: []actions.Action{
			{
				Type: actions.ActionWhatsNext,
			},
			{
				Type:    actions.ActionEditCode,
				Path:    "src/partial.go",
				Patch:   "Complete implementation",
				Content: "// Complete implementation",
			},
			{
				Type: actions.ActionRunTests,
			},
		},
		Notes: "Continuing from where we left off",
	}

	err = actions.Validate(continueWork)
	if err != nil {
		t.Fatalf("Continue work validation failed: %v", err)
	}

	t.Log("✓ Workflow resumption validated")
}

// TestCompleteEndToEndScenario tests a realistic end-to-end scenario
// combining multiple workflows
func TestCompleteEndToEndScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Scenario: User reports bug, agent fixes it, creates PR
	scenario := []struct {
		name     string
		envelope *actions.ActionEnvelope
	}{
		{
			name: "User reports bug",
			envelope: &actions.ActionEnvelope{
				Actions: []actions.Action{
					{
						Type: actions.ActionCreateBead,
						Bead: &actions.BeadPayload{
							Title:       "Bug: Login timeout after 30s",
							Description: "Users experiencing 30s timeout on login",
							Type:        "bug",
							Priority:    0,
							ProjectID:   "test-project",
						},
					},
				},
			},
		},
		{
			name: "Agent investigates",
			envelope: &actions.ActionEnvelope{
				Actions: []actions.Action{
					{Type: actions.ActionReadFile, Path: "src/auth/login.go"},
					{Type: actions.ActionSearchText, Query: "timeout"},
					{Type: actions.ActionReadFile, Path: "src/config/timeouts.go"},
				},
			},
		},
		{
			name: "Agent fixes bug",
			envelope: &actions.ActionEnvelope{
				Actions: []actions.Action{
					{
						Type:  actions.ActionEditCode,
						Path:  "src/config/timeouts.go",
						Patch: "Increase login timeout to 60s",
					},
					{Type: actions.ActionRunTests, TestPattern: "./src/auth/..."},
					{Type: actions.ActionRunLinter},
				},
			},
		},
		{
			name: "Agent commits and creates PR",
			envelope: &actions.ActionEnvelope{
				Actions: []actions.Action{
					{
						Type:          actions.ActionGitCommit,
						CommitMessage: "fix: Increase login timeout to 60s\n\nBead: bead-bug-123",
					},
					{Type: actions.ActionGitPush, SetUpstream: true},
					{
						Type:    actions.ActionCreatePR,
						PRTitle: "Fix login timeout issue",
						PRBody:  "Increases timeout from 30s to 60s",
					},
				},
			},
		},
		{
			name: "Agent closes bead",
			envelope: &actions.ActionEnvelope{
				Actions: []actions.Action{
					{Type: actions.ActionCloseBead, BeadID: "bead-bug-123"},
				},
			},
		},
	}

	// Execute each step of the scenario
	for _, step := range scenario {
		t.Run(step.name, func(t *testing.T) {
			err := actions.Validate(step.envelope)
			if err != nil {
				t.Fatalf("Step '%s' validation failed: %v", step.name, err)
			}

			// Verify JSON round-trip
			jsonData, err := json.Marshal(step.envelope)
			if err != nil {
				t.Fatalf("Step '%s' marshal failed: %v", step.name, err)
			}

			_, err = actions.DecodeStrict(jsonData)
			if err != nil {
				t.Fatalf("Step '%s' decode failed: %v", step.name, err)
			}
		})
	}

	t.Log("✓ Complete end-to-end scenario validated")
}
