package activity

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/agenticorp/internal/database"
	"github.com/jordanhubbard/agenticorp/internal/temporal/eventbus"
)

const (
	aggregationWindow = 5 * time.Minute
)

// Manager handles activity feed logic
type Manager struct {
	db               *database.Database
	eventBus         *eventbus.EventBus
	subscribers      map[string]chan *Activity
	subscribersMu    sync.RWMutex
	eventFilterSet   map[string]bool
	aggregationCache map[string]*Activity
	aggregationMu    sync.RWMutex
}

// NewManager creates a new activity manager
func NewManager(db *database.Database, eventBus *eventbus.EventBus) *Manager {
	m := &Manager{
		db:               db,
		eventBus:         eventBus,
		subscribers:      make(map[string]chan *Activity),
		eventFilterSet:   buildEventFilterSet(),
		aggregationCache: make(map[string]*Activity),
	}

	// Subscribe to EventBus
	if eventBus != nil {
		go m.subscribeToEvents()
	}

	return m
}

// buildEventFilterSet creates a set of events worth persisting
func buildEventFilterSet() map[string]bool {
	return map[string]bool{
		// Bead events
		"bead.created":       true,
		"bead.assigned":      true,
		"bead.status_change": true,
		"bead.completed":     true,

		// Agent events
		"agent.spawned":       true,
		"agent.status_change": true,
		"agent.completed":     true,

		// Project events
		"project.created": true,
		"project.updated": true,
		"project.deleted": true,

		// Provider events
		"provider.registered": true,
		"provider.deleted":    true,
		"provider.updated":    true,

		// Decision events
		"decision.created":  true,
		"decision.resolved": true,

		// Motivation events
		"motivation.fired":    true,
		"motivation.enabled":  true,
		"motivation.disabled": true,

		// Workflow events
		"workflow.started":   true,
		"workflow.completed": true,
		"workflow.failed":    true,
	}
}

// subscribeToEvents subscribes to the event bus
func (m *Manager) subscribeToEvents() {
	subscriber := m.eventBus.Subscribe("activity-manager", func(event *eventbus.Event) bool {
		// Filter events worth persisting
		return m.eventFilterSet[string(event.Type)]
	})

	for event := range subscriber.Channel {
		if err := m.RecordActivity(event); err != nil {
			log.Printf("Failed to record activity: %v", err)
		}
	}
}

// RecordActivity processes an event and records it as an activity
func (m *Manager) RecordActivity(event *eventbus.Event) error {
	activity := m.eventToActivity(event)
	if activity == nil {
		return nil
	}

	// Check if this event is aggregatable
	if activity.AggregationKey != "" {
		m.aggregationMu.Lock()
		defer m.aggregationMu.Unlock()

		// Check cache first
		if cached, exists := m.aggregationCache[activity.AggregationKey]; exists {
			// Check if within time window
			if time.Since(cached.Timestamp) < aggregationWindow {
				// Update aggregation count
				cached.AggregationCount++
				cached.IsAggregated = true

				// Update in database
				if err := m.db.UpdateAggregatedActivity(cached.ID, cached.AggregationCount); err != nil {
					return fmt.Errorf("failed to update aggregated activity: %w", err)
				}

				// Broadcast updated activity
				m.broadcastActivity(cached)
				return nil
			}
		}

		// Check database for recent aggregatable activity
		since := time.Now().Add(-aggregationWindow)
		existing, err := m.db.GetRecentAggregatableActivity(activity.AggregationKey, since)
		if err != nil {
			log.Printf("Failed to check for aggregatable activity: %v", err)
		}

		if existing != nil {
			// Update aggregation count
			existing.AggregationCount++

			if err := m.db.UpdateAggregatedActivity(existing.ID, existing.AggregationCount); err != nil {
				return fmt.Errorf("failed to update aggregated activity: %w", err)
			}

			// Update cache
			activityFromDB := FromDBActivity(existing)
			activityFromDB.AggregationCount = existing.AggregationCount
			m.aggregationCache[activity.AggregationKey] = activityFromDB

			// Broadcast updated activity
			m.broadcastActivity(activityFromDB)
			return nil
		}

		// Mark as aggregatable but first occurrence
		activity.IsAggregated = true
		m.aggregationCache[activity.AggregationKey] = activity
	}

	// Convert metadata to JSON
	dbActivity := activity.ToDBActivity()
	if activity.Metadata != nil {
		metadataJSON, err := json.Marshal(activity.Metadata)
		if err != nil {
			log.Printf("Failed to marshal activity metadata: %v", err)
		} else {
			dbActivity.MetadataJSON = string(metadataJSON)
		}
	}

	// Create new activity
	if err := m.db.CreateActivity(dbActivity); err != nil {
		return fmt.Errorf("failed to create activity: %w", err)
	}

	// Broadcast to subscribers
	m.broadcastActivity(activity)

	return nil
}

// eventToActivity converts an event to an activity
func (m *Manager) eventToActivity(event *eventbus.Event) *Activity {
	activity := &Activity{
		ID:        uuid.New().String(),
		EventType: string(event.Type),
		EventID:   event.ID,
		Timestamp: event.Timestamp,
		Source:    event.Source,
		ProjectID: event.ProjectID,
		Metadata:  event.Data,
	}

	// Extract common fields from event data
	if actorID, ok := event.Data["actor_id"].(string); ok {
		activity.ActorID = actorID
	}
	if actorType, ok := event.Data["actor_type"].(string); ok {
		activity.ActorType = actorType
	}
	if agentID, ok := event.Data["agent_id"].(string); ok {
		activity.AgentID = agentID
	}
	if beadID, ok := event.Data["bead_id"].(string); ok {
		activity.BeadID = beadID
	}

	// Extract resource information based on event type
	switch event.Type {
	case "bead.created", "bead.assigned", "bead.status_change", "bead.completed":
		activity.ResourceType = "bead"
		if beadID, ok := event.Data["bead_id"].(string); ok {
			activity.ResourceID = beadID
			activity.BeadID = beadID
		}
		activity.Action = extractAction(string(event.Type))
		if title, ok := event.Data["title"].(string); ok {
			activity.ResourceTitle = title
		}
		activity.Visibility = "project"
		activity.AggregationKey = buildAggregationKey(event, activity)

	case "agent.spawned", "agent.status_change", "agent.completed":
		activity.ResourceType = "agent"
		if agentID, ok := event.Data["agent_id"].(string); ok {
			activity.ResourceID = agentID
			activity.AgentID = agentID
		}
		activity.Action = extractAction(string(event.Type))
		if name, ok := event.Data["name"].(string); ok {
			activity.ResourceTitle = name
		}
		activity.Visibility = "project"

	case "project.created", "project.updated", "project.deleted":
		activity.ResourceType = "project"
		activity.ResourceID = event.ProjectID
		activity.Action = extractAction(string(event.Type))
		if name, ok := event.Data["name"].(string); ok {
			activity.ResourceTitle = name
		}
		activity.Visibility = "global"

	case "provider.registered", "provider.deleted", "provider.updated":
		activity.ResourceType = "provider"
		if providerID, ok := event.Data["provider_id"].(string); ok {
			activity.ResourceID = providerID
			activity.ProviderID = providerID
		}
		activity.Action = extractAction(string(event.Type))
		if name, ok := event.Data["name"].(string); ok {
			activity.ResourceTitle = name
		}
		activity.Visibility = "global"

	case "decision.created", "decision.resolved":
		activity.ResourceType = "decision"
		if decisionID, ok := event.Data["decision_id"].(string); ok {
			activity.ResourceID = decisionID
		}
		activity.Action = extractAction(string(event.Type))
		if title, ok := event.Data["title"].(string); ok {
			activity.ResourceTitle = title
		}
		activity.Visibility = "project"

	case "motivation.fired", "motivation.enabled", "motivation.disabled":
		activity.ResourceType = "motivation"
		if motivationID, ok := event.Data["motivation_id"].(string); ok {
			activity.ResourceID = motivationID
		}
		activity.Action = extractAction(string(event.Type))
		if name, ok := event.Data["name"].(string); ok {
			activity.ResourceTitle = name
		}
		activity.Visibility = "project"

	case "workflow.started", "workflow.completed", "workflow.failed":
		activity.ResourceType = "workflow"
		if workflowID, ok := event.Data["workflow_id"].(string); ok {
			activity.ResourceID = workflowID
		}
		activity.Action = extractAction(string(event.Type))
		if name, ok := event.Data["workflow_name"].(string); ok {
			activity.ResourceTitle = name
		}
		activity.Visibility = "project"

	default:
		// Unknown event type, skip
		return nil
	}

	return activity
}

// extractAction extracts the action from event type (e.g., "bead.created" -> "created")
func extractAction(eventType string) string {
	for i := len(eventType) - 1; i >= 0; i-- {
		if eventType[i] == '.' {
			return eventType[i+1:]
		}
	}
	return eventType
}

// buildAggregationKey builds a key for aggregating similar activities
func buildAggregationKey(event *eventbus.Event, activity *Activity) string {
	// Only aggregate certain event types
	switch event.Type {
	case "bead.created", "agent.spawned":
		// Format: {event_type}.{date}.{project_id}.{actor_id}
		date := event.Timestamp.Format("2006-01-02-15") // Group by hour
		return fmt.Sprintf("%s.%s.%s.%s", event.Type, date, event.ProjectID, activity.ActorID)
	default:
		return ""
	}
}

// GetActivities retrieves activities with filters
func (m *Manager) GetActivities(filters ActivityFilters) ([]*Activity, error) {
	dbFilters := database.ActivityFilters{
		ProjectIDs:   filters.ProjectIDs,
		EventType:    filters.EventType,
		ActorID:      filters.ActorID,
		ResourceType: filters.ResourceType,
		Since:        filters.Since,
		Until:        filters.Until,
		Limit:        filters.Limit,
		Offset:       filters.Offset,
		Aggregated:   filters.Aggregated,
	}

	dbActivities, err := m.db.ListActivities(dbFilters)
	if err != nil {
		return nil, err
	}

	activities := make([]*Activity, 0, len(dbActivities))
	for _, dbActivity := range dbActivities {
		activity := FromDBActivity(dbActivity)

		// Parse metadata JSON
		if dbActivity.MetadataJSON != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(dbActivity.MetadataJSON), &metadata); err == nil {
				activity.Metadata = metadata
			}
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

// Subscribe creates a new activity stream subscriber
func (m *Manager) Subscribe(subscriberID string) chan *Activity {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()

	ch := make(chan *Activity, 100)
	m.subscribers[subscriberID] = ch
	return ch
}

// Unsubscribe removes a subscriber
func (m *Manager) Unsubscribe(subscriberID string) {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()

	if ch, exists := m.subscribers[subscriberID]; exists {
		close(ch)
		delete(m.subscribers, subscriberID)
	}
}

// broadcastActivity sends an activity to all subscribers
func (m *Manager) broadcastActivity(activity *Activity) {
	m.subscribersMu.RLock()
	defer m.subscribersMu.RUnlock()

	for _, ch := range m.subscribers {
		select {
		case ch <- activity:
		default:
			// Channel full, skip
		}
	}
}
