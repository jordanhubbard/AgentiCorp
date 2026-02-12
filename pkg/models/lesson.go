package models

import "time"

// Lesson represents a learned insight from agent execution.
// Lessons are per-project and injected into future prompts to avoid repeating mistakes.
type Lesson struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	Category       string    `json:"category"` // compiler_error, test_failure, edit_failure, loop_pattern, conversation_insight
	Title          string    `json:"title"`
	Detail         string    `json:"detail"`
	SourceBeadID   string    `json:"source_bead_id,omitempty"`
	SourceAgentID  string    `json:"source_agent_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	RelevanceScore float64   `json:"relevance_score"` // Decays over time
	Embedding      []float32 `json:"-"`               // Vector embedding for semantic search (not serialized)
}
