# Arbiter

An autonomous agent orchestration system for coordinating AI coding agents working on software projects.

## Overview

Arbiter is a multi-agent coordination system that enables AI agents to work together on software projects with minimal human intervention. It uses [@steveyegge/beads](https://github.com/steveyegge/beads) for task tracking and provides sophisticated coordination mechanisms to prevent conflicts and enable autonomous decision-making.

## Key Features

- **Persona System**: Flexible agent personalities defined in markdown templates
- **Multi-Project Support**: Coordinate work across multiple repositories and branches
- **Decision Management**: Autonomous decision-making with escalation for critical choices
- **File Coordination**: Prevent merge conflicts through file locking
- **Beads Integration**: Git-backed issue tracking optimized for AI agents
- **Web UI**: Kanban board for monitoring work and claiming decision beads
- **OpenAPI Specification**: Well-documented REST API for programmatic access
- **HTTPS/PKI Support**: Production-ready security with TLS and PKI
- **100% Autonomy Goal**: Minimize human intervention while maintaining quality

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Arbiter Orchestration System                           │
│ ┌───────────────────────────────────────────────────┐   │
│ │ Web UI (Kanban Board, Persona Editor)            │   │
│ └───────────────────────────────────────────────────┘   │
│ ┌───────────────────────────────────────────────────┐   │
│ │ API Layer (HTTP/HTTPS, OpenAPI)                  │   │
│ └───────────────────────────────────────────────────┘   │
│ ┌───────────────┬─────────────┬───────────────────┐   │
│ │ Agent Manager │ File Locks  │ Decision System   │   │
│ └───────────────┴─────────────┴───────────────────┘   │
│ ┌───────────────────────────────────────────────────┐   │
│ │ Beads Integration (Task & Dependency Graph)       │   │
│ └───────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
         │                   │                   │
    ┌────────┐         ┌────────┐         ┌────────┐
    │Agent 1 │         │Agent 2 │         │Agent N │
    │(Code   │         │(Decision│         │(House- │
    │Reviewer)│         │Maker)   │         │keeping)│
    └────────┘         └────────┘         └────────┘
         │                   │                   │
    ┌─────────────────────────────────────────────────┐
    │ Project Git Repositories & Working Branches    │
    └─────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.24 or later
- [bd (beads)](https://github.com/steveyegge/beads) installed
- Git repositories for your projects

### Installation

```bash
# Clone the repository
git clone https://github.com/jordanhubbard/arbiter.git
cd arbiter

# Build
go build -o arbiter ./cmd/arbiter

# Initialize configuration
cp config.yaml.example config.yaml
vim config.yaml  # Edit as needed
```

### Configuration

Edit `config.yaml` to configure:

- Server ports (HTTP/HTTPS)
- TLS certificates (when ready)
- Beads integration settings
- Agent limits and behavior
- Project definitions
- Security settings

### Running

```bash
# Start the arbiter
./arbiter -config config.yaml

# Access web UI
open http://localhost:8080

# API is available at
# http://localhost:8080/api/v1/
```

## Persona System

Personas define how agents behave, what they can decide autonomously, and what requires escalation.

### Persona Structure

Each persona consists of two markdown files:

- `PERSONA.md`: Defines character, focus areas, autonomy level, and standards
- `AI_START_HERE.md`: Instructions for the agent on how to operate

### Example Personas

Three example personas are included:

1. **Code Reviewer**: Security-focused code review and bug finding
2. **Decision Maker**: Resolves decision points to unblock other agents
3. **Housekeeping Bot**: Maintains codebase health through continuous maintenance

### Creating Custom Personas

```bash
# Copy a template
cp -r personas/templates personas/my-agent

# Edit the files
vim personas/my-agent/PERSONA.md
vim personas/my-agent/AI_START_HERE.md

# Personas are loaded automatically from the personas directory
```

### Autonomy Levels

- **Full Autonomy**: Can make all non-P0 decisions independently
- **Semi-Autonomous**: Can handle routine decisions, escalates complex ones
- **Supervised**: Requires approval for all decisions

## Agent Coordination

### File Locking

Agents must request file access before editing:

1. Agent requests file lock via API: `POST /api/v1/file-locks`
2. Arbiter grants or denies based on current locks
3. Agent performs work
4. Agent releases lock: `DELETE /api/v1/file-locks/{project}/{path}`

This prevents merge conflicts when multiple agents work on the same branch.

### Decision Beads

When an agent encounters a decision point:

1. Agent files a decision bead: `POST /api/v1/beads` (type: decision)
2. Agent blocks its current work on the decision
3. Decision Maker agent or user claims and resolves the decision
4. Arbiter unblocks dependent work
5. Original agent continues with the decision

### Work Graph

Arbiter maintains a dependency graph of all beads:

- **Blocks**: This bead blocks another bead
- **Parent/Child**: Hierarchical relationships (epics, tasks, subtasks)
- **Related**: Informational relationships

Query the work graph: `GET /api/v1/work-graph?project_id=xxx`

## Beads Integration

Arbiter uses the [beads](https://github.com/steveyegge/beads) system for:

- Git-backed issue tracking
- Dependency management
- Agent-optimized JSON output
- Zero-conflict multi-branch workflows
- Semantic memory compaction

### Bead Types

- **Task**: Regular work items
- **Decision**: Decision points requiring resolution
- **Epic**: Parent items containing multiple tasks

### Bead Priorities

- **P0**: Critical, requires human intervention
- **P1**: High priority
- **P2**: Medium priority (default)
- **P3**: Low priority (housekeeping, nice-to-have)

## Web UI

The web UI provides:

- **Kanban Board**: Three columns (Open, In Progress, Closed)
- **Decision Queue**: Beads requiring decisions
- **Agent Status**: Live view of all agents and their status
- **Persona Editor**: Live editing of persona markdown files
- **Project Overview**: Status of all projects

Access at: `http://localhost:8080/`

## API

### Authentication

Include API key in header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/agents
```

### Key Endpoints

- `GET /api/v1/personas` - List personas
- `POST /api/v1/agents` - Spawn new agent
- `GET /api/v1/beads` - List work items
- `POST /api/v1/beads` - Create bead
- `POST /api/v1/beads/{id}/claim` - Claim a bead
- `GET /api/v1/decisions` - List decision beads
- `POST /api/v1/decisions/{id}/decide` - Make a decision
- `POST /api/v1/file-locks` - Request file lock
- `GET /api/v1/work-graph` - Get dependency graph

Full API documentation: See [api/openapi.yaml](api/openapi.yaml)

## Security

### HTTPS/TLS

Configure TLS certificates in `config.yaml`:

```yaml
server:
  enable_https: true
  https_port: 8443
  tls_cert_file: /path/to/cert.pem
  tls_key_file: /path/to/key.pem
```

### PKI Support

For mutual TLS authentication:

```yaml
security:
  pki_enabled: true
  ca_file: /path/to/ca.pem
  require_https: true
```

### API Keys

Manage API keys in config:

```yaml
security:
  enable_auth: true
  api_keys:
    - "key-1-here"
    - "key-2-here"
```

## Development

### Project Structure

```
arbiter/
├── api/                    # OpenAPI specifications
├── cmd/arbiter/           # Main application entry point
├── internal/              # Internal packages
│   ├── agent/            # Agent management
│   ├── arbiter/          # Core orchestration logic
│   ├── persona/          # Persona loading and editing
│   ├── project/          # Project management
│   ├── decision/         # Decision bead handling
│   ├── beads/            # Beads integration
│   ├── api/              # HTTP API handlers
│   └── web/              # Web UI
├── pkg/                   # Public packages
│   ├── models/           # Data models
│   └── config/           # Configuration
├── personas/              # Persona definitions
│   ├── templates/        # Template personas
│   └── examples/         # Example personas
└── web/static/           # Web UI assets
```

### Building

```bash
# Build for current platform
go build -o arbiter ./cmd/arbiter

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o arbiter-linux ./cmd/arbiter

# Build for macOS
GOOS=darwin GOARCH=arm64 go build -o arbiter-macos ./cmd/arbiter
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Use Cases

### 1. Autonomous Code Review

Deploy a code-reviewer agent with full autonomy to:
- Review all pull requests
- Fix obvious bugs automatically
- File decision beads for API changes
- Escalate security issues to P0

### 2. Continuous Maintenance

Deploy a housekeeping-bot to:
- Check for dependency updates daily
- Fix linting issues automatically
- Update documentation
- Remove dead code
- File decision beads for major upgrades

### 3. Feature Development

Deploy multiple agents working together:
- Feature developer agents on separate branches
- Code reviewer checking their work
- Decision maker resolving conflicts
- All coordinated by Arbiter to prevent merge conflicts

### 4. Human-in-the-Loop

Users can act as agents:
- Claim decision beads via web UI
- Override agent decisions
- Set priorities and direction
- Let agents handle the implementation

## Philosophy

**100% Autonomy is the Goal**

- Agents should work independently when possible
- Decision beads enable smart escalation
- Humans focus on high-level decisions
- Cost is not a concern - throughput is king
- Quality maintained through coordination and review

**Collaboration Over Competition**

- Agents coordinate, not compete
- File locking prevents conflicts
- Decision system enables consensus
- Knowledge sharing through bead context

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.

## Credits

- Built on [@steveyegge/beads](https://github.com/steveyegge/beads)
- Inspired by [ai-code-reviewer](https://github.com/jordanhubbard/ai-code-reviewer) persona system
- Designed for autonomous agent orchestration

## Support

- Issues: https://github.com/jordanhubbard/arbiter/issues
- Documentation: https://github.com/jordanhubbard/arbiter/wiki
