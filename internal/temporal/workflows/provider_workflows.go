package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/jordanhubbard/agenticorp/internal/temporal/activities"
)

// ProviderHeartbeatWorkflowInput controls provider heartbeat scheduling.
type ProviderHeartbeatWorkflowInput struct {
	ProviderID string
	Interval   time.Duration
}

// ProviderHeartbeatWorkflow periodically checks provider health.
func ProviderHeartbeatWorkflow(ctx workflow.Context, input ProviderHeartbeatWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	if input.Interval == 0 {
		input.Interval = 30 * time.Second
	}

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	for {
		var result activities.ProviderHeartbeatResult
		err := workflow.ExecuteActivity(ctx, "ProviderHeartbeatActivity", activities.ProviderHeartbeatInput{ProviderID: input.ProviderID}).Get(ctx, &result)
		if err != nil {
			logger.Warn("Provider heartbeat failed", "providerID", input.ProviderID, "error", err)
		} else {
			logger.Info("Provider heartbeat", "providerID", input.ProviderID, "status", result.Status, "latency_ms", result.LatencyMs)
		}
		_ = workflow.Sleep(ctx, input.Interval)
	}
}

// ProviderQueryWorkflowInput controls a high-priority provider query.
type ProviderQueryWorkflowInput struct {
	ProviderID   string
	SystemPrompt string
	Message      string
	Temperature  float64
	MaxTokens    int
}

// ProviderQueryWorkflow runs a direct provider query through Temporal.
func ProviderQueryWorkflow(ctx workflow.Context, input ProviderQueryWorkflowInput) (activities.ProviderQueryResult, error) {
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var result activities.ProviderQueryResult
	err := workflow.ExecuteActivity(ctx, "ProviderQueryActivity", activities.ProviderQueryInput{
		ProviderID:   input.ProviderID,
		SystemPrompt: input.SystemPrompt,
		Message:      input.Message,
		Temperature:  input.Temperature,
		MaxTokens:    input.MaxTokens,
	}).Get(ctx, &result)

	return result, err
}
