package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Activity represents an activity feed entry
type Activity struct {
	ID               string
	EventType        string
	EventID          string
	Timestamp        time.Time
	Source           string
	ActorID          string
	ActorType        string
	ProjectID        string
	AgentID          string
	BeadID           string
	ProviderID       string
	Action           string
	ResourceType     string
	ResourceID       string
	ResourceTitle    string
	MetadataJSON     string
	AggregationKey   string
	AggregationCount int
	IsAggregated     bool
	Visibility       string
}

// CreateActivity inserts a new activity
func (d *Database) CreateActivity(activity *Activity) error {
	query := `
		INSERT INTO activity_feed (
			id, event_type, event_id, timestamp, source, actor_id, actor_type,
			project_id, agent_id, bead_id, provider_id, action, resource_type,
			resource_id, resource_title, metadata_json, aggregation_key,
			aggregation_count, is_aggregated, visibility
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		activity.ID,
		activity.EventType,
		sqlNullString(activity.EventID),
		activity.Timestamp,
		activity.Source,
		sqlNullString(activity.ActorID),
		sqlNullString(activity.ActorType),
		sqlNullString(activity.ProjectID),
		sqlNullString(activity.AgentID),
		sqlNullString(activity.BeadID),
		sqlNullString(activity.ProviderID),
		activity.Action,
		activity.ResourceType,
		activity.ResourceID,
		sqlNullString(activity.ResourceTitle),
		sqlNullString(activity.MetadataJSON),
		sqlNullString(activity.AggregationKey),
		activity.AggregationCount,
		activity.IsAggregated,
		activity.Visibility,
	)

	if err != nil {
		return fmt.Errorf("failed to create activity: %w", err)
	}
	return nil
}

// GetRecentAggregatableActivity finds an aggregatable activity within a time window
func (d *Database) GetRecentAggregatableActivity(aggregationKey string, since time.Time) (*Activity, error) {
	query := `
		SELECT id, event_type, event_id, timestamp, source, actor_id, actor_type,
			   project_id, agent_id, bead_id, provider_id, action, resource_type,
			   resource_id, resource_title, metadata_json, aggregation_key,
			   aggregation_count, is_aggregated, visibility
		FROM activity_feed
		WHERE aggregation_key = ? AND timestamp >= ? AND is_aggregated = 1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	activity := &Activity{}
	var eventID, actorID, actorType, projectID, agentID, beadID, providerID, resourceTitle, metadataJSON, aggKey sql.NullString

	err := d.db.QueryRow(query, aggregationKey, since).Scan(
		&activity.ID,
		&activity.EventType,
		&eventID,
		&activity.Timestamp,
		&activity.Source,
		&actorID,
		&actorType,
		&projectID,
		&agentID,
		&beadID,
		&providerID,
		&activity.Action,
		&activity.ResourceType,
		&activity.ResourceID,
		&resourceTitle,
		&metadataJSON,
		&aggKey,
		&activity.AggregationCount,
		&activity.IsAggregated,
		&activity.Visibility,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get recent aggregatable activity: %w", err)
	}

	// Convert nullable fields
	activity.EventID = eventID.String
	activity.ActorID = actorID.String
	activity.ActorType = actorType.String
	activity.ProjectID = projectID.String
	activity.AgentID = agentID.String
	activity.BeadID = beadID.String
	activity.ProviderID = providerID.String
	activity.ResourceTitle = resourceTitle.String
	activity.MetadataJSON = metadataJSON.String
	activity.AggregationKey = aggKey.String

	return activity, nil
}

// UpdateAggregatedActivity updates an aggregated activity's count
func (d *Database) UpdateAggregatedActivity(activityID string, newCount int) error {
	query := `
		UPDATE activity_feed
		SET aggregation_count = ?, is_aggregated = 1
		WHERE id = ?
	`

	_, err := d.db.Exec(query, newCount, activityID)
	if err != nil {
		return fmt.Errorf("failed to update aggregated activity: %w", err)
	}
	return nil
}

// ListActivities retrieves activities with filters
func (d *Database) ListActivities(filters ActivityFilters) ([]*Activity, error) {
	query := `
		SELECT id, event_type, event_id, timestamp, source, actor_id, actor_type,
			   project_id, agent_id, bead_id, provider_id, action, resource_type,
			   resource_id, resource_title, metadata_json, aggregation_key,
			   aggregation_count, is_aggregated, visibility
		FROM activity_feed
		WHERE 1=1
	`
	args := []interface{}{}

	if len(filters.ProjectIDs) > 0 {
		placeholders := ""
		for i, pid := range filters.ProjectIDs {
			if i > 0 {
				placeholders += ", "
			}
			placeholders += "?"
			args = append(args, pid)
		}
		query += fmt.Sprintf(" AND (project_id IN (%s) OR visibility = 'global')", placeholders)
	}

	if filters.EventType != "" {
		query += " AND event_type = ?"
		args = append(args, filters.EventType)
	}

	if filters.ActorID != "" {
		query += " AND actor_id = ?"
		args = append(args, filters.ActorID)
	}

	if filters.ResourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, filters.ResourceType)
	}

	if !filters.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filters.Since)
	}

	if !filters.Until.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filters.Until)
	}

	if filters.Aggregated != nil {
		query += " AND is_aggregated = ?"
		args = append(args, *filters.Aggregated)
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list activities: %w", err)
	}
	defer rows.Close()

	var activities []*Activity
	for rows.Next() {
		activity := &Activity{}
		var eventID, actorID, actorType, projectID, agentID, beadID, providerID, resourceTitle, metadataJSON, aggKey sql.NullString

		err := rows.Scan(
			&activity.ID,
			&activity.EventType,
			&eventID,
			&activity.Timestamp,
			&activity.Source,
			&actorID,
			&actorType,
			&projectID,
			&agentID,
			&beadID,
			&providerID,
			&activity.Action,
			&activity.ResourceType,
			&activity.ResourceID,
			&resourceTitle,
			&metadataJSON,
			&aggKey,
			&activity.AggregationCount,
			&activity.IsAggregated,
			&activity.Visibility,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}

		// Convert nullable fields
		activity.EventID = eventID.String
		activity.ActorID = actorID.String
		activity.ActorType = actorType.String
		activity.ProjectID = projectID.String
		activity.AgentID = agentID.String
		activity.BeadID = beadID.String
		activity.ProviderID = providerID.String
		activity.ResourceTitle = resourceTitle.String
		activity.MetadataJSON = metadataJSON.String
		activity.AggregationKey = aggKey.String

		activities = append(activities, activity)
	}

	return activities, nil
}

// ActivityFilters defines filters for querying activities
type ActivityFilters struct {
	ProjectIDs   []string
	EventType    string
	ActorID      string
	ResourceType string
	Since        time.Time
	Until        time.Time
	Limit        int
	Offset       int
	Aggregated   *bool
}

// Notification represents a user notification
type Notification struct {
	ID           string
	UserID       string
	ActivityID   string
	EventType    string
	Title        string
	Message      string
	Link         string
	Status       string
	Priority     string
	MetadataJSON string
	CreatedAt    time.Time
	ReadAt       *time.Time
	ArchivedAt   *time.Time
}

// CreateNotification inserts a new notification
func (d *Database) CreateNotification(notification *Notification) error {
	query := `
		INSERT INTO notifications (
			id, user_id, activity_id, event_type, title, message, link,
			status, priority, metadata_json, created_at, read_at, archived_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		notification.ID,
		notification.UserID,
		sqlNullString(notification.ActivityID),
		notification.EventType,
		notification.Title,
		notification.Message,
		sqlNullString(notification.Link),
		notification.Status,
		notification.Priority,
		sqlNullString(notification.MetadataJSON),
		notification.CreatedAt,
		sqlNullTime(notification.ReadAt),
		sqlNullTime(notification.ArchivedAt),
	)

	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

// ListNotifications retrieves notifications for a user
func (d *Database) ListNotifications(userID string, status string, limit, offset int) ([]*Notification, error) {
	query := `
		SELECT id, user_id, activity_id, event_type, title, message, link,
			   status, priority, metadata_json, created_at, read_at, archived_at
		FROM notifications
		WHERE user_id = ?
	`
	args := []interface{}{userID}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		var activityID, link, metadataJSON sql.NullString
		var readAt, archivedAt sql.NullTime

		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&activityID,
			&notification.EventType,
			&notification.Title,
			&notification.Message,
			&link,
			&notification.Status,
			&notification.Priority,
			&metadataJSON,
			&notification.CreatedAt,
			&readAt,
			&archivedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		notification.ActivityID = activityID.String
		notification.Link = link.String
		notification.MetadataJSON = metadataJSON.String

		if readAt.Valid {
			notification.ReadAt = &readAt.Time
		}
		if archivedAt.Valid {
			notification.ArchivedAt = &archivedAt.Time
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// MarkNotificationRead marks a notification as read
func (d *Database) MarkNotificationRead(notificationID string) error {
	query := `
		UPDATE notifications
		SET status = 'read', read_at = ?
		WHERE id = ? AND status = 'unread'
	`

	_, err := d.db.Exec(query, time.Now(), notificationID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	return nil
}

// MarkAllNotificationsRead marks all unread notifications as read for a user
func (d *Database) MarkAllNotificationsRead(userID string) error {
	query := `
		UPDATE notifications
		SET status = 'read', read_at = ?
		WHERE user_id = ? AND status = 'unread'
	`

	_, err := d.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}

// NotificationPreferences represents user notification preferences
type NotificationPreferences struct {
	ID                  string
	UserID              string
	EnableInApp         bool
	EnableEmail         bool
	EnableWebhook       bool
	SubscribedEventsJSON string
	DigestMode          string
	QuietHoursStart     string
	QuietHoursEnd       string
	ProjectFiltersJSON  string
	MinPriority         string
	UpdatedAt           time.Time
}

// GetNotificationPreferences retrieves notification preferences for a user
func (d *Database) GetNotificationPreferences(userID string) (*NotificationPreferences, error) {
	query := `
		SELECT id, user_id, enable_in_app, enable_email, enable_webhook,
			   subscribed_events_json, digest_mode, quiet_hours_start,
			   quiet_hours_end, project_filters_json, min_priority, updated_at
		FROM notification_preferences
		WHERE user_id = ?
	`

	prefs := &NotificationPreferences{}
	var subscribedEvents, quietStart, quietEnd, projectFilters sql.NullString

	err := d.db.QueryRow(query, userID).Scan(
		&prefs.ID,
		&prefs.UserID,
		&prefs.EnableInApp,
		&prefs.EnableEmail,
		&prefs.EnableWebhook,
		&subscribedEvents,
		&prefs.DigestMode,
		&quietStart,
		&quietEnd,
		&projectFilters,
		&prefs.MinPriority,
		&prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification preferences: %w", err)
	}

	prefs.SubscribedEventsJSON = subscribedEvents.String
	prefs.QuietHoursStart = quietStart.String
	prefs.QuietHoursEnd = quietEnd.String
	prefs.ProjectFiltersJSON = projectFilters.String

	return prefs, nil
}

// UpsertNotificationPreferences inserts or updates notification preferences
func (d *Database) UpsertNotificationPreferences(prefs *NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			id, user_id, enable_in_app, enable_email, enable_webhook,
			subscribed_events_json, digest_mode, quiet_hours_start,
			quiet_hours_end, project_filters_json, min_priority, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			enable_in_app = excluded.enable_in_app,
			enable_email = excluded.enable_email,
			enable_webhook = excluded.enable_webhook,
			subscribed_events_json = excluded.subscribed_events_json,
			digest_mode = excluded.digest_mode,
			quiet_hours_start = excluded.quiet_hours_start,
			quiet_hours_end = excluded.quiet_hours_end,
			project_filters_json = excluded.project_filters_json,
			min_priority = excluded.min_priority,
			updated_at = excluded.updated_at
	`

	_, err := d.db.Exec(query,
		prefs.ID,
		prefs.UserID,
		prefs.EnableInApp,
		prefs.EnableEmail,
		prefs.EnableWebhook,
		sqlNullString(prefs.SubscribedEventsJSON),
		prefs.DigestMode,
		sqlNullString(prefs.QuietHoursStart),
		sqlNullString(prefs.QuietHoursEnd),
		sqlNullString(prefs.ProjectFiltersJSON),
		prefs.MinPriority,
		prefs.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert notification preferences: %w", err)
	}
	return nil
}

// User database operations
func (d *Database) CreateUser(id, username, email, role string) error {
	query := `
		INSERT INTO users (id, username, email, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, ?, ?)
	`

	now := time.Now()
	_, err := d.db.Exec(query, id, username, email, role, now, now)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (d *Database) ListUsers() ([]struct {
	ID       string
	Username string
	Email    string
	Role     string
}, error) {
	query := `SELECT id, username, email, role FROM users WHERE is_active = 1`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []struct {
		ID       string
		Username string
		Email    string
		Role     string
	}

	for rows.Next() {
		var u struct {
			ID       string
			Username string
			Email    string
			Role     string
		}
		var email sql.NullString
		err := rows.Scan(&u.ID, &u.Username, &email, &u.Role)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		u.Email = email.String
		users = append(users, u)
	}

	return users, nil
}

// Helper functions
func sqlNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func sqlNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
