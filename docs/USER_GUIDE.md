# AgentiCorp User Guide

This guide helps new users run AgentiCorp, register projects, and work with agents and beads.

## Getting Started

### Prerequisites

- Docker 20.10+
- Docker Compose 1.29+
- Go 1.24+ (optional for local development)

### Start AgentiCorp

```bash
docker compose up -d
```

For local development with the full container stack, you can also use:

```bash
make run
```

Once running, AgentiCorp serves the API on `:8080` and the Temporal UI on `:8088`.

## Project Registration

Projects are registered in `config.yaml` under `projects:`. Required fields:

- `id`
- `name`
- `git_repo`
- `branch`
- `beads_path`

Optional fields:

- `is_perpetual` (never closes)
- `is_sticky` (auto-registered on startup)
- `context` (build/test/lint commands and other agent guidance)
- `git_auth_method` (e.g., `ssh`)
- `git_credential_id` (if using managed credentials)

Example:

```yaml
projects:
  - id: agenticorp
    name: AgentiCorp
    git_repo: git@github.com:jordanhubbard/agenticorp.git
    branch: main
    beads_path: .beads
    git_auth_method: ssh
    is_perpetual: true
    is_sticky: true
    context:
      build_command: "make build"
      test_command: "make test"
```

AgentiCorp loads beads from each project’s `beads_path` and uses them to build the work graph.

### Git Access (SSH Keys)

For SSH-based repos, fetch the per-project public key and add it as a **write-enabled deploy key**:

```bash
curl http://localhost:8080/api/v1/projects/<project-id>/git-key
```

Dispatch will pause until git access and the beads path are valid.

## Personas and Agents

Default personas live under `personas/default/`. The system persona(s) live under
`personas/agenticorp/`.

Agents are created from personas and attached to projects. The Project Viewer UI
shows agent assignments and bead progress in real time.

## Beads

Beads are JSONL work items stored in `.beads/issues.jsonl` for each project and
managed by the `bd` CLI. They drive the work graph and include metadata such as
priority, status, and dependencies.

Key fields:

- `id`, `type`, `title`, `description`
- `status`, `priority`, `project_id`
- `assigned_to`, `blocked_by`, `blocks`, `parent`, `children`

## Operational Workflow

1. Register projects in `config.yaml`.
2. Start AgentiCorp (docker compose or binary).
3. Confirm beads are loaded in the UI and API.
4. Assign agents to projects and monitor progress.
5. Use decisions/approvals for escalations (e.g., CEO workflow).

## Testing

AgentiCorp’s default `make test` runs the full Docker stack with Temporal:

```bash
make test
```

## Project Management UI

The Projects section and Project Viewer both support CRUD operations:

- **Add Project**: create a new project with repo, branch, and beads path.
- **Edit Project**: update fields like branch, beads path, perpetual/sticky flags.
- **Delete Project**: remove a project and its assignments.

Changes are applied immediately and reflected across the UI.

## CEO REPL

The CEO REPL lets you send high-priority questions directly to AgentiCorp. It uses
Temporal to route the request through the best available provider (quality and
latency weighted) with the AgentiCorp persona context.

1. Navigate to the **CEO REPL** section.
2. Enter your question and click **Send**.
3. Review the response and provider/model metadata.

## Configuration

AgentiCorp is configured via `config.yaml`. Key sections include:

### Dispatch Configuration

```yaml
dispatch:
  max_hops: 5  # Maximum times a bead can be redispatched before escalation
```

**Dispatch Hop Limit**: When a bead is dispatched (assigned to an agent) more than `max_hops` times without being closed, it is automatically escalated to P0 priority and a CEO decision bead is created. This prevents infinite redispatch loops and ensures stuck work gets human attention.

The system automatically enables redispatch for open and in-progress beads, allowing them to be picked up by idle agents. Dispatch history is tracked in the bead's context.

### Other Configuration Sections

- `agents`: Agent concurrency, personas, heartbeat intervals
- `beads`: Bead CLI path, auto-sync settings
- `security`: Authentication, CORS, PKI settings
- `temporal`: Workflow orchestration connection

See `config.yaml` for full configuration options.

## Troubleshooting

- If beads fail to load, confirm `.beads/issues.jsonl` exists and `bd list` works.
- If providers are missing, register them in the Providers UI and re-negotiate models.
- If providers show as disabled, check heartbeat errors and verify the provider endpoint.
- If no work is dispatched, check the Project Viewer for blocked beads, missing agents, or readiness gate failures.
- If beads are repeatedly dispatched without progress, check the dispatch hop count in the bead's context. The system will escalate to P0 after `max_hops` dispatches.
