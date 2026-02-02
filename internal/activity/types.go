package activity

import (
	"time"

	"github.com/jordanhubbard/agenticorp/internal/database"
)

// Activity represents an activity feed entry
type Activity struct {
	ID               string                 `json:"id"`
	EventType        string                 `json:"event_type"`
	EventID          string                 `json:"event_id,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
	Source           string                 `json:"source"`
	ActorID          string                 `json:"actor_id,omitempty"`
	ActorType        string                 `json:"actor_type,omitempty"`
	ProjectID        string                 `json:"project_id,omitempty"`
	AgentID          string                 `json:"agent_id,omitempty"`
	BeadID           string                 `json:"bead_id,omitempty"`
	ProviderID       string                 `json:"provider_id,omitempty"`
	Action           string                 `json:"action"`
	ResourceType     string                 `json:"resource_type"`
	ResourceID       string                 `json:"resource_id"`
	ResourceTitle    string                 `json:"resource_title,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	AggregationKey   string                 `json:"aggregation_key,omitempty"`
	AggregationCount int                    `json:"aggregation_count"`
	IsAggregated     bool                   `json:"is_aggregated"`
	Visibility       string                 `json:"visibility"`
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

// ToDBActivity converts Activity to database.Activity
func (a *Activity) ToDBActivity() *database.Activity {
	return &database.Activity{
		ID:               a.ID,
		EventType:        a.EventType,
		EventID:          a.EventID,
		Timestamp:        a.Timestamp,
		Source:           a.Source,
		ActorID:          a.ActorID,
		ActorType:        a.ActorType,
		ProjectID:        a.ProjectID,
		AgentID:          a.AgentID,
		BeadID:           a.BeadID,
		ProviderID:       a.ProviderID,
		Action:           a.Action,
		ResourceType:     a.ResourceType,
		ResourceID:       a.ResourceID,
		ResourceTitle:    a.ResourceTitle,
		AggregationKey:   a.AggregationKey,
		AggregationCount: a.AggregationCount,
		IsAggregated:     a.IsAggregated,
		Visibility:       a.Visibility,
	}
}

// FromDBActivity converts database.Activity to Activity
func FromDBActivity(dbActivity *database.Activity) *Activity {
	return &Activity{
		ID:               dbActivity.ID,
		EventType:        dbActivity.EventType,
		EventID:          dbActivity.EventID,
		Timestamp:        dbActivity.Timestamp,
		Source:           dbActivity.Source,
		ActorID:          dbActivity.ActorID,
		ActorType:        dbActivity.ActorType,
		ProjectID:        dbActivity.ProjectID,
		AgentID:          dbActivity.AgentID,
		BeadID:           dbActivity.BeadID,
		ProviderID:       dbActivity.ProviderID,
		Action:           dbActivity.Action,
		ResourceType:     dbActivity.ResourceType,
		ResourceID:       dbActivity.ResourceID,
		ResourceTitle:    dbActivity.ResourceTitle,
		AggregationKey:   dbActivity.AggregationKey,
		AggregationCount: dbActivity.AggregationCount,
		IsAggregated:     dbActivity.IsAggregated,
		Visibility:       dbActivity.Visibility,
	}
}
