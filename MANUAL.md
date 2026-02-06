# Loom System Manual

Complete user and developer manual for Loom - the Agent Orchestration System.

## Quick Links

- **User Guide**: [docs/USER_GUIDE.md](docs/USER_GUIDE.md) - Getting started, UI usage
- **Architecture**: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - System design and components
- **Entities Reference**: [docs/ENTITIES_REFERENCE.md](docs/ENTITIES_REFERENCE.md) - All data structures
- **Temporal DSL**: [docs/TEMPORAL_DSL.md](docs/TEMPORAL_DSL.md) - Workflow language syntax
- **Worker System**: [docs/WORKER_SYSTEM.md](docs/WORKER_SYSTEM.md) - Agent execution model
- **Beads Workflow**: [docs/BEADS_WORKFLOW.md](docs/BEADS_WORKFLOW.md) - Work item definitions
- **Project State Management**: [docs/PROJECT_STATE_MANAGEMENT.md](docs/PROJECT_STATE_MANAGEMENT.md) - State persistence

## What is Loom?

Loom is a comprehensive **agent orchestration system** that:

1. **Coordinates multiple AI agents** with different roles (personas)
2. **Manages distributed work** through a powerful workflow engine
3. **Integrates with LLM providers** (local or cloud-based)
4. **Uses Temporal** for reliable, durable workflow execution
5. **Persists all state** to survive restarts
6. **Provides real-time monitoring** via web UI

## Core Concepts

### Agents
- Autonomous AI entities with specific roles (CEO, CFO, Engineer, etc.)
- Created from **personas** (behavior definitions)
- Have a **status**: idle, working, paused, complete
- Are **assigned to projects** and process work items
- **Paused** until a provider becomes available

### Beads
- Discrete units of work (features, bugs, tests, decisions)
- Defined as YAML files with metadata
- Have **status**: open, in_progress, blocked, done
- Support **dependencies**: can block other beads, be blocked by others
- Can be **assigned** to specific agents or auto-routed

### Providers
- External LLM services (vLLM, Ollama, OpenAI, etc.)
- Have **endpoints** (network addresses)
- Serve **models** (specific LLMs)
- Transition from **pending** → **active** → **error**
- **Health checked** every 30 seconds

### Projects
- Containers for related work
- Map to **git repositories** with branches
- Have **beads** stored in git
- Have **agents assigned** for work
- Can be **perpetual** (never finish) or normal

### Temporal Workflows
- Reliable, durable **long-running processes**
- Support **retry**, **pause**, **resume**, **signal**, **query**
- Survive crashes and restarts
- Can be **scheduled** for recurring execution
- Can be embedded in agent instructions via **DSL**

## Getting Started

### 1. Start the System

```bash
# Start full Docker stack
docker compose up -d

# Or run locally with external Temporal
make run
```

### 2. Register a Provider

**Via UI**:
1. Navigate to **Providers** section
2. Click **Register Provider**
3. Enter endpoint (e.g., `http://localhost:8000`)
4. Select model (e.g., `nvidia/Nemotron`)
5. Submit - provider will be checked immediately

**Via Config**:
```yaml
providers:
  - id: local-llm
    endpoint: http://localhost:8000
    model: nvidia/Nemotron
```

**Via API**:
```bash
curl -X POST http://localhost:8080/api/v1/providers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-provider",
    "endpoint": "http://provider.local:8000",
    "model": "mistral/Mistral"
  }'
```

### 3. Register a Project

**Via config.yaml**:
```yaml
projects:
  - id: myapp
    name: My App
    git_repo: https://github.com/user/myapp
    branch: main
    beads_path: .beads
    is_sticky: true
```

**Via UI**:
1. Navigate to **Projects**
2. Click **Add Project**
3. Fill in repository details
4. Submit

### 4. Create Beads (Work Items)

Create `.beads/beads/` directory in your git repo and add YAML files:

```yaml
id: bd-001-feature-auth
type: feature
title: Implement Authentication
description: Add user authentication system
project_id: myapp
status: open
priority: 4
```

Push to git, and beads load automatically.

### 5. Assign Agents

**Via UI**:
1. Go to **Project Viewer**
2. Click **+ Add Agent**
3. Select persona and project
4. Agent appears and waits for provider

**Via API**:
```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Engineer 1",
    "project_id": "myapp",
    "persona_id": "engineer"
  }'
```

### 6. Monitor Progress

**Kanban Board**: See beads organized by status  
**Project Viewer**: See agents and work assignments  
**System Status**: Overall health overview  

## Using Temporal DSL

Embed workflow requests in agent instructions without needing external providers:

### In Agent Personas

**CFO Persona** (`personas/default/cfo.md`):
```markdown
# Chief Financial Officer

Before approving budgets over $50K:

<temporal>
WORKFLOW: ReviewBudgetHistory
  INPUT: {"threshold": 50000}
  TIMEOUT: 2m
  WAIT: true
END
</temporal>

When approving:

<temporal>
WORKFLOW: LogBudgetApproval
  INPUT: {"amount": 100000}
  WAIT: false
END
</temporal>
```

### In Provider Responses

A provider can include DSL in its response:

```
Analysis complete with following findings:
- 5 issues detected
- 3 opportunities for optimization

<temporal>
WORKFLOW: GenerateFullReport
  INPUT: {"analysis_id": "a-123"}
  TIMEOUT: 5m
  WAIT: false
END
</temporal>
```

### DSL Syntax

**Schedule recurring task**:
```markdown
<temporal>
SCHEDULE: DailyHealthCheck
  INTERVAL: 24h
  INPUT: {"comprehensive": true}
END
</temporal>
```

**Query workflow status**:
```markdown
<temporal>
QUERY: wf-123
  TYPE: get_progress
END
</temporal>
```

**Send signal to workflow**:
```markdown
<temporal>
SIGNAL: approval-wf-xyz
  NAME: approve
  DATA: {"amount": 50000}
END
</temporal>
```

See [docs/TEMPORAL_DSL.md](docs/TEMPORAL_DSL.md) for complete syntax.

## Key Features

### 1. Real-Time UI
- Live status updates
- Drag-and-drop task management
- Agent/provider/bead monitoring
- Decision approval workflows

### 2. Dependency Management
- Beads block other beads
- Automatic deadlock detection
- Ready-bead identification
- Smart work ordering

### 3. Provider Integration
- Multiple LLM providers
- Automatic health checking
- Model negotiation
- API key management (encrypted)

### 4. Workflow Orchestration
- Temporal-powered reliability
- Automatic retry/resume
- Signal/Query capabilities
- Scheduled execution
- Embedded DSL support

### 5. State Persistence
- SQLite database (survives restarts)
- Temporal workflow state
- YAML bead definitions (in git)
- Full audit trail

### 6. Scalability
- Horizontal scaling via Temporal
- Multiple agents per project
- Multiple providers
- Perpetual projects (never end)

## Common Workflows

### Adding a New Feature

1. Create bead in `.beads/beads/`
2. Set status to `open`, priority appropriately
3. Beads auto-load on commit
4. Dispatcher finds ready beads
5. Engineer agent processes
6. Status updates to `done`
7. Any dependent beads become ready

### Budget Approval Process

1. Engineer creates bead requesting budget
2. System assigns to CFO agent
3. CFO reviews (via Temporal workflow)
4. CFO approves/denies via decision UI
5. Decision signal sent to workflow
6. Bead updates based on decision
7. Next steps triggered

### Monitoring Provider Health

1. Provider registered (auto health check)
2. Provider becomes `active`
3. Agents resume work
4. Periodic heartbeat checks (30s)
5. On failure, provider → `error`
6. Agents pause until provider recovers

### Running Perpetual Tasks

1. Create project with `is_perpetual: true`
2. Define recurring beads or use Temporal DSL
3. System never stops processing
4. New beads can be added anytime
5. Always checking for ready work

## Command Reference

### Docker Compose

```bash
# Start full stack
docker compose up -d

# View logs
docker compose logs -f loom

# Stop
docker compose down

# Clean reset (wipes all state)
docker compose down -v
```

### Make Commands

```bash
# Build Docker images
make build

# Run locally with external Temporal
make run

# Run tests
make test

# Clean build artifacts
make clean

# Full reset (wipes database)
make distclean

# Format/lint code
make fmt
```

### API Endpoints

```bash
# Projects
GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/:id
PUT    /api/v1/projects/:id
DELETE /api/v1/projects/:id

# Providers
GET    /api/v1/providers
POST   /api/v1/providers
GET    /api/v1/providers/:id
PUT    /api/v1/providers/:id
DELETE /api/v1/providers/:id
POST   /api/v1/providers/:id/negotiate  # Model negotiation

# Agents
GET    /api/v1/agents
POST   /api/v1/agents
GET    /api/v1/agents/:id
PUT    /api/v1/agents/:id

# Beads
GET    /api/v1/beads
GET    /api/v1/beads/:id
PUT    /api/v1/beads/:id

# Decisions
GET    /api/v1/decisions
POST   /api/v1/decisions
PUT    /api/v1/decisions/:id

# System
GET    /api/v1/system/status

# CEO REPL
POST   /api/v1/repl/query
```

## Configuration

### config.yaml

```yaml
# Server
api:
  port: 8080
  host: 0.0.0.0

# Database
database:
  path: ./loom.db

# Temporal
temporal:
  host: temporal:7233
  namespace: loom-default
  task_queue: loom-tasks
  enabled: true

# Projects
projects:
  - id: myapp
    name: My App
    git_repo: https://github.com/user/repo
    branch: main
    beads_path: .beads
    is_sticky: true
    is_perpetual: false
```

## Troubleshooting

### Providers show "pending"

1. Check provider endpoint is reachable
2. Verify `/v1/models` endpoint returns models
3. Check provider logs for errors
4. Try re-registering provider
5. Check Docker network configuration (if containerized)

### Beads not loading

1. Verify beads path exists in git repo
2. Check YAML syntax with `make lint-yaml`
3. Verify `project_id` matches registered project
4. Look for loading errors in logs: `docker logs loom`

### Agents remain paused

1. Check if provider is registered
2. Verify provider status is `active` (not `pending`)
3. Check agent assignment via UI
4. Look for errors: `docker logs loom | grep -i error`

### Temporal connection issues

1. Verify Temporal running: `docker ps | grep temporal`
2. Check connection: `curl -s http://localhost:7233/`
3. Review logs: `docker logs temporal`
4. Restart: `docker compose restart temporal`

### Database corruption

Clean reset:
```bash
# Stop and remove all containers
docker compose down -v

# Or locally:
make distclean

# Restart
docker compose up -d
```

## Performance Tuning

### For Large Projects

1. Increase Temporal worker threads
2. Use perpetual=false for bounded projects
3. Break large beads into smaller sub-beads
4. Assign multiple agents to project

### For Many Providers

1. Reduce heartbeat interval (caution: more load)
2. Use provider groups (feature: future)
3. Implement provider pooling (feature: future)

### For High Throughput

1. Enable database indexing on frequently queried fields
2. Use perpetual projects
3. Increase dispatcher frequency (default 5s)
4. Add more Temporal workers

## Development

### Adding Custom Personas

1. Create `personas/default/yourpersona.md`
2. Define instructions, capabilities, constraints
3. Optionally include `<temporal>` DSL blocks
4. Reference in agent assignments

### Adding Custom Workflows

1. Create workflow in `internal/temporal/workflows/`
2. Register in `internal/temporal/manager.go`
3. Call from activities or DSL
4. Test with existing test infrastructure

### Using Temporal DSL Programmatically

```go
// Parse DSL
instructions, cleanedText, err := temporalManager.ParseTemporalInstructions(agentOutput)

// Execute DSL
execution, err := temporalManager.ExecuteTemporalDSL(ctx, agentID, agentOutput)

// Strip DSL before sending to provider
cleaned, _ := temporalManager.StripTemporalDSL(agentOutput)
```

## Support & Contributing

- **Issues**: Report bugs via GitHub issues
- **Documentation**: Update docs/ files for new features
- **Tests**: Add tests for new functionality
- **Code Style**: Run `make fmt` before committing

## Glossary

| Term | Definition |
|------|-----------|
| **Agent** | Autonomous AI actor with specific role |
| **Bead** | Unit of work (task, story, decision) |
| **Persona** | Behavioral instructions and guidelines |
| **Provider** | External LLM service |
| **Project** | Container for related work |
| **Workflow** | Temporal long-running process |
| **DSL** | Domain-Specific Language (Temporal DSL) |
| **Signal** | Message sent to running workflow |
| **Query** | State request from running workflow |
| **Heartbeat** | Periodic health check |

## Version Information

- **Loom**: Latest
- **Temporal**: 1.22.4+
- **PostgreSQL**: 15+
- **Go**: 1.24+

---

**Last Updated**: 2026-01-20  
**Documentation Version**: 1.0
