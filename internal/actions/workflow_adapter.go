package actions

import (
	"context"
	"fmt"
)

// MCPToolCaller defines the interface for calling MCP tools
// This allows the adapter to invoke MCP tools while remaining testable
type MCPToolCaller interface {
	CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)
}

// WorkflowMCPAdapter adapts MCP responsible-vibe tools to WorkflowOperator interface
type WorkflowMCPAdapter struct {
	caller MCPToolCaller
}

// NewWorkflowMCPAdapter creates a new workflow MCP adapter
func NewWorkflowMCPAdapter(caller MCPToolCaller) *WorkflowMCPAdapter {
	return &WorkflowMCPAdapter{
		caller: caller,
	}
}

// AdvanceWorkflowWithCondition advances the workflow based on a condition (legacy method)
func (a *WorkflowMCPAdapter) AdvanceWorkflowWithCondition(beadID, agentID string, condition string, resultData map[string]string) error {
	// This is a legacy method - for now, just return nil
	// In the future, this might map to a specific workflow advancement
	return nil
}

// StartDevelopment initiates a new development workflow
func (a *WorkflowMCPAdapter) StartDevelopment(ctx context.Context, workflow string, requireReviews bool, projectPath string) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"workflow":        workflow,
		"require_reviews": requireReviews,
	}

	if projectPath != "" {
		params["project_path"] = projectPath
	}

	result, err := a.caller.CallTool(ctx, "mcp__responsible-vibe-mcp__start_development", params)
	if err != nil {
		return nil, fmt.Errorf("start_development failed: %w", err)
	}

	return result, nil
}

// WhatsNext retrieves guidance for the current development phase
func (a *WorkflowMCPAdapter) WhatsNext(ctx context.Context, userInput, contextStr, conversationSummary string, recentMessages []map[string]string) (map[string]interface{}, error) {
	params := map[string]interface{}{}

	if userInput != "" {
		params["user_input"] = userInput
	}
	if contextStr != "" {
		params["context"] = contextStr
	}
	if conversationSummary != "" {
		params["conversation_summary"] = conversationSummary
	}
	if len(recentMessages) > 0 {
		params["recent_messages"] = recentMessages
	}

	result, err := a.caller.CallTool(ctx, "mcp__responsible-vibe-mcp__whats_next", params)
	if err != nil {
		return nil, fmt.Errorf("whats_next failed: %w", err)
	}

	return result, nil
}

// ProceedToPhase transitions to a specific development phase
func (a *WorkflowMCPAdapter) ProceedToPhase(ctx context.Context, targetPhase, reviewState, reason string) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"target_phase": targetPhase,
		"review_state": reviewState,
	}

	if reason != "" {
		params["reason"] = reason
	}

	result, err := a.caller.CallTool(ctx, "mcp__responsible-vibe-mcp__proceed_to_phase", params)
	if err != nil {
		return nil, fmt.Errorf("proceed_to_phase failed: %w", err)
	}

	return result, nil
}

// ConductReview performs a review before phase transition
func (a *WorkflowMCPAdapter) ConductReview(ctx context.Context, targetPhase string) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"target_phase": targetPhase,
	}

	result, err := a.caller.CallTool(ctx, "mcp__responsible-vibe-mcp__conduct_review", params)
	if err != nil {
		return nil, fmt.Errorf("conduct_review failed: %w", err)
	}

	return result, nil
}

// ResumeWorkflow continues development after a break
func (a *WorkflowMCPAdapter) ResumeWorkflow(ctx context.Context, includeSystemPrompt bool) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"include_system_prompt": includeSystemPrompt,
	}

	result, err := a.caller.CallTool(ctx, "mcp__responsible-vibe-mcp__resume_workflow", params)
	if err != nil {
		return nil, fmt.Errorf("resume_workflow failed: %w", err)
	}

	return result, nil
}
