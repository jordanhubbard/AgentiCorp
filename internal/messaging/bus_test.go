package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/jordanhubbard/loom/internal/temporal/eventbus"
	"github.com/jordanhubbard/loom/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestBus(t *testing.T) *AgentMessageBus {
	t.Helper()

	cfg := &config.TemporalConfig{
		EventBufferSize: 100,
	}
	eb := eventbus.NewEventBus(nil, cfg)

	return NewAgentMessageBus(eb)
}

func TestNewAgentMessageBus(t *testing.T) {
	bus := setupTestBus(t)
	assert.NotNil(t, bus)
	assert.Equal(t, 1000, bus.maxHistory)
}

func TestSend_DirectMessage(t *testing.T) {
	bus := setupTestBus(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Subject:     "Test message",
		Body:        "Hello from agent-1",
		Priority:    PriorityNormal,
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	// Verify message ID was generated
	assert.NotEmpty(t, msg.MessageID)
	assert.Equal(t, "sent", msg.Status)

	// Verify message in history
	history := bus.GetHistory("agent-1", 10)
	assert.Len(t, history, 1)
	assert.Equal(t, msg.MessageID, history[0].MessageID)
}

func TestSend_Broadcast(t *testing.T) {
	bus := setupTestBus(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeBroadcast,
		FromAgentID: "agent-1",
		Subject:     "Broadcast message",
		Body:        "Hello everyone",
		Priority:    PriorityHigh,
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)
	assert.NotEmpty(t, msg.MessageID)
}

func TestSend_ValidationErrors(t *testing.T) {
	bus := setupTestBus(t)
	defer bus.Close()

	tests := []struct {
		name    string
		msg     *AgentMessage
		wantErr string
	}{
		{
			name:    "nil message",
			msg:     nil,
			wantErr: "message cannot be nil",
		},
		{
			name: "missing from_agent_id",
			msg: &AgentMessage{
				Type:      MessageTypeDirect,
				ToAgentID: "agent-2",
			},
			wantErr: "from_agent_id is required",
		},
		{
			name: "direct message missing to_agent_id",
			msg: &AgentMessage{
				Type:        MessageTypeDirect,
				FromAgentID: "agent-1",
			},
			wantErr: "to_agent_id is required",
		},
		{
			name: "consensus missing to_agent_ids",
			msg: &AgentMessage{
				Type:        MessageTypeConsensusRequest,
				FromAgentID: "agent-1",
			},
			wantErr: "to_agent_ids is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bus.Send(context.Background(), tt.msg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSubscribeAndReceive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	bus := setupTestBus(t)
	defer bus.Close()

	// Create subscription for agent-2
	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
		ToAgentID:    "agent-2",
	}
	sub := bus.Subscribe("sub-1", "agent-2", filter)
	assert.NotNil(t, sub)

	// Give subscription time to set up
	time.Sleep(100 * time.Millisecond)

	// Send message to agent-2
	msg := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Subject:     "Test",
		Body:        "Hello",
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	// Wait for message
	select {
	case received := <-sub.Channel:
		assert.Equal(t, msg.MessageID, received.MessageID)
		assert.Equal(t, "agent-1", received.FromAgentID)
		assert.Equal(t, "agent-2", received.ToAgentID)
		assert.Equal(t, "delivered", received.Status)
		assert.NotNil(t, received.DeliveredAt)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestSubscribeWithFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	bus := setupTestBus(t)
	defer bus.Close()

	// Subscribe only to high priority messages
	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
		MinPriority:  PriorityHigh,
	}
	sub := bus.Subscribe("sub-1", "agent-2", filter)

	time.Sleep(100 * time.Millisecond)

	// Send normal priority message (should be filtered out)
	msg1 := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Subject:     "Normal",
		Priority:    PriorityNormal,
	}
	err := bus.Send(context.Background(), msg1)
	require.NoError(t, err)

	// Send high priority message (should be received)
	msg2 := &AgentMessage{
		Type:        MessageTypeDirect,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Subject:     "High",
		Priority:    PriorityHigh,
	}
	err = bus.Send(context.Background(), msg2)
	require.NoError(t, err)

	// Should only receive high priority message
	select {
	case received := <-sub.Channel:
		assert.Equal(t, msg2.MessageID, received.MessageID)
		assert.Equal(t, PriorityHigh, received.Priority)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for high priority message")
	}

	// Verify no other messages received
	select {
	case <-sub.Channel:
		t.Fatal("Should not have received normal priority message")
	case <-time.After(500 * time.Millisecond):
		// Expected - no message received
	}
}

func TestSendAndWait(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	bus := setupTestBus(t)
	defer bus.Close()

	// Set up agent-2 to receive and respond to requests
	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeRequest},
	}
	sub := bus.Subscribe("agent-2-sub", "agent-2", filter)

	go func() {
		select {
		case req := <-sub.Channel:
			// Send response
			resp := &AgentMessage{
				Type:        MessageTypeResponse,
				FromAgentID: "agent-2",
				ToAgentID:   req.FromAgentID,
				InReplyTo:   req.MessageID,
				Payload: map[string]interface{}{
					"status": "success",
					"result": "test completed",
				},
			}
			_ = bus.Send(context.Background(), resp)
		case <-time.After(5 * time.Second):
			return
		}
	}()

	// Send request and wait for response
	request := &AgentMessage{
		Type:        MessageTypeRequest,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Subject:     "Run test",
		Payload: map[string]interface{}{
			"test_name": "TestAuth",
		},
	}

	response, err := bus.SendAndWait(context.Background(), request, 5*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, request.MessageID, response.InReplyTo)
	assert.Equal(t, "agent-2", response.FromAgentID)

	status, ok := response.Payload["status"].(string)
	assert.True(t, ok)
	assert.Equal(t, "success", status)
}

func TestSendAndWait_Timeout(t *testing.T) {
	bus := setupTestBus(t)
	defer bus.Close()

	request := &AgentMessage{
		Type:        MessageTypeRequest,
		FromAgentID: "agent-1",
		ToAgentID:   "agent-2",
		Subject:     "Run test",
	}

	_, err := bus.SendAndWait(context.Background(), request, 500*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestGetHistory(t *testing.T) {
	bus := setupTestBus(t)
	defer bus.Close()

	// Send several messages
	for i := 0; i < 5; i++ {
		msg := &AgentMessage{
			Type:        MessageTypeDirect,
			FromAgentID: "agent-1",
			ToAgentID:   "agent-2",
			Subject:     "Message",
			Body:        string(rune('A' + i)),
		}
		err := bus.Send(context.Background(), msg)
		require.NoError(t, err)
	}

	// Get history for agent-1 (sender)
	history := bus.GetHistory("agent-1", 10)
	assert.Len(t, history, 5)

	// Get history for agent-2 (receiver)
	history = bus.GetHistory("agent-2", 10)
	assert.Len(t, history, 5)

	// Test limit
	history = bus.GetHistory("agent-1", 3)
	assert.Len(t, history, 3)
	// Should get most recent 3
	assert.Equal(t, "C", history[0].Body)
	assert.Equal(t, "D", history[1].Body)
	assert.Equal(t, "E", history[2].Body)
}

func TestHistoryLimit(t *testing.T) {
	bus := setupTestBus(t)
	defer bus.Close()
	bus.maxHistory = 10 // Set low limit for testing

	// Send more messages than history limit
	for i := 0; i < 15; i++ {
		msg := &AgentMessage{
			Type:        MessageTypeDirect,
			FromAgentID: "agent-1",
			ToAgentID:   "agent-2",
			Subject:     "Message",
		}
		err := bus.Send(context.Background(), msg)
		require.NoError(t, err)
	}

	// History should be limited
	history := bus.GetHistory("agent-1", 100)
	assert.LessOrEqual(t, len(history), 10)
}

func TestUnsubscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	bus := setupTestBus(t)
	defer bus.Close()

	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeDirect},
	}
	sub := bus.Subscribe("sub-1", "agent-2", filter)

	// Unsubscribe
	bus.Unsubscribe("sub-1")

	// Channel should be closed
	_, ok := <-sub.Channel
	assert.False(t, ok, "Channel should be closed")
}

func TestConsensusMessage(t *testing.T) {
	bus := setupTestBus(t)
	defer bus.Close()

	msg := &AgentMessage{
		Type:        MessageTypeConsensusRequest,
		FromAgentID: "agent-pm",
		ToAgentIDs:  []string{"agent-1", "agent-2", "agent-3"},
		Subject:     "Approve refactoring?",
		Payload: map[string]interface{}{
			"question": "Should we refactor auth module?",
			"options":  []string{"yes", "no", "defer"},
		},
	}

	err := bus.Send(context.Background(), msg)
	require.NoError(t, err)

	// Verify all recipients have it in history
	for _, agentID := range msg.ToAgentIDs {
		history := bus.GetHistory(agentID, 10)
		assert.Len(t, history, 1)
		assert.Equal(t, msg.MessageID, history[0].MessageID)
	}
}

func TestMatchesFilter(t *testing.T) {
	bus := setupTestBus(t)

	tests := []struct {
		name    string
		msg     *AgentMessage
		filter  MessageFilter
		agentID string
		want    bool
	}{
		{
			name: "direct message matches",
			msg: &AgentMessage{
				Type:        MessageTypeDirect,
				FromAgentID: "agent-1",
				ToAgentID:   "agent-2",
				Priority:    PriorityNormal,
			},
			filter: MessageFilter{
				MessageTypes: []MessageType{MessageTypeDirect},
				ToAgentID:    "agent-2",
			},
			agentID: "agent-2",
			want:    true,
		},
		{
			name: "wrong recipient",
			msg: &AgentMessage{
				Type:        MessageTypeDirect,
				FromAgentID: "agent-1",
				ToAgentID:   "agent-2",
			},
			filter: MessageFilter{
				ToAgentID: "agent-3",
			},
			agentID: "agent-3",
			want:    false,
		},
		{
			name: "priority filter",
			msg: &AgentMessage{
				Type:     MessageTypeDirect,
				Priority: PriorityNormal,
			},
			filter: MessageFilter{
				MinPriority: PriorityHigh,
			},
			agentID: "agent-2",
			want:    false,
		},
		{
			name: "broadcast always matches",
			msg: &AgentMessage{
				Type:        MessageTypeBroadcast,
				FromAgentID: "agent-1",
			},
			filter: MessageFilter{
				ToAgentID: "agent-2",
			},
			agentID: "agent-2",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bus.matchesFilter(tt.msg, tt.filter, tt.agentID)
			assert.Equal(t, tt.want, got)
		})
	}
}
