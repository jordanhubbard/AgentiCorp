package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/loom/internal/temporal/eventbus"
)

// MessageType represents the type of agent message
type MessageType string

const (
	MessageTypeDirect           MessageType = "agent_message"
	MessageTypeBroadcast        MessageType = "broadcast"
	MessageTypeRequest          MessageType = "request"
	MessageTypeResponse         MessageType = "response"
	MessageTypeNotification     MessageType = "notification"
	MessageTypeConsensusRequest MessageType = "consensus_request"
	MessageTypeConsensusVote    MessageType = "consensus_vote"
)

// Priority levels for messages
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// AgentMessage represents a message between agents
type AgentMessage struct {
	MessageID        string                 `json:"message_id"`
	Type             MessageType            `json:"type"`
	FromAgentID      string                 `json:"from_agent_id"`
	ToAgentID        string                 `json:"to_agent_id,omitempty"` // Empty for broadcast
	ToAgentIDs       []string               `json:"to_agent_ids,omitempty"` // For consensus
	Subject          string                 `json:"subject,omitempty"`
	Body             string                 `json:"body,omitempty"`
	Payload          map[string]interface{} `json:"payload,omitempty"`
	Priority         Priority               `json:"priority"`
	RequiresResponse bool                   `json:"requires_response"`
	InReplyTo        string                 `json:"in_reply_to,omitempty"`
	Context          map[string]interface{} `json:"context,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
	Status           string                 `json:"status,omitempty"` // sent, delivered, read, failed
	DeliveredAt      *time.Time             `json:"delivered_at,omitempty"`
	ReadAt           *time.Time             `json:"read_at,omitempty"`
}

// MessageFilter defines subscription filters
type MessageFilter struct {
	MessageTypes []MessageType
	FromAgentIDs []string
	ToAgentID    string // Only messages to this agent
	Topics       []string
	MinPriority  Priority
}

// Subscription represents a message subscription
type Subscription struct {
	ID      string
	AgentID string
	Filter  MessageFilter
	Channel chan *AgentMessage
	cancel  context.CancelFunc
}

// AgentMessageBus handles agent-to-agent messaging
type AgentMessageBus struct {
	eventBus      *eventbus.EventBus
	subscriptions map[string]*Subscription
	history       map[string][]*AgentMessage // agent_id -> messages
	historyMu     sync.RWMutex
	subsMu        sync.RWMutex
	maxHistory    int
}

// NewAgentMessageBus creates a new agent message bus
func NewAgentMessageBus(eventBus *eventbus.EventBus) *AgentMessageBus {
	return &AgentMessageBus{
		eventBus:      eventBus,
		subscriptions: make(map[string]*Subscription),
		history:       make(map[string][]*AgentMessage),
		maxHistory:    1000, // Keep last 1000 messages per agent
	}
}

// Send sends a message from one agent to another
func (mb *AgentMessageBus) Send(ctx context.Context, msg *AgentMessage) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Validate required fields
	if msg.FromAgentID == "" {
		return fmt.Errorf("from_agent_id is required")
	}

	if msg.Type == MessageTypeDirect && msg.ToAgentID == "" {
		return fmt.Errorf("to_agent_id is required for direct messages")
	}

	if msg.Type == MessageTypeConsensusRequest && len(msg.ToAgentIDs) == 0 {
		return fmt.Errorf("to_agent_ids is required for consensus requests")
	}

	// Set defaults
	if msg.MessageID == "" {
		msg.MessageID = uuid.New().String()
	}

	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	if msg.Priority == "" {
		msg.Priority = PriorityNormal
	}

	msg.Status = "sent"

	// Store in history
	mb.addToHistory(msg)

	// Publish to event bus for distribution
	eventData := map[string]interface{}{
		"message": msg,
	}

	event := &eventbus.Event{
		Type:   eventbus.EventType("agent.message." + string(msg.Type)),
		Source: "agent-message-bus",
		Data:   eventData,
	}

	if msg.Type == MessageTypeDirect {
		event.Data["to_agent_id"] = msg.ToAgentID
	}

	if err := mb.eventBus.Publish(event); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// SendAndWait sends a request message and waits for a response
func (mb *AgentMessageBus) SendAndWait(ctx context.Context, msg *AgentMessage, timeout time.Duration) (*AgentMessage, error) {
	if msg.Type != MessageTypeRequest {
		return nil, fmt.Errorf("SendAndWait only works with request messages")
	}

	// Subscribe to responses
	respChan := make(chan *AgentMessage, 1)
	defer close(respChan)

	// Create a temporary subscription for the response
	subID := fmt.Sprintf("temp-response-%s", msg.MessageID)
	filter := MessageFilter{
		MessageTypes: []MessageType{MessageTypeResponse},
		FromAgentIDs: []string{msg.ToAgentID},
	}

	sub := mb.Subscribe(subID, msg.FromAgentID, filter)
	defer mb.Unsubscribe(subID)

	// Send the request
	if err := mb.Send(ctx, msg); err != nil {
		return nil, err
	}

	// Wait for response or timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("request timeout after %v", timeout)
		case response := <-sub.Channel:
			if response != nil && response.InReplyTo == msg.MessageID {
				return response, nil
			}
		}
	}
}

// Subscribe creates a subscription to receive messages
func (mb *AgentMessageBus) Subscribe(subscriptionID, agentID string, filter MessageFilter) *Subscription {
	mb.subsMu.Lock()
	defer mb.subsMu.Unlock()

	// Check if subscription already exists
	if sub, exists := mb.subscriptions[subscriptionID]; exists {
		return sub
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create new subscription
	sub := &Subscription{
		ID:      subscriptionID,
		AgentID: agentID,
		Filter:  filter,
		Channel: make(chan *AgentMessage, 100),
		cancel:  cancel,
	}

	mb.subscriptions[subscriptionID] = sub

	// Subscribe to event bus with filter
	eventFilter := func(event *eventbus.Event) bool {
		// Only process agent message events
		if event.Type != "agent.message.agent_message" &&
			event.Type != "agent.message.broadcast" &&
			event.Type != "agent.message.request" &&
			event.Type != "agent.message.response" &&
			event.Type != "agent.message.notification" &&
			event.Type != "agent.message.consensus_request" &&
			event.Type != "agent.message.consensus_vote" {
			return false
		}

		// Extract message from event
		msgData, ok := event.Data["message"]
		if !ok {
			return false
		}

		// Convert to AgentMessage
		msgBytes, err := json.Marshal(msgData)
		if err != nil {
			return false
		}

		var msg AgentMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			return false
		}

		// Apply subscription filter
		return mb.matchesFilter(&msg, filter, agentID)
	}

	// Subscribe to event bus
	eventSub := mb.eventBus.Subscribe(subscriptionID, eventFilter)

	// Forward events to subscription channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventSub.Channel:
				if !ok {
					return
				}

				// Extract and forward message
				if msgData, ok := event.Data["message"]; ok {
					msgBytes, _ := json.Marshal(msgData)
					var msg AgentMessage
					if json.Unmarshal(msgBytes, &msg) == nil {
						// Update delivery status
						now := time.Now()
						msg.DeliveredAt = &now
						msg.Status = "delivered"

						select {
						case sub.Channel <- &msg:
						default:
							// Subscription channel full, drop message
						}
					}
				}
			}
		}
	}()

	return sub
}

// Unsubscribe removes a subscription
func (mb *AgentMessageBus) Unsubscribe(subscriptionID string) {
	mb.subsMu.Lock()
	defer mb.subsMu.Unlock()

	if sub, exists := mb.subscriptions[subscriptionID]; exists {
		sub.cancel()
		mb.eventBus.Unsubscribe(subscriptionID)
		close(sub.Channel)
		delete(mb.subscriptions, subscriptionID)
	}
}

// GetHistory returns message history for an agent
func (mb *AgentMessageBus) GetHistory(agentID string, limit int) []*AgentMessage {
	mb.historyMu.RLock()
	defer mb.historyMu.RUnlock()

	messages, exists := mb.history[agentID]
	if !exists {
		return []*AgentMessage{}
	}

	if limit <= 0 || limit > len(messages) {
		limit = len(messages)
	}

	// Return most recent messages
	start := len(messages) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*AgentMessage, limit)
	copy(result, messages[start:])
	return result
}

// addToHistory adds a message to agent history
func (mb *AgentMessageBus) addToHistory(msg *AgentMessage) {
	mb.historyMu.Lock()
	defer mb.historyMu.Unlock()

	// Add to sender's history
	mb.appendHistory(msg.FromAgentID, msg)

	// Add to receiver's history
	if msg.ToAgentID != "" {
		mb.appendHistory(msg.ToAgentID, msg)
	}

	// Add to all recipients for consensus
	for _, agentID := range msg.ToAgentIDs {
		mb.appendHistory(agentID, msg)
	}
}

// appendHistory appends a message to an agent's history with size limit
func (mb *AgentMessageBus) appendHistory(agentID string, msg *AgentMessage) {
	history := mb.history[agentID]
	history = append(history, msg)

	// Trim if exceeds max
	if len(history) > mb.maxHistory {
		history = history[len(history)-mb.maxHistory:]
	}

	mb.history[agentID] = history
}

// matchesFilter checks if a message matches subscription filter
func (mb *AgentMessageBus) matchesFilter(msg *AgentMessage, filter MessageFilter, subscriberAgentID string) bool {
	// Check if message is for this agent
	if filter.ToAgentID != "" {
		if msg.ToAgentID != subscriberAgentID {
			// Also check consensus recipients
			found := false
			for _, id := range msg.ToAgentIDs {
				if id == subscriberAgentID {
					found = true
					break
				}
			}
			if !found && msg.Type != MessageTypeBroadcast {
				return false
			}
		}
	}

	// Check message type filter
	if len(filter.MessageTypes) > 0 {
		typeMatch := false
		for _, mt := range filter.MessageTypes {
			if msg.Type == mt {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	// Check from agent filter
	if len(filter.FromAgentIDs) > 0 {
		fromMatch := false
		for _, id := range filter.FromAgentIDs {
			if msg.FromAgentID == id {
				fromMatch = true
				break
			}
		}
		if !fromMatch {
			return false
		}
	}

	// Check priority filter
	if filter.MinPriority != "" {
		priorityOrder := map[Priority]int{
			PriorityLow:    0,
			PriorityNormal: 1,
			PriorityHigh:   2,
			PriorityUrgent: 3,
		}

		minLevel := priorityOrder[filter.MinPriority]
		msgLevel := priorityOrder[msg.Priority]

		if msgLevel < minLevel {
			return false
		}
	}

	return true
}

// Close shuts down the message bus
func (mb *AgentMessageBus) Close() {
	mb.subsMu.Lock()
	defer mb.subsMu.Unlock()

	for id := range mb.subscriptions {
		mb.Unsubscribe(id)
	}
}
