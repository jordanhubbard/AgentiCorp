package activities

import (
	"context"

	"github.com/jordanhubbard/agenticorp/internal/database"
)

// AgentiCorpActivities supplies activities for the AgentiCorp heartbeat
type AgentiCorpActivities struct {
	database *database.Database
}

func NewAgentiCorpActivities(db *database.Database) *AgentiCorpActivities {
	return &AgentiCorpActivities{database: db}
}

// AgentiCorpHeartbeatActivity is the master clock activity
// It runs on every heartbeat to check if we should dispatch work or run idle tasks
func (a *AgentiCorpActivities) AgentiCorpHeartbeatActivity(ctx context.Context, beatCount int) error {
	// This is a placeholder activity that just logs the heartbeat
	// The real work dispatch happens via the dispatcher workflow
	// which is triggered separately during initialization
	if beatCount%10 == 0 {
		// Log every 10 beats (100 seconds at 10s interval)
		_ = ctx // Use ctx to satisfy linter
	}
	return nil
}
