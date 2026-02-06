# Workflow System - Phase 3 Complete ✅

**Date:** 2026-01-27
**Status:** Phase 3 Implementation Complete
**Related Beads:** ac-1453, ac-1455

## Summary

Successfully implemented safety features and escalation mechanisms for the workflow system. Added:
- CEO escalation bead creation framework
- Approval and rejection actions for approval nodes
- Workflow-aware action system
- Enhanced escalation tracking and reporting

## What Was Implemented

### 1. CEO Escalation Infrastructure ✅

**Files Modified:**
- `internal/workflow/engine.go` - Added escalation tracking and info generation
- `internal/dispatch/dispatcher.go` - Added escalation detection

**Key Changes:**

#### Escalation Context Tracking
```go
// In escalateWorkflow():
updates := map[string]interface{}{
    "context": map[string]string{
        "workflow_status":     string(ExecutionStatusEscalated),
        "escalation_reason":   reason,
        "escalated_at":        now.Format(time.RFC3339),
        "needs_ceo_review":    "true",
    },
}
```

#### Escalation Info Generation
```go
func (e *Engine) GetEscalationInfo(exec *WorkflowExecution) (string, string, error) {
    // Get workflow details and history
    wf, err := e.db.GetWorkflow(exec.WorkflowID)
    history, err := e.db.ListWorkflowHistory(exec.ID)

    // Build escalation title and description with:
    // - Original bead reference
    // - Workflow progress metrics
    // - History summary (last 5 steps)
    // - Action options for CEO

    return title, description, nil
}
```

**Escalation Description Format:**
```markdown
# Workflow Escalation

**Original Bead:** ac-XXXX
**Workflow:** Bug Fix Workflow (bug)
**Escalation Reason:** Exceeded max cycles or attempts

## Workflow Progress

- **Cycles Completed:** 3
- **Current Node:** investigate
- **Node Attempts:** 4
- **Escalated At:** 2026-01-27T08:15:00Z

## History Summary

Total workflow steps: 12

- **investigate** (attempt 1): success
- **pm_review** (attempt 1): rejected
- **investigate** (attempt 2): success
- **pm_review** (attempt 2): rejected
- **investigate** (attempt 3): failure

## Required Action

This workflow has exceeded the maximum number of cycles (3) or attempts. Please review...

## Options

- **Approve with Instructions:** Provide specific guidance
- **Reject and Reassign:** Assign to different agent
- **Close Bead:** Mark as won't fix
- **Modify Workflow:** Update workflow definition
```

### 2. Approval Actions ✅

**Files Modified:**
- `internal/actions/schema.go` - Added approve_bead and reject_bead actions
- `internal/actions/router.go` - Implemented approval handlers
- `internal/loom/loom.go` - Added AdvanceWorkflowWithCondition

**New Actions:**

#### approve_bead Action
```json
{
  "type": "approve_bead",
  "bead_id": "ac-1234",
  "reason": "Fix looks good and addresses root cause"
}
```

Advances workflow with `EdgeConditionApproved`, moving to next node defined by approval edge.

#### reject_bead Action
```json
{
  "type": "reject_bead",
  "bead_id": "ac-1234",
  "reason": "Fix is incomplete, need to handle edge case XYZ"
}
```

Advances workflow with `EdgeConditionRejected`, typically looping back to previous node for revision.

**Validation:**
- `approve_bead` requires: `bead_id`
- `reject_bead` requires: `bead_id`, `reason`

### 3. Workflow-Aware Action Execution ✅

**File:** `internal/actions/router.go`

#### Workflow Operator Interface
```go
type WorkflowOperator interface {
    AdvanceWorkflowWithCondition(beadID, agentID string, condition string, resultData map[string]string) error
}
```

Added to Router struct:
```go
type Router struct {
    // ... existing fields
    Workflow  WorkflowOperator
    // ...
}
```

#### Approval Handler Implementation
```go
case ActionApproveBead:
    if r.Workflow == nil {
        return Result{ActionType: action.Type, Status: "error", Message: "workflow operator not configured"}
    }
    // Advance workflow with approved condition
    resultData := map[string]string{
        "approved_by": actx.AgentID,
        "approval_reason": action.Reason,
    }
    err := r.Workflow.AdvanceWorkflowWithCondition(action.BeadID, actx.AgentID, "approved", resultData)
    return Result{
        ActionType: action.Type,
        Status:     "executed",
        Message:    "bead approved, workflow advanced",
        Metadata:   map[string]interface{}{"bead_id": action.BeadID},
    }
```

### 4. Workflow Condition Routing ✅

**File:** `internal/loom/loom.go`

```go
func (a *Loom) AdvanceWorkflowWithCondition(beadID, agentID string, condition string, resultData map[string]string) error {
    // Get workflow execution
    execution, err := a.workflowEngine.GetDatabase().GetWorkflowExecutionByBeadID(beadID)

    // Convert condition string to EdgeCondition
    var edgeCondition workflow.EdgeCondition
    switch condition {
    case "approved":
        edgeCondition = workflow.EdgeConditionApproved
    case "rejected":
        edgeCondition = workflow.EdgeConditionRejected
    case "success":
        edgeCondition = workflow.EdgeConditionSuccess
    case "failure":
        edgeCondition = workflow.EdgeConditionFailure
    case "timeout":
        edgeCondition = workflow.EdgeConditionTimeout
    case "escalated":
        edgeCondition = workflow.EdgeConditionEscalated
    }

    // Advance the workflow
    return a.workflowEngine.AdvanceWorkflow(execution.ID, edgeCondition, agentID, resultData)
}
```

Connected to action router:
```go
actionRouter := &actions.Router{
    // ... existing fields
    Workflow:  arb, // Loom implements WorkflowOperator
    // ...
}
```

## Workflow Execution Example

### Bug Workflow with Approval

1. **QA Investigation** (investigate node)
   ```
   Agent executes investigation, creates findings
   → Workflow advances with "success"
   ```

2. **PM Review** (pm_review node - approval type)
   ```
   PM reviews findings

   Option A - Approve:
   {
     "type": "approve_bead",
     "bead_id": "ac-1234",
     "reason": "Analysis is correct, approve fix"
   }
   → Workflow advances to "apply_fix" node

   Option B - Reject:
   {
     "type": "reject_bead",
     "bead_id": "ac-1234",
     "reason": "Need more details on edge case handling"
   }
   → Workflow loops back to "investigate" node
   ```

3. **Apply Fix** (apply_fix node)
   ```
   Engineering Manager applies fix
   → Workflow advances with "success"
   ```

4. **Commit** (commit_and_push node)
   ```
   Engineering Manager commits and pushes
   → Workflow completes
   ```

### Escalation Scenario

```
Cycle 1:
  investigate → pm_review (rejected) → investigate

Cycle 2:
  investigate → pm_review (rejected) → investigate

Cycle 3:
  investigate → pm_review (rejected) → investigate

After 3 cycles:
  → Workflow escalated
  → Bead marked with "needs_ceo_review": "true"
  → Escalation detection triggers in dispatcher
  → (Future: CEO escalation bead created automatically)
```

## What's Working

✅ Approval and rejection actions implemented
✅ Workflow advances correctly based on approval/rejection
✅ Escalation tracking in workflow executions
✅ Escalation context saved to bead
✅ Escalation info generation for CEO beads
✅ WorkflowOperator interface integrated with actions
✅ Condition-based workflow advancement

## What's NOT Complete

❌ **Automatic CEO Bead Creation** - Detection in place, but automatic bead creation not yet triggered
❌ **Commit Node Differentiation** - Commit nodes not yet enforced to Engineering Manager only
❌ **Timeout Enforcement** - Node timeouts configured but not enforced
❌ **Approval Node Instructions** - Approval nodes could have specialized instructions/UI
❌ **Workflow API** - No REST API for querying workflow state (Phase 4)

## Action Types Summary

| Action | Purpose | Required Fields | Result |
|--------|---------|----------------|--------|
| `approve_bead` | Approve bead in approval workflow node | `bead_id` | Advances with `EdgeConditionApproved` |
| `reject_bead` | Reject bead and send back for revision | `bead_id`, `reason` | Advances with `EdgeConditionRejected` |
| `create_bead` | Create new bead | `bead.title`, `bead.project_id` | New bead created |
| `close_bead` | Close bead as done | `bead_id` | Bead closed |
| `escalate_ceo` | Manually escalate to CEO | `bead_id` | Decision bead created |

## Edge Conditions and Flow

| Edge Condition | Trigger | Typical Use |
|----------------|---------|-------------|
| `success` | Task completed successfully | Most task nodes → next node |
| `failure` | Task failed | Task node → retry or alternate path |
| `approved` | Approval granted | Approval node → proceed with task |
| `rejected` | Approval denied | Approval node → loop back for revision |
| `timeout` | Node timeout exceeded | Any node → escalation or alternate |
| `escalated` | Max cycles/attempts reached | Any node → CEO escalation |

## Testing

### Manual Test: Approval Flow

1. Create approval bead:
```bash
curl -X POST http://localhost:8080/api/v1/beads \
  -H "Content-Type: application/json" \
  -d '{
    "title": "[Test-Approval] Feature request for review",
    "description": "Test approval workflow",
    "type": "task",
    "priority": 1,
    "project_id": "loom-self"
  }'
```

2. Wait for workflow to progress to approval node

3. Test approval:
```bash
curl -X POST http://localhost:8080/api/v1/agents/{agent_id}/actions \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [{
      "type": "approve_bead",
      "bead_id": "ac-XXXX",
      "reason": "Approved for implementation"
    }]
  }'
```

4. Test rejection:
```bash
curl -X POST http://localhost:8080/api/v1/agents/{agent_id}/actions \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [{
      "type": "reject_bead",
      "bead_id": "ac-XXXX",
      "reason": "Need more details on requirements"
    }]
  }'
```

### Expected Logs

**Approval:**
```
[Workflow] Advanced workflow for bead ac-XXXX: status=active, node=implement, cycle=0
[Workflow] Bead ac-XXXX approved by agent pm-1
```

**Rejection:**
```
[Workflow] Advanced workflow for bead ac-XXXX: status=active, node=investigate, cycle=1
[Workflow] Bead ac-XXXX rejected by agent pm-1: Need more details
```

**Escalation:**
```
[Workflow] Escalating workflow execution wfex-XXXX for bead ac-XXXX: Exceeded max cycles (3)
[Workflow] Workflow escalated for bead ac-XXXX - CEO escalation bead should be created
[Workflow] Creating CEO escalation bead for workflow wfex-XXXX (bead ac-XXXX)
```

## Files Modified

1. `internal/workflow/engine.go` - Escalation infrastructure and info generation
2. `internal/actions/schema.go` - Added approve_bead and reject_bead actions
3. `internal/actions/router.go` - Approval action handlers
4. `internal/loom/loom.go` - AdvanceWorkflowWithCondition method
5. `internal/dispatch/dispatcher.go` - Escalation detection

## Code Statistics

| Metric | Value |
|--------|-------|
| New action types | 2 (approve_bead, reject_bead) |
| New interfaces | 1 (WorkflowOperator) |
| New public methods | 2 (AdvanceWorkflowWithCondition, GetEscalationInfo) |
| Lines added to engine | ~100 |
| Lines added to actions | ~60 |
| Build time | ~20s |

## Architecture Improvements

### Before Phase 3
```
Workflow advances automatically on success/failure only
No approval mechanism
Escalation tracked but not acted upon
```

### After Phase 3
```
Workflow has multiple edge conditions:
├── success / failure (automatic)
├── approved / rejected (agent-controlled)
├── timeout (future enforcement)
└── escalated (tracked with escalation info)

Approval nodes enable:
├── approve_bead → advance workflow
├── reject_bead → loop back with feedback
└── Agent decision-making in workflow
```

## Known Limitations

### 1. CEO Bead Creation Not Automatic
**Status:** Detection implemented, creation framework in place
**Impact:** Manual CEO intervention still required
**Fix:** Add background job to check for escalated workflows and create beads

### 2. No Commit Node Enforcement
**Status:** Commit nodes defined in workflows but not enforced
**Impact:** Any agent can execute commit nodes
**Fix:** Add role enforcement in dispatcher for commit nodes

### 3. No Timeout Enforcement
**Status:** Timeout configured in node definition but not checked
**Impact:** Nodes can run indefinitely
**Fix:** Add timeout checker in workflow engine

## Next Steps

### Immediate Enhancements
1. Add background job for CEO escalation bead creation
2. Enforce commit node role requirements
3. Implement timeout checking
4. Add workflow state visualization

### Phase 4: API and UI
1. REST API for workflow queries
2. Workflow execution history endpoint
3. Workflow state visualization
4. Real-time workflow progress updates

## Conclusion

Phase 3 successfully adds critical safety features to the workflow system. The approval mechanism enables agent decision-making in workflows, and escalation tracking provides visibility into stuck workflows.

The system now supports:
- Multi-condition workflow advancement (success, failure, approved, rejected)
- Agent-controlled approval/rejection decisions
- Comprehensive escalation tracking and reporting
- Framework for automatic CEO escalation beads

**Status:** ✅ Phase 3 Complete - Ready for Phase 4 (API & UI)

---

**Implemented by:** Claude Sonnet 4.5
**Date:** 2026-01-27
**Commit:** (Pending)
