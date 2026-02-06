# Activity Feed and Notifications System Implementation

## Overview

This document describes the implementation of the activity feed and notifications system for Loom (ac-aiw). The system provides real-time activity tracking and user-specific notifications based on the existing EventBus infrastructure.

## Architecture

```
EventBus → ActivityManager → NotificationManager → User SSE Streams
   ↓            ↓                    ↓
[Events]  [activity_feed]    [notifications]
```

### Key Components

1. **ActivityManager**: Subscribes to EventBus, filters and persists important events, manages aggregation
2. **NotificationManager**: Subscribes to ActivityManager, applies notification rules, manages user preferences
3. **Database**: Four new tables (users, activity_feed, notifications, notification_preferences)
4. **API Handlers**: RESTful and SSE endpoints for both activity feed and notifications

## Database Schema

### 1. users
Stores user information for notification targeting.
- Primary key: `id`
- Unique constraint: `username`
- Indexes: username, role

### 2. activity_feed
Central activity log for all important events.
- Primary key: `id`
- Foreign keys: `project_id`, `agent_id`, `provider_id`
- Indexes: timestamp DESC, project_id, actor_id, event_type, aggregation_key
- Key fields:
  - `aggregation_key`: Groups similar activities
  - `aggregation_count`: Number of aggregated events
  - `visibility`: 'project' or 'global'

### 3. notifications
User-specific notifications derived from activities.
- Primary key: `id`
- Foreign keys: `user_id` (users), `activity_id` (activity_feed)
- Indexes: user_id, status, (user_id, status, created_at DESC)
- Status values: 'unread', 'read', 'archived'
- Priority values: 'low', 'normal', 'high', 'critical'

### 4. notification_preferences
User notification preferences.
- Primary key: `id`
- Unique constraint: `user_id`
- JSON fields: `subscribed_events_json`, `project_filters_json`
- Supports: quiet hours, digest mode, priority threshold

## Event Filtering

### Events Persisted to Activity Feed
✓ `bead.created`, `bead.assigned`, `bead.status_change`, `bead.completed`
✓ `agent.spawned`, `agent.status_change`, `agent.completed`
✓ `project.created`, `project.updated`, `project.deleted`
✓ `provider.registered`, `provider.deleted`, `provider.updated`
✓ `decision.created`, `decision.resolved`
✓ `motivation.fired`, `motivation.enabled`, `motivation.disabled`
✓ `workflow.started`, `workflow.completed`, `workflow.failed`

### Events Filtered Out (Too Noisy)
✗ `agent.heartbeat` (every few seconds)
✗ `log.message` (already in logs table)
✗ `system.idle` (internal state)

## Aggregation Logic

**Goal**: Group related activities within 5-minute windows

**Aggregation Key Format**: `{event_type}.{date-hour}.{project_id}.{actor_id}`

**Example**:
- Agent creates 5 beads within 3 minutes
- Single aggregated activity: "Created 5 beads" (aggregation_count=5)
- Cache maintained in memory, persisted to database

**Implementation**:
1. Check in-memory cache for recent aggregatable activity
2. If not in cache, query database for activities within 5-minute window
3. Either increment existing activity or create new one
4. Update cache and broadcast to SSE subscribers

## Notification Rules

Default rules for triggering user notifications:

### 1. Direct Assignment
**Trigger**: `bead.assigned` where `assigned_to == user_id`
**Priority**: High
**Example**: "You've been assigned to bead: Fix login bug"

### 2. Decision Requires Input
**Trigger**: `decision.created` where `decider_id == user_id`
**Priority**: High
**Example**: "A decision needs your attention: Choose framework"

### 3. Critical Priority Beads
**Trigger**: `bead.created` where `priority == "P0"`
**Priority**: Critical
**Example**: "A P0 bead was created: Production outage"

### 4. System Errors
**Trigger**: `provider.deleted`, `workflow.failed`
**Priority**: Critical
**Example**: "System Alert: Provider GPU-1 deleted"

## API Endpoints

### Activity Feed Endpoints

#### GET /api/v1/activity-feed
Retrieve paginated activity feed with filters.

**Query Parameters**:
- `project_id`: Filter by project
- `event_type`: Filter by event type (e.g., "bead.created")
- `actor_id`: Filter by actor
- `resource_type`: Filter by resource (bead, agent, project, etc.)
- `since`: ISO 8601 timestamp
- `until`: ISO 8601 timestamp
- `limit`: Max results (default: 100)
- `offset`: Skip N results
- `aggregated`: true/false - include only aggregated activities

**Response**:
```json
{
  "activities": [
    {
      "id": "act-abc123",
      "event_type": "bead.created",
      "timestamp": "2026-01-31T10:30:00Z",
      "action": "created",
      "resource_type": "bead",
      "resource_id": "bead-xyz",
      "resource_title": "Fix login bug",
      "project_id": "proj-1",
      "actor_id": "agent-123",
      "aggregation_count": 5,
      "is_aggregated": true
    }
  ],
  "count": 1,
  "limit": 100,
  "offset": 0
}
```

#### GET /api/v1/activity-feed/stream
Server-Sent Events (SSE) stream for real-time activity updates.

**Query Parameters**: Same filters as GET endpoint

**SSE Events**:
- `connected`: Initial connection
- `activity`: New activity (data: Activity JSON)
- Keepalive pings every 30 seconds

### Notification Endpoints

All notification endpoints require authentication.

#### GET /api/v1/notifications
Retrieve user's notifications.

**Query Parameters**:
- `status`: Filter by status (unread, read, archived)
- `priority`: Filter by priority (low, normal, high, critical)
- `limit`: Max results (default: 50)
- `offset`: Skip N results

**Response**:
```json
{
  "notifications": [
    {
      "id": "notif-123",
      "user_id": "user-admin",
      "activity_id": "act-abc123",
      "event_type": "bead.assigned",
      "title": "Bead Assigned to You",
      "message": "You've been assigned to bead: Fix login bug",
      "link": "/beads/bead-xyz",
      "status": "unread",
      "priority": "high",
      "created_at": "2026-01-31T10:30:00Z"
    }
  ],
  "count": 1,
  "limit": 50,
  "offset": 0
}
```

#### GET /api/v1/notifications/stream
SSE stream for user-specific real-time notifications.

**SSE Events**:
- `connected`: Initial connection
- `notification`: New notification (data: Notification JSON)
- Keepalive pings every 30 seconds

#### POST /api/v1/notifications/{id}/read
Mark a specific notification as read.

**Response**:
```json
{
  "message": "Notification marked as read"
}
```

#### POST /api/v1/notifications/mark-all-read
Mark all unread notifications as read for the current user.

**Response**:
```json
{
  "message": "All notifications marked as read"
}
```

#### GET /api/v1/notifications/preferences
Get current user's notification preferences.

**Response**:
```json
{
  "id": "pref-123",
  "user_id": "user-admin",
  "enable_in_app": true,
  "enable_email": false,
  "enable_webhook": false,
  "subscribed_events": ["bead.assigned", "decision.created"],
  "digest_mode": "realtime",
  "quiet_hours_start": "22:00",
  "quiet_hours_end": "08:00",
  "project_filters": ["proj-1", "proj-2"],
  "min_priority": "normal",
  "updated_at": "2026-01-31T10:00:00Z"
}
```

#### PATCH /api/v1/notifications/preferences
Update notification preferences.

**Request Body**: Same as GET response (partial updates supported)

**Response**: Updated preferences object

## Configuration

### Default Preferences
- **enable_in_app**: true
- **enable_email**: false (deferred to Phase 2)
- **enable_webhook**: false (deferred to Phase 2)
- **subscribed_events**: [] (empty = all events)
- **digest_mode**: "realtime"
- **min_priority**: "normal"

### Aggregation Settings
- **Time Window**: 5 minutes
- **Aggregatable Events**: `bead.created`, `agent.spawned`
- **Cache**: In-memory with database fallback

## Permission Filtering

Activity feed respects project-level permissions:

1. **Admin users**: See all activities (global visibility)
2. **Regular users**: See activities from projects they have `projects:read` permission on
3. **Global activities**: Always visible (provider, system-level events)

Implementation: Filter activities by `project_id IN (accessible_projects) OR visibility = 'global'`

## Implementation Details

### Files Created
1. `internal/database/migrations_activity.go` - Database schema migration
2. `internal/database/activity.go` - Database access layer for activities and notifications
3. `internal/activity/types.go` - Activity type definitions
4. `internal/activity/manager.go` - Activity management and aggregation
5. `internal/notifications/types.go` - Notification type definitions
6. `internal/notifications/manager.go` - Notification rules and delivery
7. `internal/api/handlers_activity.go` - Activity feed HTTP handlers
8. `internal/api/handlers_notifications.go` - Notification HTTP handlers

### Files Modified
1. `internal/database/database.go` - Added migration call
2. `internal/loom/loom.go` - Integrated activity and notification managers
3. `internal/api/server.go` - Added routes and helper methods

### Total Lines of Code
- New code: ~2,186 lines
- Modified code: ~44 lines
- **Total: ~2,230 lines**

## Testing Verification Steps

### 1. Database Migration
```bash
# Check tables created
sqlite3 loom.db ".tables" | grep -E "users|activity_feed|notifications|notification_preferences"

# Verify schema
sqlite3 loom.db ".schema activity_feed"

# Check default admin user
sqlite3 loom.db "SELECT * FROM users WHERE username='admin';"
```

### 2. Activity Feed
```bash
# Create a bead (triggers bead.created event)
bd create -t task "Test activity feed"

# Check activity recorded
curl http://localhost:8080/api/v1/activity-feed | jq

# Stream activities in real-time
curl -N http://localhost:8080/api/v1/activity-feed/stream
```

### 3. Notifications
```bash
# Login to get token
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r .token)

# Get notifications
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/notifications | jq

# Stream notifications
curl -N -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/notifications/stream

# Get preferences
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/notifications/preferences | jq

# Update preferences
curl -X PATCH -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enable_email":true,"min_priority":"high"}' \
  http://localhost:8080/api/v1/notifications/preferences | jq
```

### 4. Aggregation
```bash
# Create 5 beads rapidly (should aggregate)
for i in {1..5}; do bd create -t task "Bead $i"; sleep 5; done

# Check activity feed - should show single aggregated activity with count=5
curl http://localhost:8080/api/v1/activity-feed?aggregated=true | jq
```

## Acceptance Criteria (from ac-aiw)

✅ **Activity feed shows all team actions** - Activity feed table persists 15+ key events, API provides paginated access

✅ **Notifications for important events** - NotificationManager creates user-specific notifications based on rules

✅ **Preferences configurable** - Preferences table + API endpoints support event subscriptions, quiet hours, priority thresholds

✅ **Feed filterable** - Query params for project_id, event_type, actor_id, resource_type, date ranges

✅ **Real-time updates** - SSE streams for both activity feed and user-specific notifications

## Future Enhancements (Out of Scope)

1. **Email Notifications**: SMTP integration for email delivery
2. **Webhook Notifications**: HTTP POST to user-configured webhooks
3. **Push Notifications**: Web Push API integration
4. **Activity Comments/Reactions**: Social features on activities
5. **Custom Notification Rules**: User-defined rules beyond defaults
6. **Notification Digests**: Batched hourly/daily summaries
7. **Explicit Team Model**: Team-based scoping (currently using projects)
8. **Full-text Search**: Search across activity descriptions
9. **Activity Analytics**: Dashboards and insights

## Performance Considerations

1. **Aggregation Cache**: In-memory cache reduces database queries for recent activities
2. **Database Indexes**: Optimized for common queries (timestamp DESC, user_id + status)
3. **SSE Buffering**: 100-item buffers prevent blocking on slow clients
4. **Event Filtering**: Only 15 out of 30+ event types persisted
5. **Pagination**: Default limits prevent large result sets

## Security

1. **Authentication Required**: All notification endpoints require valid JWT token
2. **User Isolation**: Users only see their own notifications
3. **Project Permissions**: Activity feed respects project-level access control
4. **No Sensitive Data**: Activities don't include secrets or sensitive config

## Deployment Notes

1. **Database Migration**: Automatic on server startup
2. **Backwards Compatible**: Existing functionality unaffected
3. **No Config Changes**: Works with existing config.yaml
4. **Default User**: Admin user auto-created with username=admin, password=admin
5. **Dependencies**: Requires database.path to be set in config

## Monitoring

Key metrics to monitor:
- Activity feed size growth rate
- Notification delivery latency
- SSE connection count
- Aggregation cache hit rate
- Database query performance

Suggested alerts:
- Activity feed table > 1M rows (consider archival)
- Notification backlog > 1000 unread per user
- SSE connections > 100 concurrent
- Database query time > 500ms

## Conclusion

The activity feed and notifications system provides comprehensive tracking of team activities and intelligent user notifications. Built on Loom's existing EventBus infrastructure, it offers both historical querying and real-time streaming with configurable user preferences and aggregation for improved signal-to-noise ratio.
