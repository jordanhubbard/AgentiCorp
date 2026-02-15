# Requirements: Workflow System for Autonomous Self-Healing

## Context

Loom's self-healing infrastructure is 95% complete but blocked by the absence of a workflow system. Currently:
- ✅ Bugs are auto-filed when API errors occur (status >= 500)
- ✅ Bugs are auto-routed to appropriate specialists based on error type
- ✅ Agents begin investigating bugs
- ❌ Agents execute ONE action then stop (no multi-turn investigation)
- ❌ No defined path for who commits/pushes code changes
- ❌ No verification or retry mechanisms
- ❌ No role-based assignment for specialized tasks

**Critical Blocker**: Without workflow DAGs defining task progression, autonomous self-healing cannot be achieved.

**Reference**: `docs/SELF_HEALING_TEST_RESULTS.md` (2026-01-27 test results)

**Stakeholders**:
- **CEO** (human user): Approves/rejects fixes, escalation point
- **Agents** (autonomous): Investigate bugs, propose fixes, apply fixes, verify fixes
- **Dispatcher**: Routes beads to agents based on workflow state
- **Loom System**: Maintains workflow state, enforces rules

---

## REQ-1: Workflow DAG Structure

**User Story:** As a system administrator, I want to define workflows as directed acyclic graphs so that multi-step agent processes can be orchestrated with clear task progression.

**Acceptance Criteria:**

- The system SHALL support workflow definitions with nodes and directed edges
- The system SHALL validate that workflow graphs are acyclic (no cycles)
- WHEN a workflow is created THEN the system SHALL assign it a unique identifier
- Each workflow node SHALL specify an agent role (e.g., "web-designer", "engineering-manager", "qa-engineer")
- Each workflow node SHALL support configuration for max attempts and timeout
- The system SHALL support multiple node types: task, approval, decision, merge
- WHEN validating workflow THEN the system SHALL ensure exactly one start node and at least one end node

---

## REQ-2: Multi-Dispatch Support

**User Story:** As an agent, I want to continue working on a bead across multiple dispatch cycles so that I can complete multi-step investigations.

**Acceptance Criteria:**

- While a bead is in a workflow investigation node, when the agent completes an action, the system SHALL mark the bead for redispatch
- WHEN an agent requests more turns THEN the system SHALL set `redispatch_requested: true` in bead context
- The dispatcher SHALL re-dispatch beads with `redispatch_requested: true` even if `last_run_at` is set
- WHEN max attempts are reached for a node THEN the system SHALL transition to the escalation path
- The system SHALL track the number of dispatch cycles per workflow node
- If max attempts exceeded THEN the system SHALL escalate to CEO approval node

---

## REQ-3: Role-Based Assignment

**User Story:** As a workflow designer, I want to assign specific agent roles to workflow nodes so that specialized tasks (like committing code) go to the right agents.

**Acceptance Criteria:**

- Each workflow node SHALL specify a required agent role or persona
- WHEN dispatcher processes a bead in a workflow THEN the system SHALL match the bead to agents with the required role
- The system SHALL support role assignment for: "engineering-manager", "qa-engineer", "web-designer", "backend-engineer", "project-manager", "ceo"
- WHEN a commit/push node is reached THEN the system SHALL ONLY assign to "engineering-manager" role
- WHEN a verification node is reached THEN the system SHALL ONLY assign to "qa-engineer" role
- If no agent matches required role THEN the system SHALL escalate to CEO

---

## REQ-4: Workflow State Persistence

**User Story:** As the Loom system, I want to persist workflow execution state so that workflows can resume after system restarts.

**Acceptance Criteria:**

- The system SHALL store workflow definitions in the database
- The system SHALL track current workflow node for each bead in workflow
- WHEN bead transitions to new node THEN the system SHALL persist the transition with timestamp
- The system SHALL store workflow execution history (all node transitions)
- The system SHALL persist retry count per node
- If system restarts THEN workflow SHALL resume from last persisted state

---

## REQ-5: State Machine Transitions

**User Story:** As a workflow engine, I want to transition beads through workflow nodes based on outcomes so that workflows progress correctly.

**Acceptance Criteria:**

- WHEN a bead completes a workflow node successfully THEN the system SHALL transition to the success edge's target node
- WHEN a bead fails at a workflow node THEN the system SHALL transition to the failure edge's target node if present
- If no failure edge exists and task fails THEN the system SHALL retry up to max attempts
- WHEN max retries exceeded with no failure edge THEN the system SHALL escalate to CEO approval node
- The system SHALL support conditional transitions based on bead context values
- WHEN bead reaches end node THEN the system SHALL mark workflow as complete and close bead

---

## REQ-6: Retry and Escalation Logic

**User Story:** As a system designer, I want automatic retry and escalation so that transient failures are retried and persistent failures are escalated to humans.

**Acceptance Criteria:**

- Each workflow node SHALL support configurable max_attempts (default: 3)
- WHEN task fails THEN the system SHALL retry up to max_attempts before escalating
- WHEN max attempts exceeded THEN the system SHALL create CEO approval bead with failure context
- The CEO approval bead SHALL include: original bead ID, failure reason, attempt history, proposed next steps
- WHEN CEO approves THEN the system SHALL transition to next node in workflow
- WHEN CEO rejects THEN the system SHALL close workflow and bead
- The system SHALL track retry count and timestamps for each attempt

---

## REQ-7: CEO Override Capabilities

**User Story:** As a CEO (human user), I want to override workflow decisions at any point so that I maintain control over autonomous operations.

**Acceptance Criteria:**

- The system SHALL allow CEO to approve/reject at any approval node
- The system SHALL allow CEO to manually transition bead to any workflow node
- The system SHALL allow CEO to close workflow and bead at any time
- WHEN CEO overrides THEN the system SHALL log the override action with reason
- The system SHALL allow CEO to modify workflow definitions
- The system SHALL allow CEO to pause/resume workflow execution

---

## REQ-8: Workflow Engine Core

**User Story:** As the Loom system, I want a workflow engine that executes workflows reliably so that beads progress through defined processes.

**Acceptance Criteria:**

- The workflow engine SHALL load workflow definitions from database
- WHEN bead is created with workflow_id THEN the system SHALL initialize workflow state at start node
- The workflow engine SHALL evaluate transition conditions and move beads between nodes
- The workflow engine SHALL enforce node constraints (role requirements, max attempts, timeouts)
- WHEN node timeout exceeded THEN the system SHALL escalate to CEO
- The workflow engine SHALL support concurrent execution of multiple workflows without conflicts
- The workflow engine SHALL be idempotent (safe to call multiple times on same bead)

---

## REQ-9: Dispatcher Integration

**User Story:** As the dispatcher, I want to use workflow state to route beads so that beads go to the right agents at the right workflow stage.

**Acceptance Criteria:**

- WHEN dispatcher processes bead with active workflow THEN dispatcher SHALL check current workflow node for role requirements
- The dispatcher SHALL match beads to agents based on workflow node's required role
- The dispatcher SHALL respect `redispatch_requested: true` flag for multi-turn investigations
- The dispatcher SHALL skip beads in approval nodes (awaiting CEO)
- WHEN bead reaches commit node THEN dispatcher SHALL ONLY assign to engineering-manager agents
- The dispatcher SHALL provide workflow context to agents (current node, attempt count, workflow progress)

---

## REQ-10: Database Schema

**User Story:** As a developer, I want a database schema that efficiently stores workflows and state so that workflow data is persistent and queryable.

**Acceptance Criteria:**

- The system SHALL have a `workflows` table with: id, name, definition (JSON), created_at, updated_at
- The system SHALL have a `workflow_executions` table with: id, bead_id, workflow_id, current_node_id, state (JSON), created_at, updated_at
- The system SHALL have a `workflow_transitions` table with: id, execution_id, from_node_id, to_node_id, outcome, timestamp
- Workflow definition JSON SHALL include: nodes array, edges array, metadata
- Node definition SHALL include: id, type, role, max_attempts, timeout, config
- Edge definition SHALL include: id, from_node_id, to_node_id, condition
- The system SHALL support querying active workflows by bead_id
- The system SHALL support querying workflow history by bead_id

---

## REQ-11: Default Auto-Bug Workflow

**User Story:** As a system administrator, I want a pre-configured workflow for auto-filed bugs so that self-healing works out of the box.

**Acceptance Criteria:**

- The system SHALL provide a default "auto-bug-workflow" definition
- The auto-bug workflow SHALL have nodes: Start → QA Triage → Specialist Investigation → CEO Approval → Engineering Manager Apply/Commit → QA Verification → End
- The Specialist Investigation node SHALL support multi-dispatch (max 5 attempts)
- The Engineering Manager node SHALL be restricted to engineering-manager role
- The QA Verification node SHALL be restricted to qa-engineer role
- WHEN auto-bug-workflow reaches CEO Approval THEN system SHALL create approval bead with fix proposal
- WHEN verification fails THEN workflow SHALL retry apply/commit node (max 3 times)
- If 3 retries exceeded THEN workflow SHALL escalate to CEO with failure details

---

## REQ-12: Concurrent Workflow Execution

**User Story:** As the Loom system, I want to execute multiple workflows concurrently without conflicts so that multiple bugs can be fixed simultaneously.

**Acceptance Criteria:**

- The system SHALL support multiple active workflow executions simultaneously
- WHEN multiple beads reach commit nodes THEN system SHALL serialize commit operations (one at a time)
- The system SHALL implement a commit queue to prevent git conflicts
- WHEN agent completes commit operation THEN system SHALL release commit lock
- The system SHALL timeout stale commit locks after 5 minutes
- The system SHALL prevent multiple agents from committing to same file simultaneously
- If commit conflict detected THEN system SHALL retry with fresh pull

---

## REQ-13: Workflow Validation

**User Story:** As a system administrator, I want workflow definitions to be validated so that invalid workflows are rejected before deployment.

**Acceptance Criteria:**

- WHEN workflow is created or updated THEN system SHALL validate the workflow structure
- The system SHALL reject workflows with cycles (must be DAG)
- The system SHALL reject workflows without a start node
- The system SHALL reject workflows where nodes are unreachable from start
- The system SHALL reject workflows with undefined roles
- The system SHALL reject workflows with invalid node types
- The system SHALL reject workflows with edges referencing non-existent nodes
- If validation fails THEN system SHALL return detailed error messages

---

## REQ-14: Workflow Metrics and Monitoring

**User Story:** As a system administrator, I want metrics on workflow execution so that I can monitor self-healing effectiveness.

**Acceptance Criteria:**

- The system SHALL track time to resolution per workflow (start to end)
- The system SHALL track success rate per workflow type
- The system SHALL track escalation rate (% requiring CEO intervention)
- The system SHALL track retry count distribution per node
- The system SHALL track average dispatch cycles per investigation node
- The system SHALL expose metrics via `/api/v1/metrics/workflows` endpoint
- The system SHALL log workflow state transitions for debugging

---

## Success Metrics

**From SELF_HEALING_TEST_RESULTS.md:**

1. **Time to Resolution**: Error detected → Fix applied → Verified
   - Target: < 5 minutes for simple bugs

2. **Investigation Success Rate**: % of bugs where agent identifies root cause
   - Target: > 80%

3. **Fix Success Rate**: % of proposed fixes that pass verification
   - Target: > 90%

4. **Escalation Rate**: % of bugs requiring CEO intervention
   - Target: < 10%

5. **Workflow Cycle Count**: Average cycles before completion
   - Target: < 1.5 (most bugs complete first try)

6. **Commit Safety**:
   - Zero simultaneous commits from multiple agents
   - All commits have proper authorship
   - All commits pass pre-commit hooks

---

## Constraints

### Technical Constraints
- Must integrate with existing `internal/dispatch/dispatcher.go` (minimal changes)
- Must use existing database (SQLite)
- Must maintain backward compatibility with existing beads
- Go 1.21+ required

### Time Constraints
- P0 priority - Blocks all self-healing functionality
- Target: 1-2 weeks for Phase 1 implementation

### Safety Constraints
- CEO must be able to override any autonomous action
- All code changes must be attributed to correct agent
- No workflow can bypass pre-commit hooks
- Workflow executions must be auditable (full history)

---

## Out of Scope (Future Enhancements)

- ❌ Visual workflow editor UI (Phase 3 - ac-1454)
- ❌ Workflow templates marketplace
- ❌ Machine learning for workflow optimization
- ❌ Distributed workflow execution across multiple Loom instances
- ❌ Workflow versioning and rollback
- ❌ Custom node types beyond built-in set

---

## Related Beads

- **ac-1450**: Workflow package with DAG structures (P0 - Week 1)
- **ac-1451**: Database schema for workflow configurations (P0 - Week 1)
- **ac-1452**: Workflow engine for state transitions (P0 - Week 1-2)
- **ac-1453**: Retry and escalation logic (P0 - Week 1-2)
- **ac-1455**: CEO permission checks (P0 - Week 1-2)
- **ac-1454**: Graph visualization UI (P1 - Week 2-3)
