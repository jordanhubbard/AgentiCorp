# Workflow Actions Reference

Workflow actions integrate the responsible-vibe-mcp development workflow system into Loom, enabling structured development processes (EPCC, TDD, waterfall, etc.) that guide agents through exploration, planning, coding, and verification phases.

## Overview

The workflow system provides:
- **Structured Development**: Phase-based workflows (Explore → Plan → Code → Commit)
- **Phase Guidance**: Context-aware instructions for each development phase
- **Review Gates**: Optional reviews before phase transitions
- **Workflow Resumption**: Continue work after conversation compaction/breaks

## Workflow Actions

### start_development

Initiates a new development workflow for a project.

**JSON Format:**
```json
{
  "type": "start_development",
  "workflow": "epcc",
  "require_reviews": false,
  "project_path": "/path/to/project"
}
```

**Fields:**
- `workflow` (required): Workflow type - one of:
  - `epcc` - Explore, Plan, Code, Commit (recommended for features)
  - `tdd` - Test-Driven Development (Red → Green → Refactor)
  - `waterfall` - V-Model for complex, design-heavy projects
  - `bugfix` - Focused workflow for bug fixes
  - `minor` - Streamlined for small changes
  - `greenfield` - Comprehensive for new projects
  - `custom` - Use custom workflow from .vibe/workflows
- `require_reviews` (optional): Require reviews before phase transitions (default: false)
- `project_path` (optional): Project directory path (auto-detected if not provided)

**Returns:**
```json
{
  "action_type": "start_development",
  "status": "mcp_required",
  "message": "start_development requires MCP tool call: mcp__responsible-vibe-mcp__start_development",
  "metadata": {
    "workflow": "epcc",
    "require_reviews": false,
    "mcp_tool": "mcp__responsible-vibe-mcp__start_development"
  }
}
```

**Example:**
```json
{
  "actions": [{
    "type": "start_development",
    "workflow": "epcc",
    "require_reviews": false
  }],
  "notes": "Starting EPCC workflow for feature development"
}
```

### whats_next

Retrieves phase-specific guidance and instructions for the current development phase.

**JSON Format:**
```json
{
  "type": "whats_next",
  "user_input": "user's latest message",
  "context": "current work context",
  "conversation_summary": "summary of progress",
  "recent_messages": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ]
}
```

**Fields:**
- All fields are optional
- `user_input` (optional): User's most recent message
- `context` (optional): Brief description of current work
- `conversation_summary` (optional): Summary of development progress
- `recent_messages` (optional): Array of recent conversation messages

**Returns:**
```json
{
  "action_type": "whats_next",
  "status": "mcp_required",
  "message": "whats_next requires MCP tool call: mcp__responsible-vibe-mcp__whats_next",
  "metadata": {
    "mcp_tool": "mcp__responsible-vibe-mcp__whats_next"
  }
}
```

**MCP Tool Response:**
The MCP tool returns phase-specific instructions:
```json
{
  "instructions": "You are in the Explore phase. Focus on understanding the codebase...",
  "current_phase": "explore",
  "next_phase": "plan",
  "entrance_criteria": ["Codebase understanding complete", "Requirements clarified"]
}
```

**Example:**
```json
{
  "actions": [{
    "type": "whats_next",
    "user_input": "I've explored the authentication code",
    "context": "Working on user authentication feature"
  }],
  "notes": "Checking what to do next after exploration"
}
```

### proceed_to_phase

Transitions to a specific development phase when current phase is complete.

**JSON Format:**
```json
{
  "type": "proceed_to_phase",
  "target_phase": "implementation",
  "review_state": "performed",
  "reason": "Planning complete, ready to implement"
}
```

**Fields:**
- `target_phase` (required): Phase to transition to (e.g., "explore", "plan", "code", "commit")
- `review_state` (required): Review completion status:
  - `"not-required"` - Reviews are disabled
  - `"pending"` - Review needed but not done yet
  - `"performed"` - Review complete
- `reason` (optional): Explanation for phase transition

**Returns:**
```json
{
  "action_type": "proceed_to_phase",
  "status": "mcp_required",
  "message": "proceed_to_phase requires MCP tool call: mcp__responsible-vibe-mcp__proceed_to_phase",
  "metadata": {
    "target_phase": "implementation",
    "review_state": "performed",
    "reason": "Planning complete, ready to implement",
    "mcp_tool": "mcp__responsible-vibe-mcp__proceed_to_phase"
  }
}
```

**Example:**
```json
{
  "actions": [{
    "type": "proceed_to_phase",
    "target_phase": "code",
    "review_state": "not-required",
    "reason": "Plan approved, moving to implementation"
  }],
  "notes": "Transitioning from planning to coding phase"
}
```

### conduct_review

Performs a review of the current phase before proceeding to the next phase.

**JSON Format:**
```json
{
  "type": "conduct_review",
  "target_phase": "code"
}
```

**Fields:**
- `target_phase` (required): Phase to transition to after review completes

**Returns:**
```json
{
  "action_type": "conduct_review",
  "status": "mcp_required",
  "message": "conduct_review requires MCP tool call: mcp__responsible-vibe-mcp__conduct_review",
  "metadata": {
    "target_phase": "code",
    "mcp_tool": "mcp__responsible-vibe-mcp__conduct_review"
  }
}
```

**MCP Tool Response:**
```json
{
  "review_result": "approved",
  "issues_found": [],
  "recommendations": ["Consider edge case handling"]
}
```

**Example:**
```json
{
  "actions": [{
    "type": "conduct_review",
    "target_phase": "code"
  }],
  "notes": "Reviewing plan before implementation"
}
```

### resume_workflow

Continues development after a break or conversation restart, providing full context.

**JSON Format:**
```json
{
  "type": "resume_workflow"
}
```

**Fields:**
- All fields optional
- No required parameters

**Returns:**
```json
{
  "action_type": "resume_workflow",
  "status": "mcp_required",
  "message": "resume_workflow requires MCP tool call: mcp__responsible-vibe-mcp__resume_workflow",
  "metadata": {
    "mcp_tool": "mcp__responsible-vibe-mcp__resume_workflow"
  }
}
```

**MCP Tool Response:**
Provides complete project context and next steps:
```json
{
  "current_phase": "code",
  "workflow": "epcc",
  "progress_summary": "Implemented user authentication, tests passing",
  "next_steps": ["Add API documentation", "Create PR"],
  "plan_excerpt": "..."
}
```

**Example:**
```json
{
  "actions": [{
    "type": "resume_workflow"
  }],
  "notes": "Resuming after conversation compaction"
}
```

## Status: mcp_required

All workflow actions return status `"mcp_required"` because they depend on the responsible-vibe-mcp MCP tools. The action system records the workflow operation, and the agent orchestration layer makes the actual MCP tool call.

**Workflow:**
1. Agent emits workflow action in action envelope
2. Router validates and logs the action
3. Router returns `mcp_required` status with MCP tool name
4. Agent orchestration layer detects `mcp_required` status
5. Orchestration layer calls the specified MCP tool
6. MCP tool result is provided back to the agent

## Workflow Types

### EPCC (Explore, Plan, Code, Commit)
Recommended for feature development:
1. **Explore**: Understand codebase and requirements
2. **Plan**: Design implementation approach
3. **Code**: Implement and test
4. **Commit**: Commit changes and create PR

### TDD (Test-Driven Development)
For quality-focused development:
1. **Explore**: Understand requirements
2. **Red**: Write failing test
3. **Green**: Make test pass
4. **Refactor**: Improve code quality

### Waterfall (V-Model)
For complex, design-heavy projects:
1. **Specification**: Requirements gathering
2. **Architecture**: System design
3. **Implementation**: Coding
4. **Testing**: Verification
5. **Deployment**: Release

### Bugfix
Focused workflow for bug fixes:
1. **Reproduce**: Confirm the bug
2. **Analyze**: Understand root cause
3. **Fix**: Implement solution
4. **Verify**: Test the fix

### Minor
Streamlined for small changes:
1. **Explore**: Quick analysis
2. **Implement**: Code + test + commit

### Greenfield
Comprehensive for new projects:
1. **Ideation**: Project vision
2. **Architecture**: System design
3. **Plan**: Implementation plan
4. **Code**: Development
5. **Document**: Documentation

## Integration with Beads

Workflow actions complement the beads issue tracking system:

- **Create Bead First**: Use `create_bead` before starting workflow
- **Track Progress**: Bead status reflects workflow phase
- **Close on Complete**: Close bead when workflow reaches final phase
- **Multiple Beads**: Each bead can have its own workflow instance

**Example Workflow + Beads:**
```json
{
  "actions": [
    {
      "type": "create_bead",
      "bead": {
        "title": "Add user profile feature",
        "type": "feature",
        "project_id": "proj-123"
      }
    },
    {
      "type": "start_development",
      "workflow": "epcc"
    }
  ]
}
```

## Best Practices

1. **Call whats_next() Regularly**: After each user message to get phase guidance
2. **Follow Phase Order**: Don't skip phases unless using minor workflow
3. **Document in Plan**: Use the plan file retrieved via whats_next to record decisions
4. **Review Before Transition**: Use conduct_review when require_reviews is true
5. **Resume After Breaks**: Call resume_workflow after conversation compaction

## Error Handling

**Validation Errors:**
```json
{
  "action_type": "start_development",
  "status": "error",
  "message": "start_development requires workflow"
}
```

**MCP Tool Errors:**
MCP tool errors are handled by the agent orchestration layer and returned to the agent with details.

## See Also

- [Beads Workflow](../internal/beads/README.md) - Issue tracking integration
- [Git Workflow](GIT_SECURITY_MODEL.md) - Git operations and security
- [Feedback Loops](FEEDBACK_LOOPS.md) - Build, lint, test orchestration
- [Action Router](../internal/actions/router.go) - Action execution system
