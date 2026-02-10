# Quick Start Guide

Get Loom running and your first agents working in under 10 minutes.

## Prerequisites

- Docker and Docker Compose
- A GPU with vLLM **or** an API key for a cloud LLM provider

## 1. Start Loom

```bash
git clone https://github.com/jordanhubbard/loom.git
cd loom
make start
```

This builds the container and starts the full stack (Loom, Temporal, PostgreSQL).
Wait about 30 seconds for everything to initialize, then open:

- **Loom UI**: http://localhost:8080
- **Temporal UI**: http://localhost:8088

## 2. Set Up a Provider

Loom needs at least one LLM provider to power its agents. You have two options:

### Option A: Run Your Own vLLM Server (GPU Required)

On any machine with an NVIDIA GPU (24GB+ VRAM recommended):

```bash
docker run -it --gpus all -p 8000:8000 \
    --ipc=host --ulimit memlock=-1 --ulimit stack=67108864 \
    -v ~/.cache/huggingface:/root/.cache/huggingface \
    nvcr.io/nvidia/vllm:25.12.post1-py3 \
    --model Qwen/Qwen2.5-Coder-32B-Instruct \
    --max-model-len 32768 \
    --tensor-parallel-size 1
```

Wait for the model to download and load (first run takes a while). Once you see
`Uvicorn running on http://0.0.0.0:8000`, register it with Loom:

```bash
curl -X POST http://localhost:8080/api/v1/providers \
    -H 'Content-Type: application/json' \
    -d '{
        "id": "my-gpu",
        "name": "My vLLM Server",
        "type": "openai",
        "endpoint": "http://<your-gpu-host>:8000/v1",
        "model": "Qwen/Qwen2.5-Coder-32B-Instruct"
    }'
```

Replace `<your-gpu-host>` with the hostname or IP of the machine running vLLM.
If it's the same machine as Loom, use `host.docker.internal` (macOS/Windows) or
the machine's LAN IP (Linux).

### Option B: Use a Cloud Provider (API Key)

Register any OpenAI-compatible endpoint:

```bash
curl -X POST http://localhost:8080/api/v1/providers \
    -H 'Content-Type: application/json' \
    -d '{
        "id": "cloud-llm",
        "name": "My Cloud Provider",
        "type": "openai",
        "endpoint": "https://api.example.com/v1",
        "model": "model-name",
        "api_key": "your-api-key-here"
    }'
```

### Verify the Provider

Within 30 seconds, the heartbeat will check your provider. Verify it's healthy:

```bash
curl -s http://localhost:8080/api/v1/providers | jq '.[].status'
```

You should see `"healthy"`. If you see `"failed"`, check the error:

```bash
curl -s http://localhost:8080/api/v1/providers | jq '.[].last_heartbeat_error'
```

## 3. Add a Project

Navigate to **Projects** in the Loom UI and click **Add Project**, or use the API:

```bash
curl -X POST http://localhost:8080/api/v1/projects \
    -H 'Content-Type: application/json' \
    -d '{
        "name": "My App",
        "git_repo": "git@github.com:youruser/yourrepo.git",
        "branch": "main",
        "beads_path": ".beads"
    }'
```

### Add the Deploy Key

Loom generates a unique SSH keypair for each project. Retrieve the public key:

```bash
curl -s http://localhost:8080/api/v1/projects/<project-id>/git-key | jq -r '.public_key'
```

Add this key as a **deploy key with write access** in your git hosting service:

- **GitHub**: Repository Settings > Deploy keys > Add deploy key
- **GitLab**: Settings > Repository > Deploy keys

Loom will clone the repository on the next dispatch cycle.

## 4. Use the CEO Dashboard

Open http://localhost:8080 and click **CEO Dashboard**. This is your command center.

The CEO Dashboard shows:
- Provider health and agent status
- Open beads across all projects
- The **Ask Loom** prompt for directing agents

### File Your First Beads

Use the **Ask Loom** prompt to give instructions, or create beads directly:

```bash
# Install the beads CLI (optional but recommended)
# See: https://github.com/steveyegge/beads

# Or create beads via the API:
curl -X POST http://localhost:8080/api/v1/beads \
    -H 'Content-Type: application/json' \
    -d '{
        "title": "Set up CI/CD pipeline",
        "description": "Create GitHub Actions workflow for build and test",
        "priority": 2,
        "type": "task",
        "project_id": "<project-id>"
    }'
```

### Priority Levels

| Priority | Meaning | Agent Behavior |
|----------|---------|----------------|
| P0 | Critical | Dispatched immediately to any available agent |
| P1 | High | Dispatched next after P0 work |
| P2 | Normal | Standard work queue (default) |
| P3 | Low | Backlog, picked up when nothing higher exists |

## 5. Watch Agents Work

Once you have a healthy provider and open beads, Loom's agents automatically:

1. **Claim** beads matching their expertise
2. **Read** your codebase to understand context
3. **Execute** actions (read files, search code, write changes, run tests)
4. **Iterate** through a multi-turn action loop (up to 15 turns per dispatch)
5. **Complete** or **escalate** when done

Monitor progress:

```bash
# See which agents are working
curl -s http://localhost:8080/api/v1/agents | jq '.[] | {name, status, current_bead}'

# Follow container logs
make logs
```

The agents work autonomously. Each agent has a persona (Engineering Manager,
Code Reviewer, DevOps Engineer, etc.) that determines what beads it picks up
and how it approaches the work.

## What's Next

- **[AGENTS.md](AGENTS.md)** - Full developer guide, API reference, and troubleshooting
- **[MANUAL.md](MANUAL.md)** - Complete system manual
- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System design and data flow

## Quick Reference

```bash
make start      # Start Loom (Docker)
make stop       # Stop Loom
make restart    # Rebuild and restart
make logs       # Follow container logs
make test       # Run tests locally
make help       # All available commands
```
