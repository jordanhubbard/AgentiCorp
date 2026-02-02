package notifications

import (
	"time"
)

// Notification represents a user notification
type Notification struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	ActivityID string                 `json:"activity_id,omitempty"`
	EventType  string                 `json:"event_type"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Link       string                 `json:"link,omitempty"`
	Status     string                 `json:"status"`
	Priority   string                 `json:"priority"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	ReadAt     *time.Time             `json:"read_at,omitempty"`
	ArchivedAt *time.Time             `json:"archived_at,omitempty"`
}

// NotificationPreferences represents user notification preferences
type NotificationPreferences struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	EnableInApp      bool     `json:"enable_in_app"`
	EnableEmail      bool     `json:"enable_email"`
	EnableWebhook    bool     `json:"enable_webhook"`
	SubscribedEvents []string `json:"subscribed_events"`
	DigestMode       string   `json:"digest_mode"`
	QuietHoursStart  string   `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd    string   `json:"quiet_hours_end,omitempty"`
	ProjectFilters   []string `json:"project_filters,omitempty"`
	MinPriority      string   `json:"min_priority"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Priority levels
const (
	PriorityLow      = "low"
	PriorityNormal   = "normal"
	PriorityHigh     = "high"
	PriorityCritical = "critical"
)

// Status values
const (
	StatusUnread   = "unread"
	StatusRead     = "read"
	StatusArchived = "archived"
)

// Digest modes
const (
	DigestRealtime = "realtime"
	DigestHourly   = "hourly"
	DigestDaily    = "daily"
)
