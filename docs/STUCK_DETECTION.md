# Stuck Detection Heuristics

This document describes how Loom detects when an agent is truly stuck versus actively investigating a complex problem.

## Overview

The dispatcher uses intelligent heuristics to differentiate between:

- **Active Investigation**: Agent exploring different approaches, making progress
- **Stuck Loop**: Agent repeating the same actions without making meaningful progress

This prevents premature escalation of complex work while quickly identifying when agents need help.

## Stuck Indicators

The system flags an agent as "stuck" when **all** of the following conditions are met:

### 1. Repeated Action Pattern

The agent performs the same action 3+ times consecutively:

```
Example - Stuck:
- Read file foo.go
- Read file foo.go
- Read file foo.go
→ Same action repeated without variation
```

### 2. No Recent Progress

No progress indicators in the last 5 minutes:

- Last file modification: > 5 minutes ago
- Last test execution: > 5 minutes ago
- Last command execution: > 5 minutes ago
- Last new file access: > 5 minutes ago

### 3. Exceeded Dispatch Hop Limit

The bead has been dispatched >= `max_hops` times (default: 20)

**If ANY progress indicator is recent (< 5 minutes), the agent is NOT considered stuck**, even if actions repeat.

## Progress Indicators

The system tracks these signals that indicate an agent is making progress:

### File Activity
- **New files read**: Exploring different parts of the codebase
- **Files modified**: Making code changes
- **Different files accessed**: Broadening investigation scope

### Testing Activity
- **Tests executed**: Running validation
- **Test results change**: Observing different outcomes
- **New test failures**: Discovering edge cases

### Command Execution
- **Commands run**: Building, linting, debugging
- **Different commands**: Trying different approaches
- **Command output changes**: Getting different results

### Action Diversity
- **Varied action types**: Reading → editing → testing → analyzing
- **Different file paths**: Not stuck on single file
- **Progressive exploration**: Following logical investigation path

## Example Scenarios

### ✅ Active Investigation (Not Stuck)

**Scenario**: Complex bug requiring 25 dispatch cycles

```
Hop 1-5: Read error logs, trace execution
Hop 6-10: Read multiple source files, identify suspects
Hop 11-15: Modify code, add debug logging
Hop 16-20: Run tests, analyze results
Hop 21-25: Fix edge cases, validate solution

Progress indicators:
- 15 files read (varied)
- 8 files modified (making changes)
- 12 test runs (validation)
- 5 commands executed (building, debugging)
- Last progress: 30 seconds ago
```

**Outcome**: Allowed to continue past hop limit (making progress)

### ❌ Stuck Loop (Escalated)

**Scenario**: Agent stuck reading same file

```
Hop 1-5: Read authentication.go
Hop 6-10: Read authentication.go again
Hop 11-15: Read authentication.go again
Hop 16-20: Read authentication.go again

Progress indicators:
- 1 file read repeatedly
- 0 files modified
- 0 tests run
- 0 commands executed
- Last progress: 12 minutes ago
```

**Outcome**: Escalated to CEO at hop 20 (stuck in loop, no progress)

### ✅ Iterative Testing (Not Stuck)

**Scenario**: Agent debugging flaky test

```
Hop 1-3: Read test file
Hop 4-6: Modify test logic
Hop 7-9: Run test (fails)
Hop 10-12: Modify test logic
Hop 13-15: Run test (fails differently)
Hop 16-18: Modify test logic
Hop 19-21: Run test (passes)

Progress indicators:
- 3 files modified (iterating)
- 5 test runs (validation)
- Test results changing (progress)
- Last progress: 15 seconds ago
```

**Outcome**: Allowed to continue (making progress through iteration)

### ❌ Asking Same Question (Stuck)

**Scenario**: Agent unable to find required information

```
Hop 1-5: Search for API documentation
Hop 6-10: Search for API documentation (same query)
Hop 11-15: Search for API documentation (same query)
Hop 16-20: Search for API documentation (same query)

Progress indicators:
- Same search repeated
- 0 new files found
- 0 code changes
- Last progress: 8 minutes ago
```

**Outcome**: Escalated (stuck, needs human guidance)

## Implementation Details

### Action Tracking

Each action is recorded with:

```go
type ActionRecord struct {
    Timestamp   time.Time              // When action occurred
    AgentID     string                 // Which agent took action
    ActionType  string                 // "read_file", "edit_file", "run_tests", etc.
    ActionData  map[string]interface{} // Specific details (file path, command, etc.)
    ResultHash  string                 // Hash of action result
    ProgressKey string                 // Identifies action pattern for loop detection
}
```

### Progress Metrics

Tracked per bead:

```go
type ProgressMetrics struct {
    FilesRead        int       // Total files read
    FilesModified    int       // Total files modified
    TestsRun         int       // Total test executions
    CommandsExecuted int       // Total commands run
    LastProgress     time.Time // Most recent progress indicator
}
```

### Loop Detection Algorithm

```go
func IsStuckInLoop(bead *Bead) (bool, string) {
    history := getActionHistory(bead)

    // Need sufficient history
    if len(history) < repeatThreshold * 2 {
        return false, ""
    }

    // Check for recent progress
    if hasRecentProgress(bead) {
        return false, "" // Making progress, not stuck
    }

    // Look for repeated patterns
    pattern, count := findRepeatedPattern(history)
    if count >= repeatThreshold {
        return true, fmt.Sprintf("Repeated action pattern %d times without progress", count)
    }

    return false, ""
}
```

### Configuration

Default thresholds:

```go
const (
    repeatThreshold = 3          // Actions must repeat 3+ times
    progressWindow  = 5 * time.Minute  // Progress must be within 5 minutes
    historySize     = 50         // Keep last 50 actions per bead
)
```

## Integration with Dispatcher

The dispatcher checks for stuck loops at dispatch time:

```go
if dispatchCount >= maxHops {
    stuck, reason := loopDetector.IsStuckInLoop(bead)

    if !stuck {
        // Making progress - allow to continue
        log.Printf("Bead %s making progress: %s", bead.ID, getProgressSummary(bead))
        // Dispatch bead normally
    } else {
        // Stuck in loop - escalate
        log.Printf("Bead %s stuck in loop: %s", bead.ID, reason)
        escalateToCEO(bead, reason)
    }
}
```

## Escalation Context

When a bead is escalated as stuck, the following context is provided:

- **Loop detection reason**: What pattern was detected
- **Progress summary**: Full breakdown of activity metrics
- **Action history**: Last 50 actions taken
- **Dispatch count**: How many times bead was dispatched
- **Time since last progress**: Duration of inactivity

Example escalation:

```
Reason: dispatch_count=20 exceeded max_hops=20, stuck in loop: Repeated action pattern 7 times without progress: 4678e4c9b9f231d1

Progress Summary: Files read: 7, modified: 0, tests: 0, commands: 0 (last: 10m ago)

Loop Detection: Same file read repeatedly without modification or testing
```

## Benefits

### 1. Fewer False Escalations

Complex investigations can proceed beyond hop limits if making progress:

- **Before**: Escalated at 20 hops regardless of progress
- **After**: Escalated only when stuck, can continue to 30+ hops with progress

### 2. Faster Stuck Detection

Truly stuck agents are identified quickly:

- **Before**: Waited for full hop limit before escalation
- **After**: Detects stuck patterns early, escalates at hop limit

### 3. Rich Diagnostic Context

Escalations provide actionable information:

- What actions were repeated
- What progress indicators are missing
- How long agent has been stuck
- Full action history for analysis

### 4. Improved Agent Effectiveness

Agents can focus on productive work:

- No premature interruptions during complex investigations
- Clear signal when human help is needed
- Better understanding of what constitutes progress

## Monitoring

### View Progress for a Bead

```bash
# Check progress metrics
bd show bead-123 | grep progress

# View action history
bd show bead-123 | grep action_history
```

### Check Stuck Detection Stats

```bash
# Beads that continued past hop limit (making progress)
grep "making progress" logs/dispatcher.log | wc -l

# Beads escalated as stuck
grep "stuck in loop" logs/dispatcher.log | wc -l

# Progress summaries
grep "Progress:" logs/dispatcher.log
```

### Example Log Output

```
[Dispatcher] Bead ac-123 has 22 dispatches but is making progress, allowing to continue. Progress: Files read: 12, modified: 5, tests: 8, commands: 3 (last: 45s ago)

[Dispatcher] WARNING: Bead ac-456 is stuck in loop after 20 dispatches, escalating to CEO: Repeated action pattern 6 times without progress
```

## Best Practices

### For System Administrators

1. **Monitor escalation rate**: Should see fewer escalations with this system
2. **Review stuck beads**: Check if heuristics correctly identified stuck vs. investigating
3. **Adjust thresholds if needed**: Currently hardcoded, but can be made configurable
4. **Watch for edge cases**: Some investigation patterns may not fit standard heuristics

### For Agent Developers

1. **Track actions properly**: Ensure agents record their actions for loop detection
2. **Update progress metrics**: Signal progress through file changes, tests, commands
3. **Vary actions when stuck**: If truly stuck, try different approaches
4. **Request help when needed**: Some problems genuinely require human input

### For CEOs (Human Decision Makers)

1. **Review progress summary**: Check if agent made reasonable attempts
2. **Examine action history**: Look for patterns that indicate approach
3. **Provide specific guidance**: Point agent to missing information or different approach
4. **Update agent instructions**: If stuck due to lack of context, enhance persona

## Troubleshooting

### Bead Escalated But Was Making Progress

**Symptom**: Agent investigating productively but escalated as stuck

**Possible Causes**:
- Progress window too short (5 minutes)
- Actions not being recorded properly
- Progress metrics not updating

**Solution**:
- Check that agent is recording actions correctly
- Verify progress metrics are being updated
- Consider making progress window configurable

### Bead Not Escalating When Stuck

**Symptom**: Agent clearly stuck but not escalated

**Possible Causes**:
- Agent varying actions slightly (not detected as repeats)
- Some progress metric updating (e.g., reading different lines in same file)
- Below hop limit

**Solution**:
- Review action history to understand pattern
- Check if progress metrics are accurate
- Consider tightening repeat detection

### False Loop Detection

**Symptom**: Different actions flagged as repeats

**Possible Causes**:
- Progress key generation too coarse
- Not enough context in action data

**Solution**:
- Review progress key generation logic
- Include more specific data in ActionRecord
- Adjust hashing algorithm

## Future Enhancements

Potential improvements to stuck detection:

1. **Configurable Thresholds**: Make repeat count and progress window configurable per project
2. **Machine Learning**: Learn patterns of productive vs. stuck investigation over time
3. **Agent-Specific Profiles**: Different thresholds for different agent types
4. **Contextual Heuristics**: Domain-specific stuck detection (e.g., for debugging vs. feature development)
5. **Feedback Loop**: Use CEO decisions to refine heuristics automatically

## References

- [Dispatch Configuration](DISPATCH_CONFIG.md): Overall dispatch system configuration
- [Auto Bug Dispatch](auto-bug-dispatch.md): Automatic bug routing and dispatch
- [Loop Detector Source](../internal/dispatch/loop_detector.go): Implementation details
- [Dispatcher Integration](../internal/dispatch/dispatcher.go): How loop detection is used

## Version History

- **v1.2** (Epic 7, Task 2 & 3): Initial implementation of stuck detection heuristics
