# Agent-to-Agent Communication Protocol

## Overview

Loom agents can communicate directly with each other to collaborate on complex tasks, share context, make consensus decisions, and delegate work. This document defines the protocol for inter-agent communication.

## Architecture

```
┌─────────────┐         ┌─────────────────┐         ┌─────────────┐
│  Agent A    │────────▶│  Message Bus    │◀────────│  Agent B    │
│             │         │  (Event Bus)    │         │             │
│ - Send msg  │         │                 │         │ - Receive   │
│ - Subscribe │         │ - Route msgs    │         │ - Process   │
│ - Respond   │         │ - Store history │         │ - Respond   │
└─────────────┘         └─────────────────┘         └─────────────┘
       │                         │                         │
       │                         │                         │
       ▼                         ▼                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Shared Context Store                          │
│  - Bead contexts    - Agent status    - Message history         │
└─────────────────────────────────────────────────────────────────┘
```

## Message Types

### 1. Direct Message (Agent-to-Agent)

Send a message directly to another agent.

```json
{
  "type": "agent_message",
  "from_agent_id": "agent-qa-engineer-1",
  "to_agent_id": "agent-code-reviewer-2",
  "message_id": "msg-abc-123",
  "subject": "Review request for auth module",
  "body": "Can you review the authentication changes in src/auth.go?",
  "priority": "normal",
  "requires_response": true,
  "context": {
    "bead_id": "bead-xyz-789",
    "files": ["src/auth.go", "src/auth_test.go"],
    "action": "code_review"
  },
  "timestamp": "2026-02-05T20:00:00Z"
}
```

**Fields:**
- `type`: "agent_message" (direct message)
- `from_agent_id`: Sending agent ID
- `to_agent_id`: Target agent ID
- `message_id`: Unique message identifier
- `subject`: Brief message subject
- `body`: Message content (plain text or markdown)
- `priority`: "low" | "normal" | "high" | "urgent"
- `requires_response`: Boolean - whether response is expected
- `context`: Optional context (bead, files, action type)
- `timestamp`: ISO 8601 timestamp

### 2. Broadcast Message

Send a message to all agents or filtered subset.

```json
{
  "type": "broadcast",
  "from_agent_id": "agent-project-manager-1",
  "message_id": "msg-def-456",
  "subject": "Sprint planning meeting",
  "body": "Planning session starts in 5 minutes. All agents please join.",
  "filter": {
    "persona_types": ["engineer", "qa", "code-reviewer"],
    "exclude_agents": ["agent-housekeeping-bot-1"]
  },
  "timestamp": "2026-02-05T20:00:00Z"
}
```

**Fields:**
- `type`: "broadcast"
- `filter`: Optional filters (persona types, exclude specific agents)

### 3. Request-Response

Request information or action from another agent.

**Request:**
```json
{
  "type": "request",
  "from_agent_id": "agent-engineer-1",
  "to_agent_id": "agent-qa-engineer-2",
  "message_id": "req-ghi-789",
  "request_type": "test_execution",
  "payload": {
    "test_pattern": "TestAuth*",
    "framework": "go",
    "timeout_seconds": 300
  },
  "timeout_ms": 30000,
  "timestamp": "2026-02-05T20:00:00Z"
}
```

**Response:**
```json
{
  "type": "response",
  "from_agent_id": "agent-qa-engineer-2",
  "to_agent_id": "agent-engineer-1",
  "message_id": "resp-jkl-012",
  "in_reply_to": "req-ghi-789",
  "status": "success",
  "payload": {
    "tests_run": 15,
    "tests_passed": 14,
    "tests_failed": 1,
    "duration_ms": 2340,
    "failures": [
      {"test": "TestAuthTimeout", "error": "timeout exceeded"}
    ]
  },
  "timestamp": "2026-02-05T20:00:42Z"
}
```

**Request Types:**
- `test_execution`: Run tests
- `code_review`: Review code
- `build_project`: Build project
- `security_scan`: Security analysis
- `decision_input`: Request input for decision
- `status_check`: Check agent status
- `delegate_task`: Delegate a task

### 4. Notification

Async notification about events (no response expected).

```json
{
  "type": "notification",
  "from_agent_id": "agent-code-reviewer-1",
  "broadcast": true,
  "message_id": "notif-mno-345",
  "notification_type": "bead_completed",
  "payload": {
    "bead_id": "bead-abc-123",
    "result": "approved",
    "pr_number": 456
  },
  "timestamp": "2026-02-05T20:00:00Z"
}
```

**Notification Types:**
- `bead_completed`: Bead work finished
- `bead_blocked`: Bead blocked
- `pr_opened`: Pull request created
- `build_failed`: Build failure
- `deployment_completed`: Deployment finished
- `agent_available`: Agent became available
- `agent_busy`: Agent started work

### 5. Consensus Request

Request consensus from multiple agents.

```json
{
  "type": "consensus_request",
  "from_agent_id": "agent-project-manager-1",
  "to_agent_ids": [
    "agent-engineer-1",
    "agent-qa-engineer-1",
    "agent-code-reviewer-1"
  ],
  "message_id": "cons-pqr-678",
  "decision": {
    "id": "dec-stu-901",
    "question": "Should we refactor the authentication module?",
    "options": ["yes", "no", "defer"],
    "context": {
      "bead_id": "bead-vwx-234",
      "estimated_effort": "3 days",
      "risk_level": "medium"
    }
  },
  "voting_deadline": "2026-02-06T20:00:00Z",
  "consensus_threshold": 0.66,
  "timestamp": "2026-02-05T20:00:00Z"
}
```

**Consensus Response:**
```json
{
  "type": "consensus_vote",
  "from_agent_id": "agent-engineer-1",
  "message_id": "vote-yza-567",
  "in_reply_to": "cons-pqr-678",
  "vote": "yes",
  "rationale": "Authentication module has technical debt that impacts security. Refactoring will improve maintainability.",
  "confidence": 0.85,
  "timestamp": "2026-02-05T20:15:00Z"
}
```

## Agent Discovery

### Agent Registry

Agents register with the system on startup:

```json
{
  "agent_id": "agent-qa-engineer-1",
  "persona_type": "qa-engineer",
  "capabilities": [
    "run_tests",
    "review_test_coverage",
    "write_integration_tests"
  ],
  "status": "available",
  "current_bead": null,
  "max_concurrent_beads": 3,
  "subscription_filters": {
    "message_types": ["request", "broadcast", "consensus_request"],
    "topics": ["testing", "quality", "ci-cd"]
  },
  "last_heartbeat": "2026-02-05T20:00:00Z"
}
```

### Discovery API

**List Available Agents:**
```
GET /api/v1/agents?status=available&persona_type=qa-engineer
```

**Get Agent Details:**
```
GET /api/v1/agents/{agent_id}
```

**Find Agents by Capability:**
```
GET /api/v1/agents?capability=run_tests
```

## Message Routing

### Routing Rules

1. **Direct Messages**: Route to specific agent by ID
2. **Broadcast**: Route to all agents matching filters
3. **Request-Response**: Route request, track correlation ID
4. **Consensus**: Route to specified agent list, collect responses

### Message Bus Implementation

Uses existing Event Bus infrastructure with agent-specific topics:

```
Topic: agent.messages.{agent_id}
Topic: agent.broadcast
Topic: agent.consensus.{decision_id}
```

### Message Persistence

All messages stored in database for:
- Audit trail
- Context reconstruction
- Conversation history
- Debugging

**Schema:**
```sql
CREATE TABLE agent_messages (
    message_id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    from_agent_id VARCHAR(64) NOT NULL,
    to_agent_id VARCHAR(64),
    subject TEXT,
    body TEXT,
    payload JSONB,
    priority VARCHAR(16),
    requires_response BOOLEAN,
    in_reply_to VARCHAR(64),
    status VARCHAR(32),
    created_at TIMESTAMP NOT NULL,
    delivered_at TIMESTAMP,
    read_at TIMESTAMP
);
```

## Request-Response Patterns

### Synchronous Pattern (with timeout)

```go
// Send request
response, err := agent.SendRequest(ctx, AgentRequest{
    ToAgentID: "agent-qa-1",
    RequestType: "test_execution",
    Payload: testConfig,
    Timeout: 30 * time.Second,
})

// Wait for response (blocks until response or timeout)
if err != nil {
    // Handle timeout or error
}
```

### Asynchronous Pattern (callback)

```go
// Send request with callback
agent.SendRequestAsync(ctx, AgentRequest{
    ToAgentID: "agent-qa-1",
    RequestType: "test_execution",
    Payload: testConfig,
}, func(response AgentResponse) {
    // Handle response when it arrives
    log.Printf("Tests completed: %v", response.Payload)
})

// Continue with other work
```

### Fire-and-Forget Pattern

```go
// Send notification (no response expected)
agent.SendNotification(ctx, AgentNotification{
    Type: "bead_completed",
    Broadcast: true,
    Payload: map[string]interface{}{
        "bead_id": beadID,
        "result": "success",
    },
})
```

## Agent Action: send_agent_message

New action for agents to send messages:

```json
{
  "type": "send_agent_message",
  "to_agent_id": "agent-code-reviewer-2",
  "subject": "Code review request",
  "body": "Please review the authentication module changes.",
  "requires_response": true,
  "context": {
    "bead_id": "bead-abc-123",
    "files": ["src/auth.go"]
  }
}
```

**Action Response:**
```json
{
  "action_type": "send_agent_message",
  "status": "executed",
  "message": "Message sent to agent-code-reviewer-2",
  "metadata": {
    "message_id": "msg-xyz-789",
    "delivered_at": "2026-02-05T20:00:01Z"
  }
}
```

## Async Notification System

### Event-Driven Notifications

Agents subscribe to topics of interest:

```go
agent.Subscribe("bead.status.completed", func(notification Notification) {
    // Handle bead completion
})

agent.Subscribe("pr.opened", func(notification Notification) {
    // Handle new PR
})

agent.Subscribe("build.failed", func(notification Notification) {
    // Handle build failure
})
```

### Notification Delivery Guarantees

- **At-least-once delivery**: Messages may be delivered multiple times
- **Idempotent handlers**: Agents must handle duplicate notifications
- **Ordering**: Best-effort ordering within same topic
- **Persistence**: Notifications stored for 7 days

### Dead Letter Queue

Failed message deliveries go to dead letter queue:

```
Topic: agent.messages.dlq
```

Messages in DLQ after 3 retry attempts.

## Shared Context

### Bead Context Sharing

Multiple agents working on same bead share context:

```json
{
  "bead_id": "bead-abc-123",
  "collaborating_agents": [
    "agent-engineer-1",
    "agent-qa-1",
    "agent-code-reviewer-1"
  ],
  "shared_context": {
    "files_modified": ["src/auth.go", "src/auth_test.go"],
    "test_results": {"passed": 14, "failed": 1},
    "review_status": "in_progress",
    "decision_needed": false
  },
  "message_history": [
    {
      "from": "agent-engineer-1",
      "to": "agent-qa-1",
      "subject": "Tests ready for execution",
      "timestamp": "2026-02-05T20:00:00Z"
    }
  ]
}
```

### Context Access API

```
GET /api/v1/beads/{bead_id}/context
POST /api/v1/beads/{bead_id}/context
PATCH /api/v1/beads/{bead_id}/context
```

## Security & Privacy

### Message Authentication

- All messages signed with agent private key
- Recipients verify signature
- Prevents message spoofing

### Authorization

- Agents can only send messages if authorized
- Permission model: agent roles and capabilities
- Project-scoped permissions

### Message Encryption

- Sensitive payloads encrypted at rest
- TLS for message transmission
- End-to-end encryption option for sensitive contexts

### Audit Logging

All inter-agent communication logged:
- Who sent what to whom
- When and why
- Message content (sanitized if sensitive)
- Response times and outcomes

## Performance Considerations

### Message Throughput

- Target: 10,000 messages/second
- Async processing via event bus
- Message batching for broadcasts

### Latency

- Direct messages: < 100ms p99
- Request-response: < 1s p99 (depends on work)
- Broadcasts: < 500ms to all agents

### Scalability

- Horizontal scaling via message bus partitioning
- Agent discovery caching
- Message history pruning (> 7 days)

## Error Handling

### Delivery Failures

```json
{
  "error": "delivery_failed",
  "message_id": "msg-abc-123",
  "reason": "agent_unavailable",
  "retry_count": 2,
  "next_retry_at": "2026-02-05T20:05:00Z"
}
```

### Timeout Handling

```json
{
  "error": "timeout",
  "message_id": "req-def-456",
  "timeout_ms": 30000,
  "elapsed_ms": 30124
}
```

### Response Errors

```json
{
  "type": "response",
  "status": "error",
  "error": {
    "code": "test_execution_failed",
    "message": "Build failed before tests could run",
    "details": {...}
  }
}
```

## Examples

### Example 1: Engineer Requests QA Testing

**Engineer sends request:**
```json
{
  "type": "send_agent_message",
  "to_agent_id": "agent-qa-engineer-1",
  "subject": "Test authentication module",
  "body": "I've implemented authentication changes. Please run integration tests.",
  "requires_response": true,
  "context": {
    "bead_id": "bead-auth-feature",
    "files": ["src/auth.go", "src/auth_test.go"],
    "branch": "agent/bead-auth-feature/implement-jwt"
  }
}
```

**QA Engineer responds:**
```json
{
  "type": "send_agent_message",
  "to_agent_id": "agent-engineer-1",
  "subject": "Re: Test authentication module",
  "body": "Tests completed. 14/15 passed. 1 test failing: TestAuthTimeout needs attention.",
  "context": {
    "bead_id": "bead-auth-feature",
    "test_results": {
      "passed": 14,
      "failed": 1,
      "failures": ["TestAuthTimeout: timeout exceeded"]
    }
  }
}
```

### Example 2: Consensus Decision

**Project Manager requests consensus:**
```json
{
  "type": "consensus_request",
  "to_agent_ids": ["agent-engineer-1", "agent-qa-1", "agent-reviewer-1"],
  "decision": {
    "question": "Approve refactoring of auth module?",
    "options": ["approve", "reject", "defer"],
    "context": {
      "effort": "3 days",
      "risk": "medium",
      "benefits": "improved security and maintainability"
    }
  },
  "consensus_threshold": 0.66
}
```

**Agents vote:**
- Engineer: "approve" (confidence: 0.9)
- QA: "approve" (confidence: 0.8)
- Reviewer: "approve" (confidence: 0.85)

**Result: Consensus reached (3/3 = 100% > 66%)**

### Example 3: Broadcast Notification

**Code Reviewer broadcasts PR completion:**
```json
{
  "type": "broadcast",
  "subject": "PR #456 approved",
  "body": "Code review complete for authentication module. All checks passed. Ready to merge.",
  "filter": {
    "topics": ["pr-updates"]
  },
  "payload": {
    "pr_number": 456,
    "bead_id": "bead-auth-feature",
    "status": "approved"
  }
}
```

## Future Enhancements

1. **Rich Media Messages**: Support for images, diagrams, code snippets
2. **Message Threading**: Conversation threads like email
3. **Presence Indicators**: Real-time agent availability status
4. **Message Search**: Full-text search across message history
5. **Analytics**: Message volume, response times, collaboration metrics
6. **AI Summarization**: Auto-summarize long message threads

## References

- Event Bus implementation: `internal/temporal/eventbus/`
- Database schema: `internal/database/migrations/`
- Agent actions: `internal/actions/schema.go`
- Agent registry: `internal/agent/manager.go`
