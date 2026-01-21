# AgentiCorp Developer & User Guide

Welcome to AgentiCorp - the Agent Orchestration System. This guide helps you get started with developing agents, creating work items (beads), and using the system.

## Documentation

Start with the **[System Manual](MANUAL.md)** for a complete overview.

Then reference the specific guides below:

- **[User Guide](docs/USER_GUIDE.md)** - Getting started, UI usage, project registration
- **[Architecture](docs/ARCHITECTURE.md)** - System design, components, data flow
- **[Entities Reference](docs/ENTITIES_REFERENCE.md)** - All data structures (Agent, Bead, Provider, etc.)
- **[Temporal DSL Guide](docs/TEMPORAL_DSL.md)** - Workflow language for agents
- **[Worker System](docs/WORKER_SYSTEM.md)** - Agent execution model
- **[Beads Workflow](docs/BEADS_WORKFLOW.md)** - Creating and managing work items
- **[Project State Management](docs/PROJECT_STATE_MANAGEMENT.md)** - State persistence

## Quick Start

### 1. Start the System

```bash
docker compose up -d
```

Access at `http://localhost:8080`

### 2. Register a Provider

Navigate to **Providers** → **Register Provider** and enter:
- Endpoint: `http://your-llm:8000`
- Model: `nvidia/NVIDIA-Nemotron-3-Nano-30B-A3B-FP8`

Provider automatically checks health and transitions to `active`.

### 3. Create a Project

Via `config.yaml` or **Projects** → **Add Project**:
```yaml
id: myapp
name: My App
git_repo: https://github.com/user/repo
branch: main
beads_path: .beads
is_sticky: true
```

### 4. Create Beads

Create `.beads/beads/*.yaml` files in your git repo:
```yaml
id: bd-001-feature
type: feature
title: New Feature
description: Description of work
project_id: myapp
status: open
priority: 4
```

### 5. Assign Agents

**Project Viewer** → **Assign Agents** and select personas.

Agents automatically work when providers are available.

## Creating Custom Agents

### Step 1: Define Persona

Create `personas/default/my-role.md`:

```markdown
# My Agent Role

## Instructions
Define what this agent does, how it thinks, and what decisions it can make.

## Capabilities
- Capability 1
- Capability 2
- Can request workflows via Temporal DSL

## Using Temporal Workflows

Request long-running operations without providers:

<temporal>
WORKFLOW: AnalyzeData
  INPUT: {"source": "database"}
  TIMEOUT: 5m
  WAIT: true
END
</temporal>

Or schedule recurring tasks:

<temporal>
SCHEDULE: DailyReport
  INTERVAL: 24h
  INPUT: {"scope": "all"}
END
</temporal>
```

See [Temporal DSL Guide](docs/TEMPORAL_DSL.md) for workflow syntax.

### Step 2: Assign to Project

Via UI: **Projects** → Select Project → **Assign Agent**  
Or via API: `POST /api/v1/agents`

### Step 3: Monitor

**Project Viewer** shows agent status and current work.

## Creating Work Items (Beads)

Beads are YAML files defining work:

```yaml
# .beads/beads/bd-001-example.yaml

id: bd-001-example
type: feature
title: Implement Feature X
description: |
  Detailed description of the work.
  Can span multiple lines.

project_id: myapp
assigned_to: null  # null = auto-routed to available agent

status: open       # open, in_progress, blocked, done
priority: 4        # 1-5 scale

blocked_by:        # Block until these are done
  - bd-001-dependencies

blocks:            # These can't start until this is done
  - bd-002-downstream

parent_id: null    # For sub-tasks
children_ids: []   # Sub-tasks of this bead
```

### Dependencies

- **blocked_by**: Can't progress if these aren't done
- **blocks**: Prevents other beads from starting
- **parent/children**: Sub-task relationships

Circular dependencies are detected and reported at load time.

## Using Temporal DSL in Agents

The Temporal DSL lets agents request workflows without external providers:

### WORKFLOW - Execute Workflow

```markdown
<temporal>
WORKFLOW: ProcessData
  INPUT: {"dataset": "large"}
  TIMEOUT: 10m
  WAIT: true
  RETRY: 3
END
</temporal>
```

### SCHEDULE - Recurring Task

```markdown
<temporal>
SCHEDULE: HealthMonitoring
  INTERVAL: 1h
  INPUT: {"check_type": "comprehensive"}
END
</temporal>
```

### QUERY - Get Workflow State

```markdown
<temporal>
QUERY: wf-123
  TYPE: get_progress
END
</temporal>
```

### SIGNAL - Send Message to Workflow

```markdown
<temporal>
SIGNAL: approval-wf-456
  NAME: approve
  DATA: {"amount": 50000}
END
</temporal>
```

See [TEMPORAL_DSL.md](docs/TEMPORAL_DSL.md) for complete reference.

## Repository Rules

- All documentation goes in `docs/`.
- All internal AI planning files (generated `.md` files) go in `plans/`.
- All intermediate object files go in `obj/` and are never committed to git.
- All binaries go in `bin/` and are never committed to git.

## Building & Testing

```bash
# Build Docker images
make build

# Run tests
make test

# Format code
make fmt

# Clean reset (wipes database)
make distclean
```

## Troubleshooting

### Providers show "pending"
- Check provider endpoint is accessible
- Verify `/v1/models` returns models
- Check Docker network configuration (if containerized)

### Beads not loading
- Verify YAML syntax
- Check beads path exists in git
- Verify `project_id` matches

### Agents paused
- Check provider status is `active`
- Verify agent assigned to project
- Check for errors in logs

### Temporal issues
- Verify Temporal running: `docker ps | grep temporal`
- Check logs: `docker logs temporal`
- Restart: `docker compose restart temporal`

## Common Use Cases

### Run Periodic Tasks
Use `SCHEDULE` in Temporal DSL for recurring work without external providers.

### Approval Workflows
Create decision beads that route to CEO/manager agents for approval.

### Data Analysis
Use provider agents for heavy computation, Temporal workflows for orchestration.

### Multi-Step Workflows
Chain beads with dependencies, use signals to coordinate between agents.

## Key Entities

| Entity | Purpose |
|--------|---------|
| **Agent** | AI actor with role and instructions |
| **Bead** | Unit of work with status and dependencies |
| **Provider** | External LLM service |
| **Project** | Container for related work |
| **Persona** | Behavioral instructions for agents |
| **Workflow** | Temporal long-running process |
| **DSL** | Temporal workflow syntax for agents |

## Next Steps

1. Read [MANUAL.md](MANUAL.md) for complete system overview
2. Follow [User Guide](docs/USER_GUIDE.md) for operational workflows
3. Explore [Entities Reference](docs/ENTITIES_REFERENCE.md) for data structures
4. Study [Temporal DSL](docs/TEMPORAL_DSL.md) for workflow capabilities
5. Check [Architecture](docs/ARCHITECTURE.md) for design details

## Support

- **Issues**: Report via GitHub
- **Questions**: Check documentation and architecture guides
- **Contributions**: Follow repository rules above
