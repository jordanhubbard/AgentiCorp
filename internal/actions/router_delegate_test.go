package actions

import (
	"context"
	"fmt"
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBeadCreator struct {
	createdBeads []*models.Bead
	createError  error
}

func (m *mockBeadCreator) CreateBead(title, description string, priority models.BeadPriority, beadType, projectID string) (*models.Bead, error) {
	if m.createError != nil {
		return nil, m.createError
	}

	bead := &models.Bead{
		ID:          fmt.Sprintf("bead-child-%d", len(m.createdBeads)+1),
		Title:       title,
		Description: description,
		Priority:    priority,
		Type:        beadType,
		ProjectID:   projectID,
		Status:      "pending",
	}

	m.createdBeads = append(m.createdBeads, bead)
	return bead, nil
}

func TestHandleDelegateTask_Success(t *testing.T) {
	mockBeads := &mockBeadCreator{}
	router := &Router{Beads: mockBeads}

	action := Action{
		Type:            ActionDelegateTask,
		DelegateToRole:  "qa-engineer",
		TaskTitle:       "Run integration tests",
		TaskDescription: "Execute full integration test suite for authentication module",
		TaskPriority:    2,
	}

	actx := ActionContext{
		AgentID:   "agent-eng-1",
		BeadID:    "bead-parent-123",
		ProjectID: "project-1",
	}

	result := router.handleDelegateTask(context.Background(), action, actx)

	assert.Equal(t, ActionDelegateTask, result.ActionType)
	assert.Equal(t, "executed", result.Status)
	assert.Contains(t, result.Message, "Run integration tests")
	assert.Contains(t, result.Message, "qa-engineer")

	// Verify metadata
	assert.Equal(t, "bead-parent-123", result.Metadata["parent_bead_id"])
	assert.Equal(t, "qa-engineer", result.Metadata["delegate_to_role"])
	assert.Equal(t, "Run integration tests", result.Metadata["task_title"])
	assert.Equal(t, 2, result.Metadata["task_priority"])

	// Verify bead was created
	require.Len(t, mockBeads.createdBeads, 1)
	childBead := mockBeads.createdBeads[0]
	assert.Equal(t, "Run integration tests", childBead.Title)
	assert.Equal(t, "Execute full integration test suite for authentication module", childBead.Description)
	assert.Equal(t, models.BeadPriority(2), childBead.Priority)
	assert.Equal(t, "delegated", childBead.Type)
	assert.Equal(t, "project-1", childBead.ProjectID)
}

func TestHandleDelegateTask_DefaultPriority(t *testing.T) {
	mockBeads := &mockBeadCreator{}
	router := &Router{Beads: mockBeads}

	action := Action{
		Type:           ActionDelegateTask,
		DelegateToRole: "code-reviewer",
		TaskTitle:      "Review PR",
		// TaskPriority not set - defaults to 0 (P0) since that's the zero value
		// This is intentional as we can't distinguish "not set" from "P0"
	}

	actx := ActionContext{
		AgentID:   "agent-eng-1",
		ProjectID: "project-1",
	}

	result := router.handleDelegateTask(context.Background(), action, actx)

	assert.Equal(t, "executed", result.Status)
	assert.Equal(t, 0, result.Metadata["task_priority"])

	// Verify bead was created with P0 priority (zero value)
	require.Len(t, mockBeads.createdBeads, 1)
	assert.Equal(t, models.BeadPriority(0), mockBeads.createdBeads[0].Priority)
}

func TestHandleDelegateTask_InvalidPriority(t *testing.T) {
	mockBeads := &mockBeadCreator{}
	router := &Router{Beads: mockBeads}

	tests := []struct {
		name     string
		priority int
		want     int
	}{
		{
			name:     "negative priority becomes 0",
			priority: -1,
			want:     0,
		},
		{
			name:     "too high priority defaults to 2",
			priority: 10,
			want:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{
				Type:           ActionDelegateTask,
				DelegateToRole: "qa-engineer",
				TaskTitle:      "Test Task",
				TaskPriority:   tt.priority,
			}

			actx := ActionContext{
				AgentID:   "agent-1",
				ProjectID: "project-1",
			}

			result := router.handleDelegateTask(context.Background(), action, actx)

			assert.Equal(t, "executed", result.Status)
			assert.Equal(t, tt.want, result.Metadata["task_priority"])
		})
	}
}

func TestHandleDelegateTask_WithExplicitParent(t *testing.T) {
	mockBeads := &mockBeadCreator{}
	router := &Router{Beads: mockBeads}

	action := Action{
		Type:           ActionDelegateTask,
		DelegateToRole: "qa-engineer",
		TaskTitle:      "Test Task",
		ParentBeadID:   "bead-explicit-parent",
	}

	actx := ActionContext{
		AgentID:   "agent-1",
		BeadID:    "bead-current-123",
		ProjectID: "project-1",
	}

	result := router.handleDelegateTask(context.Background(), action, actx)

	assert.Equal(t, "executed", result.Status)
	assert.Equal(t, "bead-explicit-parent", result.Metadata["parent_bead_id"])
	assert.Contains(t, result.Message, "bead-explicit-parent")
}

func TestHandleDelegateTask_ValidationErrors(t *testing.T) {
	mockBeads := &mockBeadCreator{}
	router := &Router{Beads: mockBeads}

	actx := ActionContext{AgentID: "agent-1", ProjectID: "project-1"}

	tests := []struct {
		name    string
		action  Action
		wantErr string
	}{
		{
			name: "missing delegate_to_role",
			action: Action{
				Type:      ActionDelegateTask,
				TaskTitle: "Task",
			},
			wantErr: "delegate_to_role is required",
		},
		{
			name: "missing task_title",
			action: Action{
				Type:           ActionDelegateTask,
				DelegateToRole: "qa-engineer",
			},
			wantErr: "task_title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.handleDelegateTask(context.Background(), tt.action, actx)

			assert.Equal(t, "error", result.Status)
			assert.Contains(t, result.Message, tt.wantErr)
		})
	}
}

func TestHandleDelegateTask_BeadCreatorNotConfigured(t *testing.T) {
	router := &Router{Beads: nil}

	action := Action{
		Type:           ActionDelegateTask,
		DelegateToRole: "qa-engineer",
		TaskTitle:      "Test Task",
	}

	actx := ActionContext{AgentID: "agent-1", ProjectID: "project-1"}

	result := router.handleDelegateTask(context.Background(), action, actx)

	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "bead creator not configured")
}

func TestHandleDelegateTask_CreateBeadError(t *testing.T) {
	mockBeads := &mockBeadCreator{
		createError: assert.AnError,
	}
	router := &Router{Beads: mockBeads}

	action := Action{
		Type:           ActionDelegateTask,
		DelegateToRole: "qa-engineer",
		TaskTitle:      "Test Task",
	}

	actx := ActionContext{AgentID: "agent-1", ProjectID: "project-1"}

	result := router.handleDelegateTask(context.Background(), action, actx)

	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "failed to create child bead")
}

func TestHandleDelegateTask_AllPriorities(t *testing.T) {
	mockBeads := &mockBeadCreator{}
	router := &Router{Beads: mockBeads}

	priorities := []int{0, 1, 2, 3, 4} // P0-P4

	for _, priority := range priorities {
		action := Action{
			Type:           ActionDelegateTask,
			DelegateToRole: "qa-engineer",
			TaskTitle:      "Test Task",
			TaskPriority:   priority,
		}

		actx := ActionContext{
			AgentID:   "agent-1",
			ProjectID: "project-1",
		}

		result := router.handleDelegateTask(context.Background(), action, actx)

		assert.Equal(t, "executed", result.Status)
		assert.Equal(t, priority, result.Metadata["task_priority"])
	}

	// Verify all beads created with correct priorities
	assert.Len(t, mockBeads.createdBeads, len(priorities))
	for i, priority := range priorities {
		assert.Equal(t, models.BeadPriority(priority), mockBeads.createdBeads[i].Priority)
	}
}
