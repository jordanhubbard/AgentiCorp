# Perpetual Tasks System

## Overview

The **Perpetual Tasks System** enables proactive agent workflows by automatically generating scheduled work for idle agents. Instead of agents waiting passively for beads to be assigned, agents with perpetual tasks actively seek out work based on their role and schedule.

## Architecture

Perpetual tasks are implemented through the **Motivation System** (`internal/motivation/`) using scheduled interval triggers. They run independently of system idle state or external events.

### Key Components

1. **Motivation Engine** (`internal/motivation/engine.go`)
   - Evaluates motivations on a regular tick interval
   - Fires motivations when conditions are met
   - Creates stimulus beads for perpetual tasks

2. **Perpetual Task Motivations** (`internal/motivation/perpetual.go`)
   - Defines scheduled tasks for each org chart role
   - Configures intervals (hourly, daily, weekly)
   - Specifies bead templates and priorities

3. **Calendar Evaluator** (`internal/motivation/evaluators.go`)
   - Handles `ConditionScheduledInterval` triggers
   - Checks if enough time has passed since last trigger
   - Supports configurable intervals via parameters

## Perpetual Tasks by Role

### CFO (Chief Financial Officer)
- **Daily Budget Review** (every 24 hours)
  - Reviews daily spending and budget utilization
  - Priority: 70
  - Bead Template: `daily-budget-review`

- **Weekly Cost Optimization Report** (every 7 days)
  - Analyzes cost trends and identifies optimization opportunities
  - Priority: 65
  - Bead Template: `weekly-cost-report`

### QA Engineer
- **Daily Automated Test Suite Run** (every 24 hours)
  - Runs full automated test suite to ensure quality
  - Priority: 75
  - Bead Template: `daily-test-run`

- **Weekly Integration Test Review** (every 7 days)
  - Performs comprehensive integration testing
  - Priority: 70
  - Bead Template: `weekly-integration-tests`

- **Weekly Regression Test Sweep** (every 7 days)
  - Runs regression tests to catch regressions early
  - Priority: 72
  - Bead Template: `weekly-regression-tests`

### PR Manager (Public Relations Manager)
- **Hourly GitHub Activity Check** (every 1 hour)
  - Polls GitHub for new issues, PRs, and comments
  - Priority: 60
  - Bead Template: `github-activity-check`

- **Daily Community Engagement Report** (every 24 hours)
  - Reviews and reports on community engagement metrics
  - Priority: 55
  - Bead Template: `daily-community-report`

### Documentation Manager
- **Daily Documentation Audit** (every 24 hours)
  - Reviews and updates documentation daily
  - Priority: 50
  - Bead Template: `daily-docs-audit`

- **Weekly Documentation Consistency Check** (every 7 days)
  - Ensures documentation consistency across the project
  - Priority: 55
  - Bead Template: `weekly-docs-consistency`

### DevOps Engineer
- **Daily Infrastructure Health Check** (every 24 hours)
  - Performs daily infrastructure health and monitoring review
  - Priority: 75
  - Bead Template: `daily-infra-health`

- **Weekly Security Audit** (every 7 days)
  - Performs weekly security audit and vulnerability scanning
  - Priority: 80
  - Bead Template: `weekly-security-audit`

### Project Manager
- **Daily Standup Review** (every 24 hours)
  - Reviews daily progress, blockers, and team status
  - Priority: 70
  - Bead Template: `daily-standup`

- **Weekly Sprint Planning** (every 7 days)
  - Conducts weekly sprint planning and retrospective
  - Priority: 75
  - Bead Template: `weekly-sprint-planning`

### Housekeeping Bot
- **Hourly Cleanup Tasks** (every 1 hour)
  - Performs routine cleanup tasks
  - Priority: 30
  - Bead Template: `hourly-cleanup`

- **Weekly Data Archival** (every 7 days)
  - Archives old data and logs
  - Priority: 35
  - Bead Template: `weekly-archival`

## How It Works

### 1. Motivation Registration

Perpetual tasks are registered when the motivation engine starts:

```go
import "github.com/jordanhubbard/loom/internal/motivation"

// Register default motivations (includes perpetual tasks)
err := motivation.RegisterDefaults(registry)
```

### 2. Evaluation Loop

The motivation engine runs a tick loop (default: every 30 seconds):

```go
// Engine evaluates all active motivations
for _, motivation := range registry.GetActive() {
    shouldFire, triggerData, err := engine.evaluate(ctx, motivation)
    if shouldFire {
        engine.fire(ctx, motivation, triggerData)
    }
}
```

### 3. Scheduled Interval Evaluation

For perpetual tasks with `ConditionScheduledInterval`:

```go
// Check if enough time has passed since last trigger
interval := motivation.Parameters["interval"] // e.g., "24h"
if now.Sub(lastTriggeredAt) >= interval {
    // Fire the motivation
    return true, triggerData, nil
}
```

### 4. Bead Creation

When a perpetual task fires:

1. **Creates a stimulus bead** using the configured `BeadTemplate`
2. **Wakes the target agent** (by role)
3. **Records the trigger** for history and cooldown tracking
4. **Publishes an event** to the event bus

### 5. Agent Assignment

The dispatcher assigns the created bead to an available agent with the matching role:

```go
// Agent picks up the bead
bead := dispatcher.GetNextBeadForRole(agentRole)
agent.ExecuteBead(bead)
```

## Configuration

### Motivation Structure

```go
{
    Name:        "Daily Budget Review",
    Description: "CFO reviews daily spending every 24 hours",
    Type:        MotivationTypeCalendar,
    Condition:   ConditionScheduledInterval,
    AgentRole:   "cfo",
    WakeAgent:   true,
    CreateBeadOnTrigger: true,
    BeadTemplate: "daily-budget-review",
    Priority:    70,
    CooldownPeriod: 22 * time.Hour,
    Parameters: map[string]interface{}{
        "interval": "24h",
        "task_type": "perpetual",
    },
    IsBuiltIn: true,
}
```

### Key Parameters

- **interval**: How often the task should run (e.g., "1h", "24h", "168h")
- **task_type**: Set to "perpetual" for filtering and identification
- **CooldownPeriod**: Slightly less than interval to avoid drift (e.g., 22h for 24h tasks)

### Custom Intervals

To create a custom perpetual task:

```go
motivation := &Motivation{
    Name:        "Custom Daily Task",
    Type:        MotivationTypeCalendar,
    Condition:   ConditionScheduledInterval,
    AgentRole:   "custom-role",
    CreateBeadOnTrigger: true,
    BeadTemplate: "custom-task",
    CooldownPeriod: 22 * time.Hour,
    Parameters: map[string]interface{}{
        "interval": "24h",
        "task_type": "perpetual",
    },
}
```

## Bead Templates

Each perpetual task references a **bead template** that defines the work to be done. Bead templates should be implemented in the corresponding agent's persona or bead creation logic.

Example template names:
- `daily-budget-review` - CFO daily budget analysis
- `daily-test-run` - QA daily test execution
- `github-activity-check` - PR Manager GitHub polling
- `daily-docs-audit` - Documentation Manager review

## Benefits

1. **Proactive Workflows**: Agents don't wait for work; they create it
2. **Consistent Execution**: Tasks run on reliable schedules
3. **Role-Based**: Each role has appropriate perpetual tasks
4. **Configurable**: Easy to add new tasks or adjust intervals
5. **Observable**: All triggers are recorded and can be audited

## Monitoring

### View Perpetual Task History

```bash
# Get trigger history for a motivation
curl http://localhost:8080/api/v1/motivations/{id}/history

# Get all motivations by role
curl http://localhost:8080/api/v1/motivations?role=cfo
```

### Query Perpetual Tasks

```go
// Get all perpetual tasks for a specific role
tasks := motivation.GetPerpetualTasksByRole("cfo")

// Get all perpetual tasks
allTasks := motivation.PerpetualTaskMotivations()
```

## Troubleshooting

### Perpetual Task Not Firing

1. **Check motivation is active**:
   ```go
   motivation, _ := registry.Get(motivationID)
   if motivation.Status != MotivationStatusActive {
       registry.Enable(motivationID)
   }
   ```

2. **Verify interval configuration**:
   ```go
   interval := motivation.Parameters["interval"]
   // Should be a valid duration string like "24h"
   ```

3. **Check cooldown period**:
   ```go
   // Cooldown should be less than interval
   if motivation.CooldownPeriod >= interval {
       // Adjust cooldown
   }
   ```

4. **Examine trigger history**:
   ```go
   history := registry.GetTriggerHistory(10)
   // Check last trigger time and errors
   ```

### Perpetual Task Firing Too Often

- Increase the `CooldownPeriod`
- Verify the `interval` parameter is correct
- Check for duplicate motivation registrations

### Agent Not Picking Up Perpetual Task Beads

1. **Verify agent role matches**:
   ```go
   agent.Role == motivation.AgentRole
   ```

2. **Check agent status**:
   ```go
   // Agent must be idle to pick up new beads
   agent.Status == "idle"
   ```

3. **Verify bead template exists**:
   ```go
   // Bead template must be implemented
   beadTemplate := motivation.BeadTemplate
   ```

## Testing

Run perpetual task tests:

```bash
go test ./internal/motivation -v -run TestPerpetual
```

Tests verify:
- ✅ All perpetual tasks have required fields
- ✅ Intervals are properly configured
- ✅ Cooldowns are appropriate
- ✅ All key roles have perpetual tasks
- ✅ Registration works correctly

## Future Enhancements

Potential improvements:
- **Dynamic scheduling**: Adjust intervals based on system load
- **Conditional execution**: Skip tasks if certain conditions aren't met
- **Priority-based scheduling**: Higher priority tasks fire first
- **Task dependencies**: Chain perpetual tasks together
- **Custom evaluators**: Role-specific evaluation logic

## See Also

- [Motivation System Overview](./motivation-system.md)
- [Agent Roles](./agent-roles.md)
- [Bead Management](./beads.md)
- [Dispatcher Architecture](./dispatcher.md)
