# Project Bootstrap Feature Design

## Overview

"New Project" is a bootstrap mode for starting greenfield projects from scratch. Unlike "Add Project" which joins existing projects, New Project creates a complete project from a Product Requirements Document (PRD).

## User Flow

### 1. Project Creation Entry Point

**UI: Add Project Dialog**
- Button: "Add Project" (existing - joins project)
- Button: "New Project" (new - bootstrap mode)

When user clicks "New Project":

```
┌─────────────────────────────────────────┐
│  Create New Project                     │
├─────────────────────────────────────────┤
│                                         │
│  GitHub Repository:                     │
│  ┌─────────────────────────────────┐   │
│  │ https://github.com/user/project │   │
│  └─────────────────────────────────┘   │
│                                         │
│  Project Name:                          │
│  ┌─────────────────────────────────┐   │
│  │ My Awesome Project              │   │
│  └─────────────────────────────────┘   │
│                                         │
│  Branch: main                           │
│                                         │
│  Product Requirements Document:         │
│  ┌─────────────────────────────────┐   │
│  │ Upload PRD │ Enter Text         │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │                                 │   │
│  │ [PRD text area or file upload]  │   │
│  │                                 │   │
│  │                                 │   │
│  └─────────────────────────────────┘   │
│                                         │
│  [Cancel]              [Create Project] │
└─────────────────────────────────────────┘
```

### 2. Initial Setup Phase

**Prerequisites:**
- GitHub repository exists (can be empty except LICENSE, README.md)
- Repository is accessible (user has write permissions)
- PRD is provided (text or file upload)

**Backend Actions:**
1. Clone/initialize repository locally
2. Create project structure:
   ```
   project-root/
   ├── plans/
   │   └── BOOTSTRAP.md          # User's initial PRD
   ├── .beads/                   # Beads directory (initialized)
   ├── settings.json             # Copied from Loom template
   └── .mcp.json                 # MCP configuration (if needed)
   ```

### 3. PRD Processing Phase

#### Step 1: Save Initial PRD
- Save user's PRD to `plans/BOOTSTRAP.md`
- Commit to repository: "chore: initialize project with initial PRD"

#### Step 2: Create Project Manager Bead
Create a P0 bead assigned to **Project Manager**:

```yaml
title: "[Bootstrap] Expand PRD with Best Practices"
type: task
priority: P0
assignee: project-manager
description: |
  Transform the initial PRD into a comprehensive, actionable PRD following best practices.

  ## Input
  - Initial PRD: plans/BOOTSTRAP.md
  - MCP guidance: responsible-vibe-mcp

  ## Tasks
  1. Review initial PRD for completeness
  2. Consult responsible-vibe-mcp for best practices
  3. Expand PRD with:
     - Clear user stories
     - Technical requirements
     - Architecture considerations
     - Success criteria
     - Acceptance criteria
     - Non-functional requirements (performance, security, etc.)
  4. Save expanded PRD to plans/ORIGINAL_PRD.md
  5. Copy initial PRD to plans/BOOTSTRAP.md (preserve)

  ## Deliverables
  - plans/ORIGINAL_PRD.md (fully expanded PRD)
  - plans/BOOTSTRAP.md (original, unchanged)

  ## Guidelines
  - Follow responsible-vibe-mcp best practices
  - Use clear, unambiguous language
  - Include measurable success criteria
  - Document technical constraints
  - Define MVP scope explicitly
```

#### Step 3: Wait for PM Completion
- Project Manager agent processes the bead
- Uses responsible-vibe-mcp MCP for guidance
- Creates comprehensive PRD in `plans/ORIGINAL_PRD.md`
- Marks bead complete

### 4. Epic and Story Creation Phase

#### Step 1: Create Epic Breakdown Bead
After PM completes PRD expansion, automatically create:

```yaml
title: "[Bootstrap] Create Epics and Stories from PRD"
type: task
priority: P0
assignee: project-manager
description: |
  Break down the comprehensive PRD into actionable epics and stories as beads.

  ## Input
  - Comprehensive PRD: plans/ORIGINAL_PRD.md
  - responsible-vibe-mcp guidance

  ## Tasks
  1. Identify major features (epics)
  2. Break down each epic into user stories (tasks)
  3. Create bead hierarchy:
     - Epic beads (type: epic, P1-P2)
     - Story beads (type: task, P2-P3, parent: epic-id)
  4. Assign beads to appropriate roles:
     - UI/UX work → web-designer or web-designer-engineer
     - Core engineering → engineering-manager
     - Infrastructure → devops-engineer
     - Testing strategy → qa-engineer
  5. Set dependencies between beads
  6. Ensure MVP features are P1, enhancements are P2-P3

  ## Acceptance Criteria
  - All major features have epic beads
  - Epics broken into concrete, actionable story beads
  - Beads assigned to appropriate agent roles
  - Dependencies set correctly (blockers, blocked-by)
  - Clear acceptance criteria on each story bead

  ## Guidelines
  - Follow responsible-vibe-mcp decomposition best practices
  - Keep stories small and focused (1-3 days max)
  - Set realistic priorities based on MVP definition
  - Include documentation and testing stories
```

#### Step 2: Agent Work Begins
Once PM creates epics and stories:
- **Engineering Manager**: Core backend/logic work
- **Web Designer Engineer**: UI components, styling
- **DevOps Engineer**: Infrastructure, deployment, CI/CD
- **QA Engineer**: Test strategy, test cases

Agents work autonomously on assigned beads following workflow:
1. Investigate/Research
2. Implement
3. Verify (QA)
4. Review (Code Reviewer)
5. Commit

### 5. Integration and Demo Phase

#### Step 1: Demo Readiness Detection
System monitors for:
- All MVP epic beads complete (or blocked)
- Core features implemented and tested
- Application can be built/deployed

When ready, automatically create:

```yaml
title: "[Demo] Review and Test Application"
type: decision
priority: P0
assignee: ceo
description: |
  The initial implementation is ready for review and testing.

  ## What's Been Built
  [Auto-generated summary of completed epics and features]

  ## How to Launch
  [Instructions from docs/DEPLOYMENT.md or README.md]

  Example:
  ```bash
  npm install
  npm run dev
  # Open http://localhost:3000
  ```

  ## Testing Checklist
  - [ ] Application launches successfully
  - [ ] Core features work as expected
  - [ ] UI is presentable and functional
  - [ ] No critical bugs observed
  - [ ] Meets MVP acceptance criteria

  ## Next Steps
  Choose one:
  1. **Approve and Complete**: Mark project MVP as done
  2. **Request Changes**: Create follow-up beads for improvements
  3. **Pivot**: Major changes needed, update PRD and restart

  ## Feedback
  [Provide detailed feedback here]
```

#### Step 2: CEO Review
CEO (human) reviews the demo:
- Launches application
- Tests core features
- Provides feedback in the bead

#### Step 3: Iteration or Completion
Based on CEO feedback:

**Option A: Approve** - CEO marks bead complete
- Project status: MVP Complete
- Optionally continue with enhancement beads

**Option B: Request Changes** - CEO creates new beads or comments
- PM or agents pick up feedback beads
- Cycle continues with full context of previous work
- CEO gets new demo bead when ready

**Option C: Major Pivot** - CEO updates PRD
- New PRD saved to `plans/PRD_v2.md`
- PM creates new epics/stories
- Previous work preserved, new direction started

## File Structure

### After Bootstrap Initialization
```
project-root/
├── .beads/                    # Initialized by bd init
│   ├── beads/                 # Bead YAML files
│   └── issues.jsonl           # Git-backed issue tracking
├── plans/
│   ├── BOOTSTRAP.md           # Original user-provided PRD (preserved)
│   └── ORIGINAL_PRD.md        # Expanded by PM (after Step 3)
├── settings.json              # Copied from Loom template
├── .mcp.json                  # MCP server configuration
├── README.md                  # May exist from GitHub init
└── LICENSE                    # May exist from GitHub init
```

### After Epic/Story Creation
```
project-root/
├── .beads/
│   ├── beads/
│   │   ├── epic-001-user-auth.yaml
│   │   ├── story-001-login-page.yaml
│   │   ├── story-002-auth-api.yaml
│   │   └── ... (many more story beads)
│   └── issues.jsonl
├── plans/
│   ├── BOOTSTRAP.md
│   ├── ORIGINAL_PRD.md
│   └── ARCHITECTURE.md        # Created by engineering-manager if needed
├── settings.json
├── .mcp.json
├── docs/                      # Created by agents during implementation
│   ├── DEPLOYMENT.md
│   └── API.md
└── ... (application code created by agents)
```

## Integration with responsible-vibe-mcp

### MCP Configuration
The `settings.json` copied into new projects includes:

```json
{
  "mcpServers": {
    "responsible-vibe-mcp": {
      "command": "npx",
      "args": ["-y", "responsible-vibe-mcp"]
    }
  }
}
```

### MCP Usage by Agents

**Project Manager**:
- Calls `start_development` with workflow selection
- Uses `whats_next` for guidance on PRD expansion
- Follows phase-based development approach
- Uses `setup_project_docs` for architecture templates

**All Agents**:
- Access `get_tool_info` for workflow guidance
- Follow phase transitions via `proceed_to_phase`
- Use structured planning approach from MCP

## Implementation Components

### Backend API

**New Endpoint: POST /api/projects/bootstrap**
```typescript
interface BootstrapProjectRequest {
  github_url: string;
  name: string;
  branch: string;
  prd_text?: string;        // PRD as text
  prd_file?: File;          // Or uploaded file
}

interface BootstrapProjectResponse {
  project_id: string;
  status: "initializing" | "ready";
  initial_bead_id: string;  // PM's PRD expansion bead
}
```

**Workflow:**
1. Validate GitHub URL and access
2. Clone repository (or initialize if empty)
3. Create project directory structure
4. Copy template files (settings.json, .mcp.json)
5. Initialize beads (`bd init`)
6. Save PRD to `plans/BOOTSTRAP.md`
7. Create PM bead for PRD expansion
8. Commit initial structure
9. Register project in Loom
10. Return project_id and bead_id

### Frontend UI

**New Component: NewProjectDialog.tsx**
- Form with fields: GitHub URL, name, branch, PRD input
- PRD input: tabbed interface (text area vs file upload)
- Validation: GitHub URL format, PRD not empty
- Loading states during bootstrap
- Success: Redirect to project view showing initial PM bead

**Modified Component: AddProjectButton.tsx**
- Split button: "Add Project" (dropdown)
  - "Join Existing Project" (current functionality)
  - "Create New Project" (new bootstrap flow)

### Project Service

**New Method: `BootstrapProject()`**
```go
func (s *ProjectService) BootstrapProject(ctx context.Context, req BootstrapProjectRequest) (*Project, error) {
    // 1. Validate and clone repo
    // 2. Create directory structure
    // 3. Copy template files
    // 4. Initialize beads
    // 5. Create PM bead
    // 6. Register project
    // 7. Trigger initial workflow
    return project, nil
}
```

## Template Files

### settings.json (copied to new projects)
```json
{
  "mcpServers": {
    "responsible-vibe-mcp": {
      "command": "npx",
      "args": ["-y", "responsible-vibe-mcp"]
    }
  },
  "workflowMode": "guided",
  "enableReviews": true
}
```

### .mcp.json (if needed)
```json
{
  "mcpServers": {
    "responsible-vibe-mcp": {
      "command": "npx",
      "args": ["-y", "responsible-vibe-mcp"]
    }
  }
}
```

## User Experience

### Happy Path Timeline

1. **T+0**: User clicks "New Project", provides GitHub URL and PRD
2. **T+30s**: Project bootstrapped, PM bead created and assigned
3. **T+2min**: PM agent expands PRD using MCP guidance
4. **T+5min**: PM agent creates 5-10 epic beads, 20-40 story beads
5. **T+10min**: Agents begin autonomous work on assigned beads
6. **T+2hr**: Core features implemented, tests written
7. **T+3hr**: CEO demo bead created, human reviews application
8. **T+3.5hr**: CEO provides feedback, new beads created or project approved

### Error Handling

**Invalid GitHub URL**
- Validate before starting bootstrap
- Show clear error message with examples

**Repository Not Empty**
- Warn user: "Repository has existing files. Use 'Add Project' instead?"
- Allow override: "Bootstrap anyway" (merge with existing)

**PRD Too Vague**
- PM agent detects insufficient detail
- Creates clarification bead for CEO
- Requests specific information before proceeding

**No Agents Available**
- Queue bootstrap operation
- Show status: "Waiting for available agents"

## Testing Strategy

### Manual Testing
1. Create test repository on GitHub
2. Use "New Project" with sample PRD
3. Verify directory structure created
4. Confirm PM bead appears and is worked on
5. Check that epics/stories are created
6. Monitor agent work on beads
7. Test CEO demo bead creation
8. Complete full cycle to MVP

### Automated Testing
- Unit tests for bootstrap service
- Integration tests for file creation
- E2E tests for full bootstrap flow
- Mock GitHub API for CI/CD testing

## Success Metrics

- **Bootstrap Time**: < 2 minutes from click to PM working
- **PRD Quality**: PM-expanded PRD includes all required sections
- **Bead Creation**: Average 8-12 epics, 30-50 stories per project
- **Agent Utilization**: All agent roles receive appropriate beads
- **Time to Demo**: < 4 hours for simple projects
- **CEO Approval Rate**: > 70% projects approved on first demo

## Future Enhancements

1. **PRD Templates**: Pre-built PRD templates for common project types
2. **Tech Stack Selection**: Let user choose framework (React, Vue, etc.)
3. **Architecture Patterns**: Select architecture (MVC, microservices, etc.)
4. **Integration Wizards**: Connect to external services (Auth0, Stripe)
5. **Progress Visualization**: Real-time progress dashboard during bootstrap
6. **AI PRD Assistant**: Help user write better initial PRDs

## Related Documentation

- [Workflow System](WORKFLOW_SYSTEM_PHASE2.md)
- [responsible-vibe-mcp](https://mrsimpson.github.io/responsible-vibe-mcp/)
- [Beads Workflow](BEADS_WORKFLOW.md)
- [Agent Roles](AGENT_ROLES.md)

## Implementation Priority

**Phase 1 (MVP)**: P0 for first release
- Backend bootstrap endpoint
- Frontend "New Project" dialog
- Template file copying
- PM PRD expansion bead creation

**Phase 2** (Post-launch enhancement): P1
- Epic/Story auto-creation by PM
- Agent auto-assignment
- Demo bead creation

**Phase 3** (Future): P2
- PRD templates
- Tech stack selection
- Progress visualization
