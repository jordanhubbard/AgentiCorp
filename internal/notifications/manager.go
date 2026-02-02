package notifications

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/agenticorp/internal/activity"
	"github.com/jordanhubbard/agenticorp/internal/database"
)

// Manager handles notification logic
type Manager struct {
	db            *database.Database
	activityMgr   *activity.Manager
	subscribers   map[string]map[string]chan *Notification // userID -> subscriberID -> channel
	subscribersMu sync.RWMutex
}

// NewManager creates a new notification manager
func NewManager(db *database.Database, activityMgr *activity.Manager) *Manager {
	m := &Manager{
		db:          db,
		activityMgr: activityMgr,
		subscribers: make(map[string]map[string]chan *Notification),
	}

	// Subscribe to activity manager
	go m.subscribeToActivities()

	return m
}

// subscribeToActivities subscribes to activity feed
func (m *Manager) subscribeToActivities() {
	activityChan := m.activityMgr.Subscribe("notification-manager")

	for activity := range activityChan {
		if err := m.ProcessActivity(activity); err != nil {
			log.Printf("Failed to process activity for notifications: %v", err)
		}
	}
}

// ProcessActivity processes an activity and creates notifications
func (m *Manager) ProcessActivity(activity *activity.Activity) error {
	// Get all users from database
	users, err := m.db.ListUsers()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	for _, user := range users {
		// Check if user should be notified
		shouldNotify, notification := m.ShouldNotify(activity, user.ID)
		if !shouldNotify {
			continue
		}

		// Get user preferences
		prefs, err := m.GetPreferences(user.ID)
		if err != nil {
			log.Printf("Failed to get preferences for user %s: %v", user.ID, err)
			continue
		}

		// Check if in-app notifications are enabled
		if !prefs.EnableInApp {
			continue
		}

		// Create notification
		if err := m.CreateNotification(notification); err != nil {
			log.Printf("Failed to create notification for user %s: %v", user.ID, err)
			continue
		}

		// Broadcast to user's SSE streams
		m.broadcastToUser(user.ID, notification)
	}

	return nil
}

// ShouldNotify determines if a user should be notified about an activity
func (m *Manager) ShouldNotify(activity *activity.Activity, userID string) (bool, *Notification) {
	// Get user preferences
	prefs, err := m.GetPreferences(userID)
	if err != nil {
		return false, nil
	}

	// Check if event type is subscribed
	if !m.isEventSubscribed(activity.EventType, prefs.SubscribedEvents) {
		return false, nil
	}

	// Check quiet hours
	if m.inQuietHours(prefs) {
		return false, nil
	}

	// Apply notification rules
	priority := m.determinePriority(activity)

	// Check priority threshold
	if !m.meetsPriorityThreshold(priority, prefs.MinPriority) {
		return false, nil
	}

	// Apply specific rules
	title, message, link := m.formatNotification(activity, userID)
	if title == "" {
		return false, nil
	}

	notification := &Notification{
		ID:         uuid.New().String(),
		UserID:     userID,
		ActivityID: activity.ID,
		EventType:  activity.EventType,
		Title:      title,
		Message:    message,
		Link:       link,
		Status:     StatusUnread,
		Priority:   priority,
		CreatedAt:  time.Now(),
	}

	return true, notification
}

// formatNotification formats a notification based on activity and user
func (m *Manager) formatNotification(activity *activity.Activity, userID string) (title, message, link string) {
	// Check for direct assignment
	if activity.EventType == "bead.assigned" {
		if assignedTo, ok := activity.Metadata["assigned_to"].(string); ok && assignedTo == userID {
			title = "Bead Assigned to You"
			message = fmt.Sprintf("You've been assigned to bead: %s", activity.ResourceTitle)
			link = fmt.Sprintf("/beads/%s", activity.ResourceID)
			return
		}
		return "", "", ""
	}

	// Check for decision requiring user input
	if activity.EventType == "decision.created" {
		if deciderID, ok := activity.Metadata["decider_id"].(string); ok && deciderID == userID {
			title = "Decision Requires Your Input"
			message = fmt.Sprintf("A decision needs your attention: %s", activity.ResourceTitle)
			link = fmt.Sprintf("/decisions/%s", activity.ResourceID)
			return
		}
		return "", "", ""
	}

	// Check for critical priority beads
	if activity.EventType == "bead.created" {
		if priority, ok := activity.Metadata["priority"].(string); ok && priority == "P0" {
			title = "Critical Bead Created"
			message = fmt.Sprintf("A P0 bead was created: %s", activity.ResourceTitle)
			link = fmt.Sprintf("/beads/%s", activity.ResourceID)
			return
		}
	}

	// Check for system errors
	if activity.EventType == "provider.deleted" || activity.EventType == "workflow.failed" {
		title = "System Alert"
		message = fmt.Sprintf("%s: %s", activity.Action, activity.ResourceTitle)
		link = fmt.Sprintf("/%ss/%s", activity.ResourceType, activity.ResourceID)
		return
	}

	return "", "", ""
}

// determinePriority determines notification priority based on activity
func (m *Manager) determinePriority(activity *activity.Activity) string {
	// Check metadata for explicit priority
	if priority, ok := activity.Metadata["priority"].(string); ok {
		switch priority {
		case "P0":
			return PriorityCritical
		case "P1":
			return PriorityHigh
		case "P2":
			return PriorityNormal
		default:
			return PriorityLow
		}
	}

	// Determine priority based on event type
	switch activity.EventType {
	case "bead.assigned", "decision.created":
		return PriorityHigh
	case "workflow.failed", "provider.deleted":
		return PriorityCritical
	case "bead.created", "agent.spawned":
		return PriorityNormal
	default:
		return PriorityLow
	}
}

// isEventSubscribed checks if an event type is in the subscribed list
func (m *Manager) isEventSubscribed(eventType string, subscribedEvents []string) bool {
	// If no specific events are subscribed, subscribe to all
	if len(subscribedEvents) == 0 {
		return true
	}

	for _, e := range subscribedEvents {
		if e == eventType {
			return true
		}
	}
	return false
}

// inQuietHours checks if current time is in quiet hours
func (m *Manager) inQuietHours(prefs *NotificationPreferences) bool {
	if prefs.QuietHoursStart == "" || prefs.QuietHoursEnd == "" {
		return false
	}

	// Parse quiet hours
	start, err := time.Parse("15:04", prefs.QuietHoursStart)
	if err != nil {
		return false
	}

	end, err := time.Parse("15:04", prefs.QuietHoursEnd)
	if err != nil {
		return false
	}

	// Get current time (hours and minutes only)
	now := time.Now()
	currentTime := time.Date(0, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)

	// Handle quiet hours spanning midnight
	if start.Before(end) {
		return currentTime.After(start) && currentTime.Before(end)
	} else {
		return currentTime.After(start) || currentTime.Before(end)
	}
}

// meetsPriorityThreshold checks if notification priority meets user's threshold
func (m *Manager) meetsPriorityThreshold(notificationPriority, minPriority string) bool {
	priorities := map[string]int{
		PriorityLow:      0,
		PriorityNormal:   1,
		PriorityHigh:     2,
		PriorityCritical: 3,
	}

	notifLevel := priorities[notificationPriority]
	minLevel := priorities[minPriority]

	return notifLevel >= minLevel
}

// CreateNotification creates a new notification
func (m *Manager) CreateNotification(notification *Notification) error {
	// Convert metadata to JSON
	var metadataJSON string
	if notification.Metadata != nil {
		data, err := json.Marshal(notification.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(data)
	}

	dbNotification := &database.Notification{
		ID:           notification.ID,
		UserID:       notification.UserID,
		ActivityID:   notification.ActivityID,
		EventType:    notification.EventType,
		Title:        notification.Title,
		Message:      notification.Message,
		Link:         notification.Link,
		Status:       notification.Status,
		Priority:     notification.Priority,
		MetadataJSON: metadataJSON,
		CreatedAt:    notification.CreatedAt,
		ReadAt:       notification.ReadAt,
		ArchivedAt:   notification.ArchivedAt,
	}

	return m.db.CreateNotification(dbNotification)
}

// GetNotifications retrieves notifications for a user
func (m *Manager) GetNotifications(userID string, status string, limit, offset int) ([]*Notification, error) {
	dbNotifications, err := m.db.ListNotifications(userID, status, limit, offset)
	if err != nil {
		return nil, err
	}

	notifications := make([]*Notification, 0, len(dbNotifications))
	for _, dbNotif := range dbNotifications {
		notification := &Notification{
			ID:         dbNotif.ID,
			UserID:     dbNotif.UserID,
			ActivityID: dbNotif.ActivityID,
			EventType:  dbNotif.EventType,
			Title:      dbNotif.Title,
			Message:    dbNotif.Message,
			Link:       dbNotif.Link,
			Status:     dbNotif.Status,
			Priority:   dbNotif.Priority,
			CreatedAt:  dbNotif.CreatedAt,
			ReadAt:     dbNotif.ReadAt,
			ArchivedAt: dbNotif.ArchivedAt,
		}

		// Parse metadata JSON
		if dbNotif.MetadataJSON != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(dbNotif.MetadataJSON), &metadata); err == nil {
				notification.Metadata = metadata
			}
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// MarkRead marks a notification as read
func (m *Manager) MarkRead(notificationID string) error {
	return m.db.MarkNotificationRead(notificationID)
}

// MarkAllRead marks all unread notifications as read for a user
func (m *Manager) MarkAllRead(userID string) error {
	return m.db.MarkAllNotificationsRead(userID)
}

// GetPreferences retrieves notification preferences for a user
func (m *Manager) GetPreferences(userID string) (*NotificationPreferences, error) {
	dbPrefs, err := m.db.GetNotificationPreferences(userID)
	if err != nil {
		return nil, err
	}

	// Return default preferences if none exist
	if dbPrefs == nil {
		return m.createDefaultPreferences(userID)
	}

	prefs := &NotificationPreferences{
		ID:              dbPrefs.ID,
		UserID:          dbPrefs.UserID,
		EnableInApp:     dbPrefs.EnableInApp,
		EnableEmail:     dbPrefs.EnableEmail,
		EnableWebhook:   dbPrefs.EnableWebhook,
		DigestMode:      dbPrefs.DigestMode,
		QuietHoursStart: dbPrefs.QuietHoursStart,
		QuietHoursEnd:   dbPrefs.QuietHoursEnd,
		MinPriority:     dbPrefs.MinPriority,
		UpdatedAt:       dbPrefs.UpdatedAt,
	}

	// Parse JSON fields
	if dbPrefs.SubscribedEventsJSON != "" {
		var events []string
		if err := json.Unmarshal([]byte(dbPrefs.SubscribedEventsJSON), &events); err == nil {
			prefs.SubscribedEvents = events
		}
	}

	if dbPrefs.ProjectFiltersJSON != "" {
		var projects []string
		if err := json.Unmarshal([]byte(dbPrefs.ProjectFiltersJSON), &projects); err == nil {
			prefs.ProjectFilters = projects
		}
	}

	return prefs, nil
}

// createDefaultPreferences creates default preferences for a user
func (m *Manager) createDefaultPreferences(userID string) (*NotificationPreferences, error) {
	prefs := &NotificationPreferences{
		ID:               uuid.New().String(),
		UserID:           userID,
		EnableInApp:      true,
		EnableEmail:      false,
		EnableWebhook:    false,
		SubscribedEvents: []string{}, // Subscribe to all by default
		DigestMode:       DigestRealtime,
		MinPriority:      PriorityNormal,
		UpdatedAt:        time.Now(),
	}

	// Save to database
	if err := m.UpdatePreferences(prefs); err != nil {
		return nil, err
	}

	return prefs, nil
}

// UpdatePreferences updates notification preferences
func (m *Manager) UpdatePreferences(prefs *NotificationPreferences) error {
	// Convert to DB format
	var subscribedEventsJSON, projectFiltersJSON string

	if len(prefs.SubscribedEvents) > 0 {
		data, err := json.Marshal(prefs.SubscribedEvents)
		if err != nil {
			return fmt.Errorf("failed to marshal subscribed events: %w", err)
		}
		subscribedEventsJSON = string(data)
	}

	if len(prefs.ProjectFilters) > 0 {
		data, err := json.Marshal(prefs.ProjectFilters)
		if err != nil {
			return fmt.Errorf("failed to marshal project filters: %w", err)
		}
		projectFiltersJSON = string(data)
	}

	prefs.UpdatedAt = time.Now()

	dbPrefs := &database.NotificationPreferences{
		ID:                   prefs.ID,
		UserID:               prefs.UserID,
		EnableInApp:          prefs.EnableInApp,
		EnableEmail:          prefs.EnableEmail,
		EnableWebhook:        prefs.EnableWebhook,
		SubscribedEventsJSON: subscribedEventsJSON,
		DigestMode:           prefs.DigestMode,
		QuietHoursStart:      prefs.QuietHoursStart,
		QuietHoursEnd:        prefs.QuietHoursEnd,
		ProjectFiltersJSON:   projectFiltersJSON,
		MinPriority:          prefs.MinPriority,
		UpdatedAt:            prefs.UpdatedAt,
	}

	return m.db.UpsertNotificationPreferences(dbPrefs)
}

// Subscribe creates a new notification stream subscriber for a user
func (m *Manager) Subscribe(userID, subscriberID string) chan *Notification {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()

	if m.subscribers[userID] == nil {
		m.subscribers[userID] = make(map[string]chan *Notification)
	}

	ch := make(chan *Notification, 100)
	m.subscribers[userID][subscriberID] = ch
	return ch
}

// Unsubscribe removes a subscriber
func (m *Manager) Unsubscribe(userID, subscriberID string) {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()

	if userSubs, exists := m.subscribers[userID]; exists {
		if ch, exists := userSubs[subscriberID]; exists {
			close(ch)
			delete(userSubs, subscriberID)
		}

		// Clean up empty user map
		if len(userSubs) == 0 {
			delete(m.subscribers, userID)
		}
	}
}

// broadcastToUser sends a notification to all of a user's subscribers
func (m *Manager) broadcastToUser(userID string, notification *Notification) {
	m.subscribersMu.RLock()
	defer m.subscribersMu.RUnlock()

	if userSubs, exists := m.subscribers[userID]; exists {
		for _, ch := range userSubs {
			select {
			case ch <- notification:
			default:
				// Channel full, skip
			}
		}
	}
}
