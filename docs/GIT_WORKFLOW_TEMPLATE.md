## Git Workflow

You have access to git operations for version control. Use these actions to commit, push, and create pull requests for your work.

### When to Use Git Actions

**Commit your changes when:**
- You've completed a logical unit of work (feature, bugfix, refactoring)
- All tests pass successfully
- Linter shows no errors
- Build completes without issues
- You're about to hand off work to another agent

**Push to remote when:**
- You've made one or more commits
- You want to back up your work
- You're ready for code review
- Another agent needs your changes

**Create a pull request when:**
- Your feature/fix is complete and tested
- You want code review from other agents or humans
- You're ready to merge work into the main branch

### Git Action Examples

**1. Commit Changes:**
```json
{
  "actions": [{
    "type": "git_commit",
    "commit_message": "feat: Add user authentication\n\nImplements JWT-based authentication with refresh tokens.\nIncludes unit tests and integration tests.\n\nBead: bead-abc-123\nAgent: agent-worker-42\nCo-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>",
    "files": ["src/auth.go", "src/auth_test.go"]
  }]
}
```

**2. Push to Remote:**
```json
{
  "actions": [{
    "type": "git_push",
    "branch": "agent/bead-abc-123/add-auth",
    "set_upstream": true
  }]
}
```

**3. Create Pull Request:**
```json
{
  "actions": [{
    "type": "create_pr",
    "pr_title": "Add user authentication feature",
    "pr_body": "## Summary\n- Implements JWT authentication\n- Adds refresh token support\n- Includes comprehensive tests\n\n## Test Plan\n- Unit tests: auth_test.go\n- Integration tests: auth_integration_test.go\n\nBead: bead-abc-123",
    "pr_base": "main",
    "pr_reviewers": ["code-reviewer"]
  }]
}
```

### Commit Message Format

Follow conventional commits format:

```
<type>: <summary>

<detailed description>

Bead: <bead-id>
Agent: <agent-id>
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring
- `test`: Adding or updating tests
- `docs`: Documentation changes
- `chore`: Maintenance tasks

**Example:**
```
feat: Implement user profile management

Adds CRUD operations for user profiles with validation.
Includes API endpoints and database migrations.

Bead: bead-xyz-789
Agent: agent-engineer-5
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### Git Best Practices

1. **Commit After Success**: Only commit when tests pass and builds succeed
2. **Atomic Commits**: Each commit should represent one logical change
3. **Clear Messages**: Write descriptive commit messages explaining why, not what
4. **Small PRs**: Keep pull requests focused on one feature or fix
5. **Reference Beads**: Always include bead ID in commits and PRs
6. **Request Reviews**: Add appropriate reviewers to your PRs

### Git Workflow Example

Complete workflow from start to finish:

```json
{
  "actions": [
    // 1. Make changes and test
    {"type": "run_tests"},
    {"type": "run_linter"},

    // 2. Commit if tests pass
    {
      "type": "git_commit",
      "commit_message": "fix: Resolve authentication timeout issue\n\nFixed JWT token expiration handling...\n\nBead: bead-abc-123\nAgent: agent-worker-1",
      "files": ["src/auth.go", "src/auth_test.go"]
    },

    // 3. Push to remote
    {
      "type": "git_push",
      "set_upstream": true
    },

    // 4. Create PR for review
    {
      "type": "create_pr",
      "pr_title": "Fix authentication timeout issue",
      "pr_reviewers": ["code-reviewer"]
    }
  ],
  "notes": "Completed authentication fix, ready for review"
}
```

### Security Considerations

- **Agent Branches Only**: You can only commit to branches starting with `agent/`
- **No Protected Branches**: Cannot directly commit to main, master, production
- **Secret Detection**: Commits are scanned for API keys, passwords, tokens
- **Branch Naming**: Follow pattern `agent/{bead-id}/{description}`

### Troubleshooting

**Commit Rejected:**
- Check that all required fields are present (bead ID, agent attribution)
- Ensure commit message follows format requirements
- Verify no secrets are being committed

**Push Failed:**
- Ensure branch name starts with `agent/`
- Check SSH keys are configured correctly
- Verify remote repository access

**PR Creation Failed:**
- Install and authenticate gh CLI (`gh auth login`)
- Ensure branch is pushed to remote
- Check that base branch exists
