# Git Security Model

This document defines the security model for git operations performed by Loom agents. Agents need git access to commit code, push branches, and create pull requests, but must operate within strict security constraints to protect repository integrity.

## Security Principles

### 1. Least Privilege

Agents receive minimum necessary git permissions:
- ✅ Read current repository state
- ✅ Commit to agent-specific branches
- ✅ Push to agent-specific branches
- ❌ Push to main/master/production branches
- ❌ Force push operations
- ❌ Delete branches
- ❌ Modify git configuration
- ❌ Access other repositories

### 2. Attribution & Audit

All git operations include:
- Agent ID in commit metadata
- Bead ID linking to work item
- Timestamp and execution context
- Audit trail in dedicated logs

### 3. Isolation

Each agent operates in isolated context:
- Cannot see other agents' branches (unless sharing is explicit)
- Cannot modify other agents' commits
- Cannot access git credentials directly
- Sandboxed git operations with resource limits

### 4. Reversibility

All operations must be reversible:
- Commits can be reverted
- Branches can be deleted
- Pull requests can be closed
- No irreversible destructive operations

## Branch Naming Convention

### Pattern

```
agent/{bead-id}/{description}
```

**Examples:**
```
agent/bead-abc-123/fix-authentication-bug
agent/bead-def-456/add-user-profile-feature
agent/bead-ghi-789/refactor-database-layer
```

### Rules

1. **Prefix**: All agent branches MUST start with `agent/`
2. **Bead ID**: Second segment is the bead ID (e.g., `bead-abc-123`)
3. **Description**: Slug-ified description (lowercase, hyphens, no special chars)
4. **Maximum length**: 72 characters total
5. **No spaces**: Use hyphens for word separation

### Branch Lifecycle

```
1. Agent creates branch: agent/{bead-id}/{description}
2. Agent commits changes to branch
3. Agent pushes branch to remote
4. Agent creates PR: agent/{bead-id}/{description} → main
5. Human reviews and merges PR
6. Branch deleted after merge (manual or automated)
```

### Branch Protection

**Allowed Operations:**
- Create new agent branches
- Push commits to own agent branches
- Delete own agent branches (after PR merged)

**Blocked Operations:**
- Push to main/master/production
- Force push to any branch
- Delete other agents' branches
- Rewrite history on pushed branches

## Commit Message Format

### Standard Format

```
<type>: <summary>

<body>

<footer>
```

**Example:**

```
feat: Add user authentication with JWT

Implements JWT-based authentication for API endpoints. Includes:
- Login/logout handlers
- Token validation middleware
- Refresh token support
- Session management

Bead: bead-abc-123
Agent: agent-worker-42
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### Required Elements

**1. Type** (required):
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring
- `test`: Test additions/changes
- `docs`: Documentation updates
- `style`: Formatting, missing semicolons, etc
- `perf`: Performance improvements
- `chore`: Maintenance tasks

**2. Summary** (required):
- First line, max 72 characters
- Imperative mood ("Add feature" not "Added feature")
- No period at end
- Concise description of change

**3. Body** (optional but recommended):
- Detailed explanation of changes
- Why the change was made
- What problem it solves
- Breaking changes noted

**4. Footer** (required for agents):
- `Bead: {bead-id}` - Links to work item
- `Agent: {agent-id}` - Identifies agent
- `Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>` - Attribution

### Commit Message Validation

All agent commits must pass validation:

```go
func ValidateCommitMessage(msg string) error {
    // Must have type and summary
    if !hasTypePrefix(msg) {
        return errors.New("missing type prefix")
    }

    // Must include bead reference
    if !strings.Contains(msg, "Bead:") {
        return errors.New("missing bead reference")
    }

    // Must include agent attribution
    if !strings.Contains(msg, "Agent:") && !strings.Contains(msg, "Co-Authored-By:") {
        return errors.New("missing agent attribution")
    }

    // Summary must be reasonable length
    firstLine := strings.Split(msg, "\n")[0]
    if len(firstLine) > 72 {
        return errors.New("summary too long (max 72 chars)")
    }

    return nil
}
```

## SSH Key Management

### Per-Project Keys

Each project has dedicated SSH keys for agent operations:

```
~/.loom/projects/{project-id}/
  ├── git_key           # Private SSH key
  ├── git_key.pub       # Public SSH key
  └── git_config        # Git configuration
```

### Key Generation

```bash
# Generate key for project
ssh-keygen -t ed25519 -C "loom-{project-id}@example.com" \
    -f ~/.loom/projects/{project-id}/git_key -N ""

# Add to GitHub/GitLab deploy keys (with write access)
cat ~/.loom/projects/{project-id}/git_key.pub
```

### Key Usage

```go
func (s *GitService) configureSSH(projectID string) error {
    keyPath := fmt.Sprintf("%s/.loom/projects/%s/git_key",
        os.Getenv("HOME"), projectID)

    // Configure git to use project-specific key
    os.Setenv("GIT_SSH_COMMAND",
        fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=accept-new", keyPath))

    return nil
}
```

### Key Security

1. **Private keys** never exposed to agents directly
2. **Environment-based** authentication (GIT_SSH_COMMAND)
3. **Read-only** key permissions (chmod 600)
4. **Project isolation** - each project has unique keys
5. **Key rotation** - keys can be regenerated without affecting agents

### Deploy Key Configuration

**GitHub:**
```
Settings → Deploy keys → Add deploy key
Title: Loom Agent Access ({project-id})
Key: <public key contents>
☑ Allow write access
```

**GitLab:**
```
Settings → Repository → Deploy Keys
Title: Loom Agent Access
Key: <public key contents>
☑ Write access allowed
```

## Git Operations Security

### Commit Operations

**Pre-Commit Checks:**
1. Validate commit message format
2. Check for secrets/credentials in code
3. Verify bead ID matches active work
4. Ensure agent has permission for files changed
5. Run linter/formatter if configured

**Commit Execution:**
```go
func (s *GitService) Commit(ctx context.Context, req CommitRequest) error {
    // Validate message
    if err := ValidateCommitMessage(req.Message); err != nil {
        return fmt.Errorf("invalid commit message: %w", err)
    }

    // Check for secrets
    if hasSecrets(req.Files) {
        return errors.New("commit contains potential secrets")
    }

    // Execute git commit
    cmd := exec.CommandContext(ctx, "git", "commit", "-m", req.Message)
    cmd.Dir = req.ProjectPath
    cmd.Env = s.buildEnv(req.ProjectID)

    return cmd.Run()
}
```

### Push Operations

**Pre-Push Checks:**
1. Verify branch name matches agent pattern
2. Check not pushing to protected branches
3. Validate remote repository matches project
4. Ensure no force push attempt
5. Check push size limits

**Push Execution:**
```go
func (s *GitService) Push(ctx context.Context, req PushRequest) error {
    // Validate branch name
    if !strings.HasPrefix(req.Branch, "agent/") {
        return errors.New("agents can only push to agent/* branches")
    }

    // Check not pushing to protected branch
    if isProtectedBranch(req.Branch) {
        return errors.New("cannot push to protected branch")
    }

    // Execute git push
    cmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", req.Branch)
    cmd.Dir = req.ProjectPath
    cmd.Env = s.buildEnv(req.ProjectID)

    return cmd.Run()
}
```

### Pull Request Operations

**PR Creation Requirements:**
1. Source branch must be agent branch
2. Target branch must be main/master (or specified)
3. PR title includes bead reference
4. PR body includes agent attribution
5. PR template fields populated

**PR Execution:**
```go
func (s *GitService) CreatePR(ctx context.Context, req PRRequest) error {
    // Validate source branch
    if !strings.HasPrefix(req.SourceBranch, "agent/") {
        return errors.New("PR source must be agent branch")
    }

    // Create PR using GitHub/GitLab API
    pr := &PullRequest{
        Title:  req.Title,
        Body:   req.Body,
        Source: req.SourceBranch,
        Target: req.TargetBranch,
    }

    return s.gitProvider.CreatePR(ctx, pr)
}
```

## Audit Trail

### Audit Log Format

All git operations logged to:
```
~/.loom/projects/{project-id}/git_audit.log
```

**Log Entry Format:**
```json
{
  "timestamp": "2026-02-04T10:30:45Z",
  "operation": "commit",
  "agent_id": "agent-worker-42",
  "bead_id": "bead-abc-123",
  "project_id": "proj-xyz-789",
  "branch": "agent/bead-abc-123/fix-authentication",
  "files_changed": ["internal/auth.go", "internal/auth_test.go"],
  "commit_sha": "a1b2c3d4e5f6",
  "success": true,
  "error": null,
  "duration_ms": 234
}
```

### Audit Retention

- **Production**: Retain all audit logs indefinitely
- **Development**: Retain for 90 days minimum
- **Testing**: Retain for duration of test execution

### Audit Analysis

Audit logs enable:
1. **Security review**: Identify suspicious patterns
2. **Debugging**: Trace agent actions leading to issues
3. **Compliance**: Demonstrate code provenance
4. **Analytics**: Measure agent productivity

## Protected Branches

### Protected Branch Configuration

**Main branches** (never allow direct agent push):
- `main`
- `master`
- `production`
- `release/*`
- `hotfix/*`

**Development branches** (controlled agent access):
- `develop` - PR required
- `staging` - PR required
- `feature/*` - owner can push

**Agent branches** (full agent access):
- `agent/*` - agents can create/push/delete

### Branch Protection Rules

```go
var ProtectedBranchPatterns = []string{
    "^main$",
    "^master$",
    "^production$",
    "^release/.*",
    "^hotfix/.*",
}

func isProtectedBranch(branch string) bool {
    for _, pattern := range ProtectedBranchPatterns {
        if matched, _ := regexp.MatchString(pattern, branch); matched {
            return true
        }
    }
    return false
}
```

### Override Mechanism

In emergency situations, humans can override protection:

```go
// Requires explicit human approval
func (s *GitService) EmergencyPush(ctx context.Context, req PushRequest, approval HumanApproval) error {
    if !approval.Verified {
        return errors.New("emergency push requires human approval")
    }

    // Log emergency override
    s.auditLog.LogEmergency(approval)

    // Execute push with --force if needed
    return s.forcePush(ctx, req)
}
```

## Secret Detection

### Pre-Commit Secret Scanning

Before each commit, scan for common secret patterns:

```go
var SecretPatterns = []regexp.Regexp{
    regexp.MustCompile(`(?i)api[_-]?key[_-]?=\s*['"][a-zA-Z0-9]{20,}['"]`),
    regexp.MustCompile(`(?i)secret[_-]?key[_-]?=\s*['"][a-zA-Z0-9]{20,}['"]`),
    regexp.MustCompile(`(?i)password[_-]?=\s*['"][^'"]{8,}['"]`),
    regexp.MustCompile(`(?i)token[_-]?=\s*['"][a-zA-Z0-9]{20,}['"]`),
    regexp.MustCompile(`(?i)aws[_-]?access[_-]?key[_-]?id`),
    regexp.MustCompile(`(?i)private[_-]?key[_-]?=`),
    regexp.MustCompile(`-----BEGIN (RSA|DSA|EC|OPENSSH) PRIVATE KEY-----`),
}

func hasSecrets(files []string) bool {
    for _, file := range files {
        content, _ := os.ReadFile(file)
        for _, pattern := range SecretPatterns {
            if pattern.Match(content) {
                return true
            }
        }
    }
    return false
}
```

### Secret Detection Bypass

For test fixtures or documentation:

```go
// Add .gitignore-secrets file with patterns
secrets/test_keys.txt
fixtures/*.key
docs/examples/*.secret
```

## Resource Limits

### Git Operation Limits

```go
const (
    MaxCommitSize     = 100 * 1024 * 1024  // 100 MB per commit
    MaxFilesPerCommit = 1000               // Max files in one commit
    MaxCommitRate     = 10                 // Max commits per minute per agent
    MaxPushRetries    = 3                  // Max push retry attempts
    PushTimeout       = 5 * time.Minute    // Max time for push operation
)
```

### Enforcement

```go
func (s *GitService) checkCommitSize(files []string) error {
    var totalSize int64
    for _, file := range files {
        info, _ := os.Stat(file)
        totalSize += info.Size()
    }

    if totalSize > MaxCommitSize {
        return fmt.Errorf("commit size %d exceeds limit %d",
            totalSize, MaxCommitSize)
    }

    return nil
}
```

## Error Handling

### Git Operation Errors

**Common Errors:**
1. **Merge conflicts**: Agent must resolve or escalate
2. **Authentication failures**: Check SSH keys, permissions
3. **Network issues**: Retry with exponential backoff
4. **Repository not found**: Verify remote configuration
5. **Permission denied**: Check deploy keys, branch protection

**Error Response:**
```go
type GitError struct {
    Operation string
    Error     error
    Retryable bool
    Context   map[string]interface{}
}

func (e *GitError) Error() string {
    return fmt.Sprintf("git %s failed: %v", e.Operation, e.Error)
}
```

### Conflict Resolution

When conflicts occur:

1. **Automatic resolution**: Try for simple conflicts
2. **Agent escalation**: Complex conflicts escalated to human
3. **Merge abort**: Agent can abort and retry
4. **Context preservation**: Save conflict state for review

```go
func (s *GitService) handleMergeConflict(ctx context.Context, conflict ConflictInfo) error {
    if conflict.IsSimple() {
        return s.autoResolve(ctx, conflict)
    }

    // Escalate to human
    return s.escalateConflict(ctx, conflict)
}
```

## Security Review Checklist

### Pre-Deployment Review

- [ ] SSH keys generated and configured per project
- [ ] Deploy keys added to Git providers with write access
- [ ] Branch protection rules configured on remote
- [ ] Audit logging enabled and tested
- [ ] Secret detection patterns validated
- [ ] Resource limits configured
- [ ] Error handling tested for all scenarios
- [ ] Agent attribution in commit messages verified
- [ ] Pull request templates configured
- [ ] Emergency override procedures documented

### Ongoing Security

- [ ] Regular audit log review (weekly)
- [ ] SSH key rotation (annually)
- [ ] Secret detection pattern updates (monthly)
- [ ] Branch cleanup (monthly)
- [ ] Failed operation analysis (daily)
- [ ] Resource usage monitoring (continuous)

## Related Documentation

- [Agent Actions Reference](AGENT_ACTIONS.md) - Complete action schema
- [Feedback Loops](FEEDBACK_LOOPS.md) - Code verification system
- [Conversation Architecture](CONVERSATION_ARCHITECTURE.md) - Multi-turn workflows

## Implementation Roadmap

### Phase 1: Core Git Operations (Current)

- Git service layer with commit/push/PR support
- Branch naming and commit message validation
- SSH key management
- Basic audit logging

### Phase 2: Advanced Security

- Secret detection integration (gitleaks, trufflehog)
- Automatic conflict resolution
- Advanced branch protection rules
- Real-time security monitoring

### Phase 3: Collaboration Features

- Agent-to-agent branch sharing
- Collaborative code review
- Automated merge conflict resolution
- Multi-agent pull request workflow

## License

See [LICENSE](../LICENSE) for details.
