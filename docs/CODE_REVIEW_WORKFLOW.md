# Code Review Workflow

## Overview

Loom's automated code review system enables code-reviewer agents to review pull requests, provide feedback, and ensure code quality before merging. This document describes the end-to-end workflow for automated PR reviews.

## Workflow Diagram

```
┌─────────────────┐
│ PR Created/     │
│ Updated         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ GitHub Webhook  │◄─── PR opened, synchronize, ready_for_review events
│ Event Received  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Parse PR Event  │
│ - PR number     │
│ - Author        │
│ - Branch        │
│ - Changed files │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Create Review   │
│ Bead            │
│ - Type: pr-rev  │
│ - Metadata      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Assign to       │
│ Code Reviewer   │
│ Agent           │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────┐
│ Code Reviewer Performs Review       │
│ ┌─────────────────────────────────┐ │
│ │ 1. Fetch PR Details             │ │
│ │ 2. Get Changed Files & Diffs    │ │
│ │ 3. Analyze Code Quality         │ │
│ │ 4. Check Against Criteria       │ │
│ │ 5. Generate Review Comments     │ │
│ └─────────────────────────────────┘ │
└────────┬────────────────────────────┘
         │
         ▼
    ┌────────┐
    │Review  │
    │Result? │
    └───┬────┘
        │
    ┌───┴───┬──────────────┐
    │       │              │
    ▼       ▼              ▼
┌─────┐ ┌──────┐     ┌──────────┐
│Appr-│ │Chang-│     │Comment   │
│ove  │ │es    │     │Only      │
│     │ │Req'd │     │          │
└──┬──┘ └───┬──┘     └────┬─────┘
   │        │             │
   │        │             │
   ▼        ▼             ▼
┌─────────────────────────────┐
│ Post Review to GitHub       │
│ - Approve / Request Changes │
│ - Add inline comments       │
│ - Add summary comment       │
└────────┬────────────────────┘
         │
         ▼
    ┌────────┐
    │Changes │
    │Req'd?  │
    └───┬────┘
        │
    ┌───┴───┐
    │       │
    ▼       ▼
  ┌───┐  ┌──────────┐
  │No │  │Yes       │
  │   │  │          │
  └─┬─┘  └────┬─────┘
    │         │
    │         ▼
    │   ┌──────────────┐
    │   │ Wait for     │
    │   │ Author Fixes │
    │   └──────┬───────┘
    │          │
    │          ▼
    │   ┌──────────────┐
    │   │ PR Updated   │
    │   │ Event        │
    │   └──────┬───────┘
    │          │
    │          └──────────┐
    │                     │
    ▼                     ▼
┌─────────────────────────────┐
│ Close Review Bead           │
│ Mark as Approved/Rejected   │
└─────────────────────────────┘
```

## Integration Points

### 1. GitHub Webhooks

**Webhook URL:** `POST /api/webhooks/github`

**Events to Subscribe:**
- `pull_request` - opened, reopened, synchronize, ready_for_review
- `pull_request_review` - submitted (for tracking external reviews)
- `pull_request_review_comment` - created (for tracking discussion)

**Webhook Payload Processing:**
```json
{
  "action": "opened",
  "number": 123,
  "pull_request": {
    "id": 456,
    "number": 123,
    "title": "Add feature X",
    "user": {"login": "author"},
    "head": {"ref": "feature-branch", "sha": "abc123"},
    "base": {"ref": "main", "sha": "def456"},
    "state": "open",
    "draft": false,
    "changed_files": 5,
    "additions": 150,
    "deletions": 30
  },
  "repository": {
    "full_name": "org/repo",
    "clone_url": "https://github.com/org/repo.git"
  }
}
```

### 2. GitHub API Integration

**Required API Endpoints:**
- `GET /repos/:owner/:repo/pulls/:number` - Fetch PR details
- `GET /repos/:owner/:repo/pulls/:number/files` - Get changed files
- `GET /repos/:owner/:repo/pulls/:number/reviews` - Get existing reviews
- `POST /repos/:owner/:repo/pulls/:number/reviews` - Submit review
- `POST /repos/:owner/:repo/pulls/:number/comments` - Add inline comment

**Authentication:**
- GitHub App installation token (preferred)
- Personal Access Token (fallback)
- Required scopes: `repo`, `pull_request`, `write:discussion`

### 3. Temporal Workflow Integration

**Workflow:** `CodeReviewWorkflow`

**Activities:**
- `FetchPRDetailsActivity` - Get PR information from GitHub
- `FetchPRFilesActivity` - Get changed files and diffs
- `AnalyzeCodeActivity` - Run code analysis (linting, security scans)
- `GenerateReviewCommentsActivity` - AI-powered review comment generation
- `PostReviewActivity` - Submit review to GitHub

**Signals:**
- `PRUpdatedSignal` - Triggered when PR is updated after review
- `ReviewApprovedSignal` - Manual approval override
- `ReviewCancelledSignal` - Cancel review (PR closed)

### 4. Beads Integration

**Bead Type:** `pr-review`

**Bead Metadata:**
```json
{
  "pr_number": 123,
  "pr_url": "https://github.com/org/repo/pull/123",
  "author": "username",
  "branch": "feature-branch",
  "base_branch": "main",
  "files_changed": 5,
  "additions": 150,
  "deletions": 30,
  "review_status": "in_progress|approved|changes_requested|commented",
  "reviewer_agent_id": "agent-123",
  "review_submitted_at": "2026-02-05T20:30:00Z"
}
```

## Review Criteria Checklist

The code reviewer agent evaluates PRs against the following criteria:

### 1. Code Quality (Weight: 30%)
- [ ] Code follows project style guidelines
- [ ] Functions are appropriately sized (< 50 lines)
- [ ] Clear, descriptive variable and function names
- [ ] Appropriate use of comments (why, not what)
- [ ] No commented-out code or debug statements
- [ ] DRY principle applied (no code duplication)

### 2. Functionality (Weight: 25%)
- [ ] Code accomplishes stated objective
- [ ] Edge cases handled appropriately
- [ ] Error handling is comprehensive
- [ ] No obvious bugs or logic errors
- [ ] Performance considerations addressed

### 3. Testing (Weight: 20%)
- [ ] Tests included for new functionality
- [ ] Tests cover happy path and edge cases
- [ ] Test names are descriptive
- [ ] Tests are independent and repeatable
- [ ] Mocking/stubbing used appropriately
- [ ] Test coverage meets project standards (>80%)

### 4. Security (Weight: 15%)
- [ ] No SQL injection vulnerabilities
- [ ] No command injection vulnerabilities
- [ ] No path traversal vulnerabilities
- [ ] Input validation implemented
- [ ] Authentication/authorization checked
- [ ] Secrets not hardcoded
- [ ] Dependencies are up-to-date

### 5. Documentation (Weight: 10%)
- [ ] Public APIs documented
- [ ] README updated if needed
- [ ] Architecture docs updated for significant changes
- [ ] Breaking changes clearly marked
- [ ] Migration guide provided if needed

### Scoring System

- **90-100%**: Approve ✅
- **70-89%**: Comment with suggestions, approve conditionally 💬
- **Below 70%**: Request changes ❌

## Approval/Rejection Flow

### Approval Flow

```
PR meets criteria (≥90%)
    ↓
Agent posts review comment with summary
    ↓
Agent approves PR (GitHub review state: APPROVED)
    ↓
Bead marked as completed (status: approved)
    ↓
Optional: Auto-merge if configured
```

### Request Changes Flow

```
PR does not meet criteria (<70%)
    ↓
Agent identifies specific issues
    ↓
Agent posts inline comments on problematic lines
    ↓
Agent posts summary comment with checklist
    ↓
Agent requests changes (GitHub review state: CHANGES_REQUESTED)
    ↓
Bead marked as blocked (status: changes_requested)
    ↓
Wait for PR update event
    ↓
New commit pushed
    ↓
Re-trigger review workflow
```

### Comment Only Flow

```
PR mostly meets criteria (70-89%)
    ↓
Agent posts suggestions and recommendations
    ↓
Agent posts as comment (GitHub review state: COMMENT)
    ↓
Bead marked as completed (status: commented)
    ↓
Human reviewer makes final decision
```

## Review Comments Format

### Summary Comment Template

```markdown
## Code Review Summary

**Overall Score:** 85/100 ✓

### Strengths
- Well-structured code with clear separation of concerns
- Comprehensive test coverage (92%)
- Good error handling

### Issues Found
- 🔴 **Critical (1):** Potential SQL injection in `users.go:45`
- 🟡 **Medium (2):** Missing input validation in API handlers
- 🟢 **Minor (3):** Code style inconsistencies

### Recommendations
1. Add prepared statements for database queries
2. Implement input validation using validator library
3. Run `gofmt` to fix formatting

### Action Required
☑️ Please address the critical issue before merging.

---
🤖 Automated review by Loom Code Reviewer
```

### Inline Comment Template

```markdown
🔴 **Security Issue:** Potential SQL injection

This query concatenates user input directly into SQL:
```go
query := "SELECT * FROM users WHERE id = " + userID
```

**Fix:**
```go
query := "SELECT * FROM users WHERE id = ?"
db.Query(query, userID)
```

**Severity:** Critical
**Category:** Security
```

## Configuration

### Project-Level Configuration

File: `.loom/review-config.yaml`

```yaml
code_review:
  enabled: true
  auto_assign: true
  reviewer_agent: code-reviewer-default

  # Review triggers
  triggers:
    - pr_opened
    - pr_updated
    - pr_ready_for_review

  # Auto-approval settings
  auto_approve:
    enabled: false
    min_score: 95
    require_tests: true

  # Review criteria weights
  criteria_weights:
    code_quality: 30
    functionality: 25
    testing: 20
    security: 15
    documentation: 10

  # Minimum score thresholds
  thresholds:
    approve: 90
    request_changes: 70

  # File patterns to ignore
  ignore_patterns:
    - "*.md"
    - "docs/**"
    - "*_test.go"  # Don't review test files as strictly

  # Security scanning
  security:
    enabled: true
    fail_on_critical: true
    scanners:
      - gosec
      - semgrep
```

## Agent Actions

### New Actions for Code Review

1. **fetch_pr** - Get PR details from GitHub
   ```json
   {
     "type": "fetch_pr",
     "pr_number": 123,
     "include_files": true,
     "include_diff": true
   }
   ```

2. **review_code** - Analyze code against criteria
   ```json
   {
     "type": "review_code",
     "pr_number": 123,
     "criteria": ["quality", "security", "testing"]
   }
   ```

3. **add_review_comment** - Post inline or summary comment
   ```json
   {
     "type": "add_review_comment",
     "pr_number": 123,
     "body": "Review summary...",
     "path": "file.go",
     "line": 45,
     "side": "RIGHT"
   }
   ```

4. **submit_review** - Submit overall review decision
   ```json
   {
     "type": "submit_review",
     "pr_number": 123,
     "event": "APPROVE|REQUEST_CHANGES|COMMENT",
     "body": "Overall review summary"
   }
   ```

5. **request_review** - Request review from another agent
   ```json
   {
     "type": "request_review",
     "pr_number": 123,
     "reviewer": "code-reviewer-security"
   }
   ```

## Implementation Phases

### Phase 1: Webhook Integration (ac-8yp.2)
- Implement GitHub webhook endpoint
- Parse PR events
- Create review beads
- Assign to code reviewer agent

### Phase 2: Review Actions (ac-8yp.3)
- Implement fetch_pr action
- Implement review_code action
- Implement add_review_comment action
- Implement submit_review action
- GitHub API client integration

### Phase 3: Code Reviewer Persona (ac-8yp.4)
- Update code-reviewer persona with review logic
- Implement criteria evaluation
- Implement scoring algorithm
- Implement comment generation

### Phase 4: Feedback Loop (ac-8yp.5)
- Handle PR update events after review
- Re-review workflow
- Track review iterations
- Handle edge cases (PR closed, force-pushed)

## Success Metrics

- **Review Latency:** Time from PR opened to review submitted < 5 minutes
- **Review Quality:** Agent-generated comments are helpful (measured by author feedback)
- **Accuracy:** Agent approval/rejection aligns with human reviewer decisions ≥85%
- **Coverage:** Percentage of PRs that receive automated review ≥95%
- **False Positives:** Incorrect "request changes" decisions <10%

## Future Enhancements

1. **Multi-Agent Review:** Multiple specialized reviewers (security, performance, style)
2. **Learning from Feedback:** Train models on accepted/rejected reviews
3. **Custom Linters:** Project-specific code analysis rules
4. **Review Templates:** Customizable review comment templates
5. **Review Analytics:** Dashboard showing review trends and metrics
