# Temporal DSL (Domain Specific Language)

The Temporal DSL is an English-like markdown-based syntax that allows agents and providers to request work from Temporal without requiring external provider services. This enables orchestration and complex workflows purely within the Loom system.

## Overview

The DSL is embedded in agent personas and instructions using markdown `<temporal>...</temporal>` blocks. When an agent processes instructions or responds with output containing Temporal DSL, Loom automatically:

1. **Extracts** the DSL blocks from the text
2. **Parses** them into structured instructions
3. **Executes** them via the Temporal manager
4. **Strips** the DSL from the text before sending to providers (hidden notation)

This allows agents to:
- Request workflows without provider availability
- Schedule recurring tasks
- Query running workflows for status
- Send signals to coordinate work
- Execute activities directly
- Request system-level operations

## Syntax

### Basic Structure

```markdown
<temporal>
INSTRUCTION_TYPE: instruction_name
  PARAMETER_NAME: parameter_value
  PARAMETER_NAME: parameter_value
END
</temporal>
```

Multiple instructions can be in a single block:

```markdown
<temporal>
WORKFLOW: TaskProcessor
  INPUT: {"task_id": "123"}
  TIMEOUT: 5m
  WAIT: true
END

SIGNAL: workflow-xyz
  NAME: update_status
  DATA: {"status": "approved"}
END
</temporal>
```

## Instruction Types

### 1. WORKFLOW - Schedule a Workflow

Schedules a workflow execution on Temporal.

**Syntax:**
```markdown
<temporal>
WORKFLOW: workflow_name
  INPUT: {"key": "value", "nested": {"data": "here"}}
  TIMEOUT: 5m
  RETRY: 3
  WAIT: true
  PRIORITY: 10
  IDEMPOTENCY_KEY: "unique-key"
END
</temporal>
```

**Parameters:**
- `INPUT` (JSON, optional): Input parameters for the workflow as a JSON object
- `TIMEOUT` (duration, optional): Maximum time to wait for completion (default: 5m)
  - Formats: `30s`, `5m`, `2h`, `immediate`, `default`
- `RETRY` (integer, optional): Number of retry attempts (default: 0)
- `WAIT` (true/false, optional): Whether to wait for completion (default: false)
- `PRIORITY` (integer, optional): Workflow priority (default: 0)
- `IDEMPOTENCY_KEY` (string, optional): For idempotent execution

**Examples:**
```markdown
Process a task immediately:
<temporal>
WORKFLOW: ProcessTask
  INPUT: {"task_id": "456", "priority": "high"}
  TIMEOUT: 10m
  WAIT: true
END
</temporal>

Schedule background processing without waiting:
<temporal>
WORKFLOW: BackgroundAnalysis
  INPUT: {"data_set": "large"}
  WAIT: false
END
</temporal>
```

### 2. SCHEDULE - Create Recurring Schedule

Creates a recurring schedule that repeatedly executes a workflow.

**Syntax:**
```markdown
<temporal>
SCHEDULE: schedule_name
  INPUT: {"config": "value"}
  INTERVAL: 30m
  TIMEOUT: 5m
  RETRY: 2
END
</temporal>
```

**Parameters:**
- `INPUT` (JSON, optional): Input for each workflow execution
- `INTERVAL` (duration, required): How often to run (e.g., `1h`, `30m`, `15s`)
- `TIMEOUT` (duration, optional): Timeout for each execution
- `RETRY` (integer, optional): Retries per execution

**Examples:**
```markdown
Monitor provider health every 30 seconds:
<temporal>
SCHEDULE: ProviderHealthCheck
  INTERVAL: 30s
  INPUT: {"check_type": "full"}
END
</temporal>

Daily budget review:
<temporal>
SCHEDULE: DailyBudgetReview
  INTERVAL: 24h
  INPUT: {"org_id": "acme"}
END
</temporal>
```

### 3. QUERY - Query a Running Workflow

Queries a running workflow to get its status or data.

**Syntax:**
```markdown
<temporal>
QUERY: workflow_id
  TYPE: query_type
END
</temporal>
```

**Parameters:**
- `TYPE` (string, required): The type of query to run
- `WORKFLOW_ID` (string, required - the instruction name is actually the workflow ID)

**Examples:**
```markdown
Check task status:
<temporal>
QUERY: task-workflow-123
  TYPE: get_status
END
</temporal>

Get progress:
<temporal>
QUERY: processing-456
  TYPE: get_progress
END
</temporal>
```

### 4. SIGNAL - Send Signal to Workflow

Sends a signal to a running workflow to trigger an action or provide data.

**Syntax:**
```markdown
<temporal>
SIGNAL: workflow_id
  NAME: signal_name
  DATA: {"key": "value"}
END
</temporal>
```

**Parameters:**
- `NAME` (string, required): The signal name
- `DATA` (JSON, optional): Data to send with the signal
- `WORKFLOW_ID` (string, required - the instruction name is the workflow ID)

**Examples:**
```markdown
Approve a workflow:
<temporal>
SIGNAL: approval-wf-789
  NAME: approve
  DATA: {"approved_by": "cfo", "amount": 50000}
END
</temporal>

Pause processing:
<temporal>
SIGNAL: processor-001
  NAME: pause
END
</temporal>

Resume with new config:
<temporal>
SIGNAL: processor-001
  NAME: resume
  DATA: {"config": "optimized"}
END
</temporal>
```

### 5. ACTIVITY - Execute Activity Directly

Executes an activity synchronously and waits for the result.

**Syntax:**
```markdown
<temporal>
ACTIVITY: activity_name
  INPUT: {"param": "value"}
  TIMEOUT: 2m
  RETRY: 2
END
</temporal>
```

**Parameters:**
- `INPUT` (JSON, optional): Activity input
- `TIMEOUT` (duration, optional): Execution timeout (default: 2m)
- `RETRY` (integer, optional): Retry attempts

**Examples:**
```markdown
Call external service:
<temporal>
ACTIVITY: FetchMarketData
  INPUT: {"symbol": "AAPL", "period": "daily"}
  TIMEOUT: 30s
END
</temporal>
```

### 6. CANCEL - Cancel a Workflow

Cancels a running workflow execution.

**Syntax:**
```markdown
<temporal>
CANCEL: workflow_id
END
</temporal>
```

**Examples:**
```markdown
Stop processing:
<temporal>
CANCEL: long-running-task-123
END
</temporal>
```

### 7. LIST - List Running Workflows

Lists all running workflows (currently returns count).

**Syntax:**
```markdown
<temporal>
LIST
END
</temporal>
```

**Examples:**
```markdown
<temporal>
LIST
END
</temporal>
```

## Real-World Examples

### Agent Persona with DSL

**CEO Role:**
```markdown
You are the Chief Executive Officer of Loom.

When reviewing strategic initiatives:
<temporal>
WORKFLOW: GetStrategicInitiatives
  INPUT: {"org_id": "acme", "status": "active"}
  TIMEOUT: 3m
  WAIT: true
END
</temporal>

Analyze the data and make strategic decisions. When approving major initiatives:

<temporal>
WORKFLOW: LogStrategicDecision
  INPUT: {"decision_type": "capital_allocation"}
  WAIT: false
END
</temporal>
```

**CFO Role:**
```markdown
You are the Chief Financial Officer.

Before approving budget changes:
<temporal>
WORKFLOW: ValidateBudget
  INPUT: {"amount": 100000, "department": "engineering"}
  TIMEOUT: 2m
  WAIT: true
END
</temporal>

Validate the request against current budget constraints.

When budget is approved:
<temporal>
SIGNAL: budget-approval-workflow-${workflow_id}
  NAME: approve
  DATA: {"approved_by": "cfo", "effective_date": "2026-01-20"}
END
</temporal>

Log the approval:
<temporal>
WORKFLOW: LogBudgetApproval
  INPUT: {"action": "approved", "amount": 100000}
  WAIT: false
END
</temporal>
```

**DevOps Engineer Role:**
```markdown
You are the DevOps Engineer responsible for infrastructure health.

Regularly check provider health:
<temporal>
SCHEDULE: ProviderHealthMonitoring
  INTERVAL: 1m
  INPUT: {"check_comprehensive": true}
END
</temporal>

When issues are detected:
<temporal>
WORKFLOW: AlertOncall
  INPUT: {"severity": "critical", "issue": "provider_down"}
  WAIT: true
END
</temporal>
```

### Provider Response with DSL

Providers can also emit DSL in their responses to request Temporal operations:

```
Analysis complete! Results:
- 1,250 items processed
- 3 anomalies detected
- Performance: 98.5% optimal

I've scheduled a detailed report generation for later:

<temporal>
WORKFLOW: GenerateDetailedReport
  INPUT: {"analysis_id": "a-123", "format": "pdf"}
  TIMEOUT: 5m
  WAIT: false
END
</temporal>

Please check back in 5 minutes for the full report.
```

## Implementation Notes

### Parsing

The DSL parser:
1. Extracts all `<temporal>...</temporal>` blocks using regex
2. Splits instructions by `END` keywords
3. Parses each instruction's header and parameters
4. Validates required fields
5. Returns both parsed instructions and cleaned text

### Execution

The DSL executor:
1. Validates each instruction before execution
2. Executes instructions in order
3. Returns results for each instruction
4. Collects execution time and status
5. Returns `TemporalDSLExecution` with full details

### Text Cleaning

When DSL is extracted, the cleaned text:
1. Removes all `<temporal>...</temporal>` blocks
2. Cleans up extra whitespace
3. Preserves all other content
4. Is suitable for sending to providers

This ensures providers never see the Temporal DSL notation.

## Usage in Loom

### From Agent Code

```go
// Parse and execute DSL from agent instructions
execution, err := temporalManager.ExecuteTemporalDSL(ctx, agentID, agentOutput)
if err != nil {
    log.Printf("DSL execution failed: %v", err)
}

// Get cleaned text (without DSL) for provider
cleanedText, _ := temporalManager.StripTemporalDSL(agentOutput)
// Send cleanedText to provider
```

### From Provider Response

```go
// Provider response may contain DSL
providerOutput := "Task complete. " + provider.Response

// Extract and execute DSL
execution, err := temporalManager.ExecuteTemporalDSL(ctx, agentID, providerOutput)

// Strip DSL before storing/displaying
cleaned, _ := temporalManager.StripTemporalDSL(providerOutput)
// Use cleaned text for display
```

## Duration Format

Durations support common formats:
- Go format: `30s`, `5m`, `2h`, `1m30s`
- Special values: `immediate`, `now` (0 duration)
- Default: `default` (5 minutes)
- Numbers: `60` (interpreted as seconds)

## Error Handling

Invalid DSL results in:
1. Parse errors for malformed syntax
2. Validation errors for missing required fields
3. Execution errors when Temporal operations fail

All errors are collected in the execution result without stopping other instructions.

## Version History

- v1.0 (2026-01-20): Initial DSL implementation
  - WORKFLOW, SCHEDULE, QUERY, SIGNAL, ACTIVITY, CANCEL, LIST instructions
  - Markdown-based syntax with `<temporal>` blocks
  - Integrated with TemporalManager
