## Shared Bead Context

The Shared Bead Context system enables multiple agents to collaborate on the same bead by sharing real-time state, activity logs, and data updates with automatic conflict resolution.

## Overview

When multiple agents work on the same bead, they need to coordinate their efforts, share progress, and avoid conflicting changes. The Shared Bead Context provides:

- **Real-time Updates**: Agents see each other's actions via Server-Sent Events (SSE)
- **Shared Data Store**: Key-value store for sharing state between agents
- **Activity Log**: Complete history of all agent actions on the bead
- **Conflict Resolution**: Optimistic locking with version numbers prevents lost updates
- **Agent Discovery**: See which agents are currently working on a bead

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Agent A     │────▶│  Context     │◀────│  Agent B     │
│              │     │  Store       │     │              │
│ - Join       │     │              │     │ - Subscribe  │
│ - Update     │     │ - Storage    │     │ - Receive    │
│ - Subscribe  │     │ - Locking    │     │ - Update     │
└──────────────┘     │ - Events     │     └──────────────┘
                     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  SSE Handler │
                     │ (Real-time)  │
                     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │ Web Clients  │
                     │ (Dashboard)  │
                     └──────────────┘
```

## Core Concepts

### Shared Context

Each bead has a shared context containing:

- **Collaborating Agents**: List of agents currently working on the bead
- **Shared Data**: Key-value pairs for sharing state (test results, file modifications, etc.)
- **Activity Log**: Timeline of all agent actions
- **Version Number**: For optimistic locking and conflict detection

### Activity Types

Standard activity types logged in the context:

- `joined`: Agent joined collaboration
- `left`: Agent left collaboration
- `updated`: Agent updated shared data
- `file_modified`: Agent modified a file
- `test_completed`: Agent finished running tests
- `review_started`: Agent started code review
- `message`: Agent posted a message

### Conflict Resolution

Uses **optimistic locking** with version numbers:

1. Agent reads context with version N
2. Agent makes changes locally
3. Agent updates context, providing expected version N
4. If current version is still N, update succeeds (version becomes N+1)
5. If current version != N, update fails with ConflictError
6. Agent must re-read context and retry

## Usage

### Joining a Bead

```go
import "github.com/jordanhubbard/loom/internal/collaboration"

store := collaboration.NewContextStore()

// Create or get context
ctx, err := store.GetOrCreate(context.Background(), "bead-abc-123", "project-1")

// Agent joins
err = store.JoinBead(context.Background(), "bead-abc-123", "agent-engineer-1")
```

### Updating Shared Data

```go
// Update without version check (no conflict detection)
err := store.UpdateData(
    context.Background(),
    "bead-abc-123",
    "agent-engineer-1",
    "test_status",
    "passed",
    0, // 0 = skip version check
)

// Update with version check (conflict detection)
currentVersion := ctx.Version
err := store.UpdateData(
    context.Background(),
    "bead-abc-123",
    "agent-engineer-1",
    "coverage",
    "92%",
    currentVersion,
)

// Handle conflict
if conflictErr, ok := err.(*collaboration.ConflictError); ok {
    // Another agent updated the context
    // Re-read and retry
    ctx, _ := store.Get(context.Background(), "bead-abc-123")
    // ... retry update
}
```

### Adding Activity

```go
err := store.AddActivity(
    context.Background(),
    "bead-abc-123",
    "agent-qa-1",
    "test_completed",
    "Ran 45 tests, all passed",
    map[string]interface{}{
        "tests_run": 45,
        "tests_passed": 45,
        "duration_ms": 2340,
    },
)
```

### Subscribing to Updates

```go
// Subscribe to real-time updates
updateChan := store.Subscribe("bead-abc-123")
defer store.Unsubscribe("bead-abc-123", updateChan)

// Process updates
for update := range updateChan {
    switch update.UpdateType {
    case "joined":
        fmt.Printf("Agent %s joined\n", update.AgentID)
    case "data_changed":
        fmt.Printf("Agent %s updated %s\n", update.AgentID, update.Data["key"])
    case "activity":
        fmt.Printf("Activity: %s\n", update.Data["description"])
    }
}
```

### Leaving a Bead

```go
err := store.LeaveBead(context.Background(), "bead-abc-123", "agent-engineer-1")
```

## HTTP API (SSE)

### Stream Context Updates

```http
GET /api/v1/beads/{bead_id}/context/stream
```

Server-Sent Events stream providing real-time updates.

**Response (SSE format):**

```
event: initial
data: {"type":"initial","bead_id":"bead-abc-123","context":{...}}

event: update
data: {"bead_id":"bead-abc-123","update_type":"joined","agent_id":"agent-eng-1"}

event: update
data: {"bead_id":"bead-abc-123","update_type":"data_changed","data":{"key":"status","value":"running"}}

: ping
```

### Get Current Context

```http
GET /api/v1/beads/{bead_id}/context?bead_id=bead-abc-123
```

Returns the current context state as JSON.

**Response:**

```json
{
  "bead_id": "bead-abc-123",
  "project_id": "project-1",
  "collaborating_agents": ["agent-eng-1", "agent-qa-1"],
  "data": {
    "test_status": "passed",
    "coverage": "92%",
    "build_status": "success"
  },
  "activity_log": [
    {
      "timestamp": "2026-02-05T20:00:00Z",
      "agent_id": "agent-eng-1",
      "activity_type": "joined",
      "description": "Agent agent-eng-1 joined collaboration"
    },
    {
      "timestamp": "2026-02-05T20:01:30Z",
      "agent_id": "agent-qa-1",
      "activity_type": "test_completed",
      "description": "Ran 45 tests, all passed",
      "data": {
        "tests_run": 45,
        "tests_passed": 45
      }
    }
  ],
  "version": 5,
  "last_updated": "2026-02-05T20:01:30Z",
  "last_updated_by": "agent-qa-1"
}
```

### Join Bead

```http
POST /api/v1/beads/context/join
Content-Type: application/json

{
  "bead_id": "bead-abc-123",
  "agent_id": "agent-eng-1"
}
```

**Response:**

```json
{
  "status": "joined",
  "bead_id": "bead-abc-123",
  "agent_id": "agent-eng-1"
}
```

### Leave Bead

```http
POST /api/v1/beads/context/leave
Content-Type: application/json

{
  "bead_id": "bead-abc-123",
  "agent_id": "agent-eng-1"
}
```

### Update Data

```http
POST /api/v1/beads/context/update
Content-Type: application/json

{
  "bead_id": "bead-abc-123",
  "agent_id": "agent-eng-1",
  "key": "test_status",
  "value": "running",
  "expected_version": 5
}
```

**Success Response:**

```json
{
  "status": "updated",
  "bead_id": "bead-abc-123",
  "key": "test_status",
  "version": 6
}
```

**Conflict Response (409):**

```json
{
  "error": "version_conflict",
  "expected_version": 5,
  "actual_version": 7
}
```

### Add Activity

```http
POST /api/v1/beads/context/activity
Content-Type: application/json

{
  "bead_id": "bead-abc-123",
  "agent_id": "agent-qa-1",
  "activity_type": "test_completed",
  "description": "Ran integration tests",
  "data": {
    "tests_run": 25,
    "tests_passed": 24,
    "tests_failed": 1
  }
}
```

## Use Cases

### 1. Collaborative Code Review

```go
// Reviewer joins
store.JoinBead(ctx, beadID, "agent-reviewer-1")

// Update review status
store.UpdateData(ctx, beadID, "agent-reviewer-1", "review_status", "in_progress", 0)

// Log file review
store.AddActivity(ctx, beadID, "agent-reviewer-1", "file_reviewed",
    "Reviewed auth.go - found 2 issues",
    map[string]interface{}{
        "file": "src/auth.go",
        "issues_found": 2,
    })

// Engineer sees update in real-time via SSE
// Engineer fixes issues

// Engineer updates status
store.UpdateData(ctx, beadID, "agent-engineer-1", "issues_fixed", true, 0)

// Reviewer sees notification, re-reviews

// Final approval
store.UpdateData(ctx, beadID, "agent-reviewer-1", "review_status", "approved", 0)
```

### 2. Parallel Testing

```go
// QA agent 1 runs unit tests
store.AddActivity(ctx, beadID, "agent-qa-1", "test_started",
    "Running unit tests", nil)

// QA agent 2 runs integration tests (simultaneously)
store.AddActivity(ctx, beadID, "agent-qa-2", "test_started",
    "Running integration tests", nil)

// Both update shared results
store.UpdateData(ctx, beadID, "agent-qa-1", "unit_tests",
    map[string]interface{}{"passed": 120, "failed": 0}, 0)

store.UpdateData(ctx, beadID, "agent-qa-2", "integration_tests",
    map[string]interface{}{"passed": 35, "failed": 2}, 0)

// Engineer sees both results in real-time
```

### 3. Build Pipeline Coordination

```go
// Builder agent
store.UpdateData(ctx, beadID, "agent-builder-1", "build_status", "running", 0)

// After build completes
store.UpdateData(ctx, beadID, "agent-builder-1", "build_status", "success", 0)
store.AddActivity(ctx, beadID, "agent-builder-1", "build_completed",
    "Build completed successfully",
    map[string]interface{}{
        "duration_ms": 45000,
        "artifacts": []string{"app.bin", "app.tar.gz"},
    })

// Deployer agent sees build complete, starts deployment
store.UpdateData(ctx, beadID, "agent-deployer-1", "deploy_status", "deploying", 0)
```

## Best Practices

### 1. Always Join Before Working

```go
// Correct
store.JoinBead(ctx, beadID, agentID)
// ... do work ...
defer store.LeaveBead(ctx, beadID, agentID)

// Incorrect - other agents won't know you're working on this bead
store.UpdateData(ctx, beadID, agentID, "key", "value", 0)
```

### 2. Use Version Checks for Critical Updates

```go
// Critical: Don't overwrite another agent's test results
ctx, _ := store.Get(context.Background(), beadID)
version := ctx.Version

err := store.UpdateData(ctx, beadID, agentID, "final_status", "approved", version)
if conflictErr, ok := err.(*collaboration.ConflictError); ok {
    // Someone else updated, re-check before approving
}

// Non-critical: It's okay to overwrite
store.UpdateData(ctx, beadID, agentID, "last_heartbeat", time.Now(), 0)
```

### 3. Log Meaningful Activities

```go
// Good - specific and actionable
store.AddActivity(ctx, beadID, agentID, "test_failed",
    "TestAuthentication failed: timeout after 30s",
    map[string]interface{}{
        "test": "TestAuthentication",
        "error": "timeout",
        "duration_ms": 30000,
    })

// Bad - vague
store.AddActivity(ctx, beadID, agentID, "update", "Did something", nil)
```

### 4. Handle SSE Reconnections

```javascript
// Client-side SSE with auto-reconnect
function connectSSE(beadID) {
  const eventSource = new EventSource(`/api/v1/beads/${beadID}/context/stream`);

  eventSource.addEventListener('initial', (e) => {
    const data = JSON.parse(e.data);
    updateUI(data.context);
  });

  eventSource.addEventListener('update', (e) => {
    const update = JSON.parse(e.data);
    handleUpdate(update);
  });

  eventSource.onerror = () => {
    eventSource.close();
    // Reconnect after delay
    setTimeout(() => connectSSE(beadID), 5000);
  };
}
```

## Performance Considerations

- **Memory Usage**: ~1KB per activity entry, limited to last N entries per bead
- **SSE Connections**: Each connection holds a goroutine and buffered channel
- **Update Latency**: < 100ms from update to all subscribers
- **Concurrent Access**: Thread-safe with RWMutex, optimized for read-heavy workloads

## Implementation Details

### Thread Safety

- All operations use appropriate locking (RWMutex)
- Fine-grained locking per bead context
- Lock-free update distribution via channels

### Conflict Resolution Algorithm

```
1. Read current version: V
2. Make changes locally
3. Attempt update with expected version V
4. Store checks: current version == V?
   - Yes: Apply update, increment version to V+1, broadcast update
   - No: Return ConflictError with actual version
5. On conflict: Re-read, merge changes, retry
```

### Activity Log Retention

- Default: Keep last 1000 activity entries per bead
- Configurable per bead
- Automatic pruning when limit exceeded
- Full history can be exported to persistent storage

## Related Documentation

- [Agent Communication Protocol](AGENT_COMMUNICATION.md)
- [Agent Message Bus](../internal/messaging/README.md)
- [Event Bus](../internal/temporal/eventbus/)

## Future Enhancements

1. **Persistent Storage**: PostgreSQL/Redis backend for contexts
2. **Context Snapshots**: Save/restore context state
3. **Conflict Resolution Strategies**: Automatic merge strategies for common cases
4. **Activity Search**: Query activity logs by type, agent, date
5. **Context Templates**: Pre-defined structures for common workflows
6. **Metrics**: Context access patterns, hot beads, agent collaboration stats
