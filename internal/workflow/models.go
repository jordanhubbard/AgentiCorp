package workflow

import (
	"time"
)

// NodeType represents the type of workflow node
type NodeType string

const (
	NodeTypeTask     NodeType = "task"     // General task node
	NodeTypeApproval NodeType = "approval" // Requires approval to proceed
	NodeTypeCommit   NodeType = "commit"   // Git commit/push operation
	NodeTypeVerify   NodeType = "verify"   // Verification/testing node
)

// EdgeCondition represents conditions for workflow transitions
type EdgeCondition string

const (
	EdgeConditionSuccess   EdgeCondition = "success"   // Task completed successfully
	EdgeConditionFailure   EdgeCondition = "failure"   // Task failed
	EdgeConditionApproved  EdgeCondition = "approved"  // Approval granted
	EdgeConditionRejected  EdgeCondition = "rejected"  // Approval rejected
	EdgeConditionTimeout   EdgeCondition = "timeout"   // Node timed out
	EdgeConditionEscalated EdgeCondition = "escalated" // Escalated to higher authority
)

// ExecutionStatus represents the status of a workflow execution
type ExecutionStatus string

const (
	ExecutionStatusActive    ExecutionStatus = "active"    // Currently running
	ExecutionStatusBlocked   ExecutionStatus = "blocked"   // Waiting for dependencies
	ExecutionStatusCompleted ExecutionStatus = "completed" // Successfully finished
	ExecutionStatusFailed    ExecutionStatus = "failed"    // Failed permanently
	ExecutionStatusEscalated ExecutionStatus = "escalated" // Escalated to CEO
)

// Workflow represents a workflow definition
type Workflow struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	WorkflowType string         `json:"workflow_type"` // "bug", "feature", "ui", "custom"
	IsDefault    bool           `json:"is_default"`    // Is this a default workflow?
	ProjectID    string         `json:"project_id"`    // Empty for global defaults
	Nodes        []WorkflowNode `json:"nodes"`
	Edges        []WorkflowEdge `json:"edges"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// WorkflowNode represents a node in the workflow
type WorkflowNode struct {
	ID             string            `json:"id"`
	WorkflowID     string            `json:"workflow_id"`
	NodeKey        string            `json:"node_key"`        // Unique key within workflow (e.g., "investigate", "approve", "commit")
	NodeType       NodeType          `json:"node_type"`       // task, approval, commit, verify
	RoleRequired   string            `json:"role_required"`   // Agent role required (e.g., "Engineering Manager")
	PersonaHint    string            `json:"persona_hint"`    // Persona path hint for dispatcher
	MaxAttempts    int               `json:"max_attempts"`    // Max attempts before escalation (0 = unlimited)
	TimeoutMinutes int               `json:"timeout_minutes"` // Timeout in minutes (0 = no timeout)
	Instructions   string            `json:"instructions"`    // Instructions for the agent
	Metadata       map[string]string `json:"metadata"`        // Additional node-specific metadata
	CreatedAt      time.Time         `json:"created_at"`
}

// WorkflowEdge represents a transition between nodes
type WorkflowEdge struct {
	ID          string        `json:"id"`
	WorkflowID  string        `json:"workflow_id"`
	FromNodeKey string        `json:"from_node_key"` // Source node key (empty = workflow start)
	ToNodeKey   string        `json:"to_node_key"`   // Target node key (empty = workflow end)
	Condition   EdgeCondition `json:"condition"`     // Condition for transition
	Priority    int           `json:"priority"`      // Priority when multiple edges match (higher = first)
	CreatedAt   time.Time     `json:"created_at"`
}

// WorkflowExecution represents an active workflow execution for a bead
type WorkflowExecution struct {
	ID               string          `json:"id"`
	WorkflowID       string          `json:"workflow_id"`
	BeadID           string          `json:"bead_id"`
	ProjectID        string          `json:"project_id"`
	CurrentNodeKey   string          `json:"current_node_key"`   // Current node being executed (empty = workflow start)
	Status           ExecutionStatus `json:"status"`             // active, blocked, completed, failed, escalated
	CycleCount       int             `json:"cycle_count"`        // Number of times workflow has cycled
	NodeAttemptCount int             `json:"node_attempt_count"` // Attempts at current node
	StartedAt        time.Time       `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	EscalatedAt      *time.Time      `json:"escalated_at,omitempty"`
	LastNodeAt       time.Time       `json:"last_node_at"` // Last time node was updated
}

// WorkflowExecutionHistory represents an audit trail of workflow state changes
type WorkflowExecutionHistory struct {
	ID            string        `json:"id"`
	ExecutionID   string        `json:"execution_id"`
	NodeKey       string        `json:"node_key"`       // Node that was executed
	AgentID       string        `json:"agent_id"`       // Agent that executed the node
	Condition     EdgeCondition `json:"condition"`      // Condition that was satisfied
	ResultData    string        `json:"result_data"`    // JSON-encoded result data
	AttemptNumber int           `json:"attempt_number"` // Which attempt was this?
	CreatedAt     time.Time     `json:"created_at"`
}
