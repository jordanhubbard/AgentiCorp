# Git Service

Secure git operations for AgentiCorp agents with proper attribution, validation, and audit logging.

## Overview

The GitService provides safe git operations that enforce security constraints while maintaining complete auditability. All operations are logged, validated against security policies, and attributed to specific agents and beads.

## Quick Start

```go
import "github.com/jordanhubbard/agenticorp/internal/git"

// Create service
service, err := git.NewGitService("/path/to/project", "proj-123")
if err != nil {
    log.Fatal(err)
}

// Create agent branch
branchResult, err := service.CreateBranch(ctx, git.CreateBranchRequest{
    BeadID:      "bead-abc-123",
    Description: "fix authentication bug",
})

// Make changes... then commit
commitResult, err := service.Commit(ctx, git.CommitRequest{
    BeadID:  "bead-abc-123",
    AgentID: "agent-worker-42",
    Message: `fix: Resolve authentication timeout issue

Fixed JWT token expiration handling to properly refresh tokens
before they expire, preventing authentication timeouts.

Bead: bead-abc-123
Agent: agent-worker-42
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>`,
    AllowAll: true,
})

// Push to remote
pushResult, err := service.Push(ctx, git.PushRequest{
    BeadID:      "bead-abc-123",
    SetUpstream: true,
})
```

## Methods

### CreateBranch

Creates agent branch following `agent/{bead-id}/{description}` pattern.

**Validation:**
- Branch name must start with "agent/"
- Maximum 72 characters
- No whitespace
- Description is slugified (lowercase, hyphens)

**Example:**
```go
result, err := service.CreateBranch(ctx, CreateBranchRequest{
    BeadID:      "bead-abc-123",
    Description: "Add User Profile Feature",
})
// Creates: agent/bead-abc-123/add-user-profile-feature
```

### Commit

Creates commit with bead/agent attribution and secret scanning.

**Validation:**
- Commit message must include bead reference
- Must include agent attribution (Agent: or Co-Authored-By:)
- Summary line max 72 characters
- Scans for secrets (API keys, passwords, tokens, private keys)

**Secret Detection:**
- API keys (20+ char alphanumeric)
- Secret keys
- Passwords (8+ chars)
- Tokens
- AWS credentials
- Private SSH/RSA keys

**Example:**
```go
result, err := service.Commit(ctx, CommitRequest{
    BeadID:  "bead-abc-123",
    AgentID: "agent-worker-42",
    Message: commitMessage, // Must include Bead: and Agent:
    Files:   []string{"src/auth.go", "src/auth_test.go"},
})
```

### Push

Pushes commits to remote with branch protection.

**Security:**
- Can only push to agent/* branches
- Cannot push to protected branches (main, master, production, release/*, hotfix/*)
- Force push blocked
- Requires SSH key configuration

**Example:**
```go
result, err := service.Push(ctx, PushRequest{
    BeadID:      "bead-abc-123",
    SetUpstream: true, // Use -u flag for first push
})
```

### GetStatus / GetDiff

Retrieve git status or diff for code inspection.

```go
status, err := service.GetStatus(ctx)
diff, err := service.GetDiff(ctx, false) // unstaged changes
diffStaged, err := service.GetDiff(ctx, true) // staged changes
```

## Security Model

See [Git Security Model](../../docs/GIT_SECURITY_MODEL.md) for complete security design.

### Branch Naming

- **Pattern:** `agent/{bead-id}/{description}`
- **Protected:** main, master, production, release/*, hotfix/*
- **Allowed:** agent/* only

### Commit Format

```
<type>: <summary>

<body>

Bead: {bead-id}
Agent: {agent-id}
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### SSH Configuration

Keys stored per project:
```
~/.agenticorp/projects/{project-id}/
  ├── git_key       # Private SSH key
  ├── git_key.pub   # Public key
  └── git_audit.log # Audit trail
```

## Audit Logging

All operations logged to `~/.agenticorp/projects/{project-id}/git_audit.log`:

```json
{
  "timestamp": "2026-02-04T10:30:45Z",
  "operation": "commit",
  "bead_id": "bead-abc-123",
  "project_id": "proj-xyz-789",
  "ref": "a1b2c3d4e5f6",
  "success": true,
  "duration_ms": 234
}
```

## Error Handling

**Common Errors:**

```go
// Invalid branch name
err := service.CreateBranch(...) // "branch name must start with 'agent/'"

// Missing attribution
err := service.Commit(...) // "commit message must include agent attribution"

// Protected branch
err := service.Push(...) // "cannot push to protected branch: main"

// Secret detected
err := service.Commit(...) // "potential secret detected in src/config.go"

// SSH key missing
err := service.Push(...) // "SSH key not found: ~/.agenticorp/projects/proj-123/git_key"
```

## Testing

Run tests:
```bash
go test ./internal/git/...
```

Integration tests require test repository:
```bash
# Setup test repo
./scripts/setup_git_test_repo.sh

# Run integration tests
go test ./internal/git/... -tags=integration
```

## Related Documentation

- [Git Security Model](../../docs/GIT_SECURITY_MODEL.md) - Security design
- [Agent Actions](../../docs/AGENT_ACTIONS.md) - Action integration
- [Feedback Loops](../../docs/FEEDBACK_LOOPS.md) - Verification system

## Implementation Notes

**SSH Key Setup:**

```bash
# Generate key for project
cd ~/.agenticorp/projects/proj-123
ssh-keygen -t ed25519 -C "agenticorp-proj-123" -f git_key -N ""

# Add public key to GitHub/GitLab deploy keys with write access
cat git_key.pub
```

**Branch Cleanup:**

```bash
# Delete merged agent branches
git branch --merged | grep '^  agent/' | xargs -r git branch -d

# Delete remote agent branches
git branch -r --merged | grep 'origin/agent/' | sed 's/origin\///' | xargs -r git push origin --delete
```

## License

See [LICENSE](../../LICENSE) for details.
