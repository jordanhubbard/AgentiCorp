# Loom Developer & User Guide

Welcome to Loom - the Agent Orchestration System. This guide helps you get started with developing agents, creating work items (beads), and using the system.

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

Use the `bd` command to create beads:
```bash
bd create "New Feature" --type feature --priority 4 \
  --description "Description of work"
```

All beads are stored in `.beads/issues.jsonl` and managed via the `bd` CLI tool.

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

## Working with Beads - Two Contexts

Loom uses the **Beads** CLI tool for issue tracking in two contexts:

### 1. Loom's Own Beads (Meta-Work)

Located in **this repository** at `.beads/issues.jsonl`, these track work ON Loom itself:

- Features/bugs in Loom
- Documentation updates  
- Infrastructure work
- CI/CD improvements

**Managed via:**
- `bd` CLI tool exclusively
- JSONL format (not YAML files)
- Git-native sync with `bd sync`

### 2. Project Beads (Application Work)

When you register a project with Loom, it:

1. **Clones the project's git repository** into a work area
2. **Loads beads** from that project's `.beads/issues.jsonl` file
3. **Assigns agents** to work on those beads
4. **Commits changes** back to the project's repository

**Project registration** requires:
```yaml
id: myapp
name: My Application
git_repo: https://github.com/user/myapp
branch: main
beads_path: .beads
```

**Each project's beads live in its own repo**, not in Loom's repo.

### Git Repository Management

Loom runs in containers and proxies all git operations for managed projects:

- **Clone**: Fetches project repos into isolated work areas (`/app/src/<project-id>`)
- **Pull**: Keeps projects up-to-date with remote changes
- **Commit**: Saves agent work with descriptive commit messages
- **Push**: Publishes completed work back to origin
- **SSH/Credentials**: Managed securely per project

### Summary

- **Loom beads**: Live in `.beads/issues.jsonl` in THIS repo (Loom itself)
- **Project beads**: Live in `.beads/issues.jsonl` in EACH project's own repo
- **Beads CLI**: All bead operations use the `bd` command
- **Git proxying**: Loom manages git operations for all registered projects
- **Isolation**: Each project gets its own work area and git workspace

## Creating Agent Work Items (Beads)

Agent beads are work items that AI agents pick up and execute. Use the `bd` CLI to manage them:

```bash
# Create a new bead
bd create "Implement Feature X" \
  --type feature \
  --priority 2 \
  --description "Detailed description of the work"

# Update status
bd update bd-001 --status in_progress

# Add dependencies
bd update bd-002 --deps "blocked-by:bd-001"

# Close when done
bd update bd-001 --status closed

# View all beads
bd list

# View open beads
bd list open
```

**Priority Levels**:
- P0 (0): Critical, blocking work
- P1 (1): High priority
- P2 (2): Normal priority (default)
- P3 (3): Low priority, nice-to-have

**Prefer using the Loom API or Web UI to create/update beads for agents.**

### Dependencies

Beads support dependency relationships:

```bash
# Create a dependency (bd-002 is blocked by bd-001)
bd update bd-002 --deps "blocked-by:bd-001"

# Create a blocks relationship
bd update bd-001 --deps "blocks:bd-002"

# Parent-child relationships
bd create "Sub-task" --parent bd-050
```

Circular dependencies are automatically detected by the `bd` CLI.

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

## Operating Procedures

### Makefile Targets

| Target | Description | When to Use |
|--------|-------------|-------------|
| `make start` | Build and start loom natively (no Docker), PID in `.loom.pid`, logs to `loom.log`, serves on `http://localhost:8081` | Local development |
| `make stop` | Stop the locally-running loom process | End of dev session |
| `make build` | Lint YAML + build Docker containers | Before deploying |
| `make build-all` | Cross-compile for linux/darwin/windows | Release builds |
| `make run` | Build + `docker compose up --build` | Full Docker dev |
| `make restart` | `docker compose down` then build + up | Reset Docker state |
| `make test` | Run unit tests in Docker | CI or pre-commit |
| `make test-api` | Run post-flight API validation | After deployment |
| `make coverage` | Run tests with coverage HTML report | Code review |
| `make fmt` | `go fmt ./...` | Before committing |
| `make vet` | `go vet ./...` | Before committing |
| `make lint` | fmt + vet + lint-yaml + lint-docs | Full lint pass |
| `make deps` | `go mod download && go mod tidy` | After dependency changes |
| `make clean` | Remove binaries and coverage files | Quick cleanup |
| `make distclean` | Stop containers, remove images, prune Docker, clear Go cache | Full reset |
| `make release` | Create patch release (x.y.Z) | Versioned release |
| `make release-minor` | Create minor release (x.Y.0) | Feature release |
| `make release-major` | Create major release (X.0.0) | Breaking change release |

**Build failures auto-file P0 beads** in `.beads/beads/` with full error output.

### Local Development Workflow

**Prerequisites:**
```bash
# Start Temporal infrastructure (required)
docker compose up -d temporal-postgresql temporal temporal-ui

# Start loom
make start

# Verify
curl -s http://localhost:8081/health | jq .status
```

**Logs:** `tail -f loom.log`
**PID file:** `.loom.pid`

**Stop:** `make stop`

### Telemetry & Observability APIs

All endpoints are at `http://localhost:8081` (native) or `http://localhost:8080` (Docker).

#### Analytics Endpoints

```bash
# Request logs — filter by provider, time range
curl 'http://localhost:8081/api/v1/analytics/logs?provider_id=prov-1&start_time=2026-01-01T00:00:00Z'

# Aggregated stats — total requests, tokens, costs
curl 'http://localhost:8081/api/v1/analytics/stats'

# Cost breakdown by provider and user
curl 'http://localhost:8081/api/v1/analytics/costs'

# Export logs as CSV
curl 'http://localhost:8081/api/v1/analytics/export?format=csv'

# Export stats as JSON
curl 'http://localhost:8081/api/v1/analytics/export-stats?format=json'

# Batching optimization recommendations
curl 'http://localhost:8081/api/v1/analytics/batching?max_recommendations=5&window_minutes=60'
```

**Analytics log fields:** timestamp, user, method, path, provider, model, tokens (input/output), latency_ms, status, cost_usd

#### Event Endpoints

```bash
# Recent events (with optional filters)
curl 'http://localhost:8081/api/v1/events?project_id=myapp&type=bead.status_change&limit=50'

# Event bus statistics
curl 'http://localhost:8081/api/v1/events/stats'

# Live SSE event stream (real-time)
curl -N 'http://localhost:8081/api/v1/events/stream?project_id=myapp'
```

#### Health & Readiness

```bash
# Detailed health with dependency checks
curl 'http://localhost:8081/health'

# Kubernetes liveness probe
curl 'http://localhost:8081/health/live'

# Kubernetes readiness probe (checks DB, providers)
curl 'http://localhost:8081/health/ready'

# Prometheus metrics
curl 'http://localhost:8081/metrics'
```

### Monitoring & Debugging

**Check agent status:**
```bash
curl -s http://localhost:8081/api/v1/agents | jq '.[] | {name, status, provider_id, current_bead}'
```

**Check provider health:**
```bash
curl -s http://localhost:8081/api/v1/providers | jq '.[] | {id, status, model}'
```
Provider status must be `"healthy"` (set by heartbeat workflow), not `"active"`.

**Check dispatch status:**
```bash
curl -s http://localhost:8081/api/v1/dispatch/status | jq .
```

**Loop detection:** When Ralph detects a stuck agent loop (dispatch_count exceeds max_hops with no progress), it auto-blocks the bead and records:
- `ralph_blocked_reason` — why it was blocked
- `loop_detection_reason` — specific loop pattern detected
- `progress_summary` — files read/modified, tests run, commands executed
- `revert_status` — recommended commit revert range

### Git Workflow for Agents

Agents use a **branch-per-bead** strategy with safety guardrails:

**Branch naming:** `agent/{bead-id}/{description-slug}`
- Configurable prefix (default: `agent/`)
- Protected branches (`main`, `master`, `production`, `release/*`, `hotfix/*`) cannot be pushed to or deleted

**Available git actions for agents:**

| Action | Purpose |
|--------|---------|
| `git_status` | Check working tree state |
| `git_diff` | View unstaged changes |
| `git_commit` | Commit with bead/agent attribution |
| `git_push` | Push to remote (agent branches only) |
| `create_pr` | Create pull request via `gh` CLI |
| `git_merge` | Merge branch with `--no-ff` (default) |
| `git_revert` | Revert specific commit(s) |
| `git_branch_delete` | Delete local + optional remote branch |
| `git_checkout` | Switch branches (requires clean tree) |
| `git_log` | View structured commit history |
| `git_fetch` | Fetch remote refs with prune |
| `git_list_branches` | List all local and remote branches |
| `git_diff_branches` | Cross-branch diff (`branch1...branch2`) |
| `git_bead_commits` | Find all commits for a bead ID |

**Commit metadata trailers** (added automatically):
```
feat: Implement token caching

Bead: loom-abc123
Agent: agent-456-Engineering-Manager
Project: myapp
Dispatch: 5
```

**Merge practices:**
- Always use `--no-ff` for audit trail (merge commits show what was merged)
- Create PR for review before merging to main
- Delete feature branches after merge

**When to revert:**
- Build fails after agent commits
- Agent is stuck in a loop (Ralph auto-blocks and recommends revert range)
- Tests regress after changes

### Beads Workflow (bd CLI)

```bash
# Find available work
bd ready

# Claim and start work
bd update <id> --status=in_progress

# Create related issues
bd create --title="Fix X" --type=bug --priority=2

# Add dependencies
bd dep add <issue> <depends-on>

# Close completed work
bd close <id1> <id2> ...

# Sync with remote
bd sync

# Project health check
bd stats
bd doctor
```

## Building & Testing

```bash
# Build Docker images
make build

# Run tests
make test

# Run specific package tests
go test ./internal/git/... ./internal/actions/... ./internal/dispatch/...

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
- Provider status is set to `"healthy"` by heartbeat workflow — check Temporal is running

### Beads not loading
- Verify YAML syntax
- Check beads path exists in git
- Verify `project_id` matches

### Agents paused
- Check provider status is `healthy` (not just `active`)
- Verify agent assigned to project
- Agent lifecycle: `CreateAgent()` -> `"paused"`, `SpawnAgentWorker()` -> `"idle"`
- Check for errors in `loom.log`

### Temporal issues
- Verify Temporal running: `docker ps | grep temporal`
- Check logs: `docker logs temporal`
- Restart: `docker compose restart temporal`
- Prerequisites: `docker compose up -d temporal-postgresql temporal temporal-ui`

### Dispatch not working
- Readiness mode `"block"` + failed `git ls-remote` = no beads dispatched
- Check readiness mode in config (default: `"warn"`)
- Verify idle agents exist: `curl http://localhost:8081/api/v1/agents | jq '.[] | select(.status=="idle")'`

### Auth errors in development
- `AUTH_ENABLED=false` skips auth for all endpoints
- `loadUsers()`/`loadAPIKeys()` have early-return guards when auth is disabled

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

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
