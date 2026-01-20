# Project State Management

This document describes the project lifecycle management capabilities of AgentiCorp.

## Overview

AgentiCorp supports sophisticated project state management with three lifecycle states:
- **Open**: Active project with ongoing work
- **Closed**: Completed project with no remaining work
- **Reopened**: Previously closed project that has been reopened

## Key Features

- **Comments**: Add timestamped comments to track project decisions
- **Closure Workflow**: Close projects only when no open work remains
- **Agent Consensus**: If open work exists, requires agent agreement to close
- **Perpetual Projects**: Mark projects (like AgentiCorp itself) that never close

## API Endpoints

### Check Project State
```bash
GET /api/v1/projects/{id}/state
```

### Add Comment
```bash
POST /api/v1/projects/{id}/comments
{
  "author_id": "agent-id",
  "comment": "Your comment here"
}
```

### Close Project
```bash
POST /api/v1/projects/{id}/close
{
  "author_id": "agent-id",
  "comment": "Closure reason"
}
```

### Reopen Project
```bash
POST /api/v1/projects/{id}/reopen
{
  "author_id": "agent-id",
  "comment": "Reason for reopening"
}
```

## Configuration

Add projects to your `config.yaml`:

```yaml
projects:
  - id: agenticorp-self
    name: AgentiCorp Self-Improvement
    git_repo: https://github.com/jordanhubbard/agenticorp
    branch: main
    is_perpetual: true  # Never closes
    
  - id: example-project
    name: Example Project
    git_repo: https://github.com/example/repo
    branch: main
    is_perpetual: false  # Can be closed
```

## The AgentiCorp Persona

The AgentiCorp includes a special persona that works on improving the platform itself. This persona:
- Works on the perpetual `agenticorp-self` project
- Collaborates with UX, Engineering, PM, and Product personas
- Continuously improves the platform
- Never closes because there's always room for improvement

See `personas/agenticorp/` for the complete definition.
