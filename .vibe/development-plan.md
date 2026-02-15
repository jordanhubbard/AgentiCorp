# Development Plan: Workflow System Implementation

*Generated on 2026-02-15 by Vibe Feature MCP*
*Workflow: [waterfall](https://mrsimpson.github.io/responsible-vibe-mcp/workflows/waterfall)*

## Goal
Implement a comprehensive workflow system to enable multi-step agent workflows, role-based task assignment, and autonomous self-healing. This will unblock Loom's ability to fully fix its own bugs by defining who does what in multi-step processes (investigation → approval → fix → commit → verify).

## Requirements
<!-- beads-phase-id: loom-vhi7k -->

### Objectives
Define comprehensive requirements for the workflow system based on findings from SELF_HEALING_TEST_RESULTS.md.

### Tasks
*Tasks managed via `bd` CLI*

### Key Requirements to Define
- Workflow DAG structure (nodes, edges, transitions)
- Multi-dispatch support for long-running agent investigations
- Role-based assignment (Engineering Manager for commits, QA for verification)
- Retry and escalation logic
- CEO override capabilities
- Workflow state persistence
- Concurrent workflow execution

---

## Design
<!-- beads-phase-id: TBD -->

### Phase Entrance Criteria:
- [ ] Requirements for workflow DAG structure are clearly defined
- [ ] Multi-dispatch mechanism requirements are documented
- [ ] Role-based assignment requirements are specified
- [ ] Retry/escalation logic requirements are defined
- [ ] Database schema requirements are outlined
- [ ] Integration points with dispatcher are identified
- [ ] Success criteria and metrics are defined

### Objectives
Design the architecture and implementation approach for the workflow system.

### Tasks
*Tasks managed via `bd` CLI*

### Key Design Areas
- Workflow package structure and API
- Database schema for workflows and state
- Workflow engine state machine
- Dispatcher integration
- Role assignment logic
- Bead context extensions

---

## Implementation
<!-- beads-phase-id: TBD -->

### Phase Entrance Criteria:
- [ ] Architecture design is complete and documented in $ARCHITECTURE_DOC
- [ ] Database schema is designed
- [ ] API interfaces are defined
- [ ] Integration approach with existing dispatcher is clear
- [ ] Data structures for workflow DAGs are specified
- [ ] State machine transitions are documented
- [ ] Error handling strategy is defined

### Objectives
Implement the workflow system components.

### Tasks
*Tasks managed via `bd` CLI*

### Implementation Components
1. Workflow package (`internal/workflow`)
   - DAG structures and validation
   - Workflow engine
   - State management
2. Database layer
   - Schema migrations
   - Workflow CRUD operations
3. Dispatcher integration
   - Multi-dispatch support
   - Role-based assignment
4. Retry and escalation logic
5. CEO permission checks

---

## Unit Testing
<!-- beads-phase-id: TBD -->

### Phase Entrance Criteria:
- [ ] All core workflow components are implemented
- [ ] Workflow DAG creation and validation work
- [ ] State machine transitions are functional
- [ ] Database operations are working
- [ ] Integration with dispatcher is functional
- [ ] Code compiles without errors

### Objectives
Write comprehensive unit tests for all workflow components.

### Tasks
*Tasks managed via `bd` CLI*

### Testing Areas
- Workflow DAG validation
- State machine transitions
- Role assignment logic
- Retry and escalation
- Multi-dispatch behavior
- Database operations
- Edge cases and error handling

---

## Integration Testing
<!-- beads-phase-id: TBD -->

### Phase Entrance Criteria:
- [ ] Unit tests are written and passing
- [ ] Test coverage is >75% for workflow package
- [ ] All critical paths have test coverage
- [ ] Edge cases are tested
- [ ] Database operations are tested

### Objectives
Test end-to-end workflow execution with real agents and beads.

### Tasks
*Tasks managed via `bd` CLI*

### Integration Test Scenarios
1. Simple linear workflow (Start → Task → End)
2. Multi-step investigation with auto-redispatch
3. Approval workflow (Agent → CEO → Apply)
4. Retry on failure
5. Escalation to CEO after max retries
6. Concurrent workflows without conflicts
7. Role-based assignment (Engineering Manager commits)

---

## System Testing
<!-- beads-phase-id: TBD -->

### Phase Entrance Criteria:
- [ ] Integration tests are passing
- [ ] End-to-end workflows execute successfully
- [ ] Multi-dispatch works correctly
- [ ] Role assignment works correctly
- [ ] Retry/escalation logic works
- [ ] No critical bugs found in integration testing

### Objectives
Test complete self-healing loop from bug detection through fix verification.

### Tasks
*Tasks managed via `bd` CLI*

### System Test Scenarios
From SELF_HEALING_TEST_RESULTS.md:
1. **Investigation Continuation**: Agent gets multiple dispatch cycles
2. **Full Self-Healing Loop**: Bug → Investigate → Approve → Fix/Commit → Verify → Close
3. **Failure Handling**: Agent attempts fix 3 times → Escalate to CEO
4. **Concurrent Bugs**: 5 bugs simultaneously without conflicts

---

## Deployment
<!-- beads-phase-id: TBD -->

### Phase Entrance Criteria:
- [ ] System tests pass
- [ ] Full self-healing loop works end-to-end
- [ ] Performance is acceptable
- [ ] No critical bugs found
- [ ] Documentation is complete
- [ ] Migration path is defined

### Objectives
Deploy workflow system to enable autonomous self-healing.

### Tasks
*Tasks managed via `bd` CLI*

### Deployment Steps
1. Run database migrations
2. Deploy updated loom binary
3. Configure default workflows (auto-bug workflow)
4. Enable auto-redispatch in dispatcher
5. Monitor initial workflow executions
6. Measure self-healing metrics

---

## Key Decisions
*Important decisions will be documented here as they are made*

### Requirements Phase (2026-02-15):
- **Workflow Type**: Directed Acyclic Graph (DAG) for deterministic execution
- **Database**: SQLite with 3 tables (workflows, workflow_executions, workflow_transitions)
- **Node Types**: task, approval, decision, merge
- **Roles**: engineering-manager, qa-engineer, web-designer, backend-engineer, project-manager, ceo
- **Multi-Dispatch**: `redispatch_requested` flag in bead context enables multi-turn work
- **Commit Safety**: Serialize commit operations with queue and lock mechanism
- **Max Attempts**: Default 3 retries per node before escalation
- **Escalation**: Auto-create CEO approval bead when max attempts exceeded

### From SELF_HEALING_TEST_RESULTS.md:
- **Beads to Implement**: ac-1450 through ac-1455
- **Priority**: P0 - Blocks all self-healing functionality
- **Critical Path**: Engineering Manager must handle commits/pushes
- **Safety**: Single commit node per workflow to prevent conflicts

---

## Notes
*Additional context and observations*

### Current State (from SELF_HEALING_TEST_RESULTS.md):
- Auto-filing: ✅ Working
- Auto-routing: ✅ Working
- Auto-dispatch: ✅ Working
- Agent investigation start: ✅ Working
- Multi-step investigation: ❌ Blocked (agents get 1 turn then stop)
- Fix application: ❌ Blocked (no defined commit/push role)
- Verification: ❌ Blocked (no workflow for testing)

### Architecture References:
- Workflow DAG structures needed
- State machine for transitions
- Database schema for persistence
- Integration with `internal/dispatch/dispatcher.go`
- Extension of bead context for workflow state

---

*This plan is maintained by the LLM and uses beads CLI for task management. Tool responses provide guidance on which bd commands to use for task management.*
