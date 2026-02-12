package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/loom/internal/actions"
	"github.com/jordanhubbard/loom/internal/database"
	"github.com/jordanhubbard/loom/internal/provider"
	"github.com/jordanhubbard/loom/pkg/models"
)

// PairChatRequest represents a request for pair-programming chat
type PairChatRequest struct {
	AgentID string `json:"agent_id"`
	BeadID  string `json:"bead_id"`
	Message string `json:"message"`
}

// handlePairChat handles pair-programming chat with streaming response
// POST /api/v1/pair
func (s *Server) handlePairChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request
	var req PairChatRequest
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AgentID == "" || req.BeadID == "" || req.Message == "" {
		s.respondError(w, http.StatusBadRequest, "agent_id, bead_id, and message are required")
		return
	}

	// Get database first — we query agent info from DB to avoid agent manager lock contention
	db := s.app.GetDatabase()
	if db == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}

	// Look up agent from DB directly (avoids agent manager mutex which can block during dispatch)
	agent, err := lookupAgentFromDB(db, req.AgentID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, fmt.Sprintf("Agent not found: %s", req.AgentID))
		return
	}

	// Get provider for this agent
	providerReg := s.app.GetProviderRegistry()
	if providerReg == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Provider registry not available")
		return
	}

	providerID := agent.ProviderID
	if providerID == "" {
		s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Agent %s has no provider assigned", req.AgentID))
		return
	}

	registeredProvider, err := providerReg.Get(providerID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, fmt.Sprintf("Provider not found: %s", providerID))
		return
	}

	// Check streaming support
	_, ok := registeredProvider.Protocol.(provider.StreamingProtocol)
	if !ok {
		s.respondError(w, http.StatusBadRequest, "Provider does not support streaming")
		return
	}

	// Disable write timeout for streaming - the server's WriteTimeout (30s default)
	// would kill long-running streams.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	conversationCtx, err := db.GetConversationContextByBeadID(req.BeadID)
	if err != nil {
		// No existing conversation, create new one
		projectID := agent.ProjectID
		if projectID == "" {
			projectID = "loom-self"
		}
		conversationCtx = models.NewConversationContext(
			uuid.New().String(),
			req.BeadID,
			projectID,
			7*24*time.Hour, // 7 day expiration for pair sessions
		)
		if agent.Name != "" {
			conversationCtx.Metadata["agent_name"] = agent.Name
		}
		if createErr := db.CreateConversationContext(conversationCtx); createErr != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create conversation: %v", createErr))
			return
		}
	}

	// Build system prompt from persona if conversation is new
	if len(conversationCtx.Messages) == 0 {
		systemPrompt := buildPairSystemPrompt(agent)
		conversationCtx.AddMessage("system", systemPrompt, len(systemPrompt)/4)
	}

	// Append user message to conversation
	conversationCtx.AddMessage("user", req.Message, len(req.Message)/4)

	// Save user message immediately
	if err := db.UpdateConversationContext(conversationCtx); err != nil {
		log.Printf("Warning: Failed to save user message: %v", err)
	}

	// Build provider message list from conversation history
	var messages []provider.ChatMessage
	for _, msg := range conversationCtx.Messages {
		messages = append(messages, provider.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Apply token limits (sliding window)
	messages = applyTokenLimits(messages, registeredProvider.Config.Model)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.respondError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Send connected event
	connData, _ := json.Marshal(map[string]any{
		"session_id":    conversationCtx.SessionID,
		"agent_name":    agent.Name,
		"message_count": len(conversationCtx.Messages),
	})
	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", connData)
	flusher.Flush()

	// Create provider request
	providerReq := &provider.ChatCompletionRequest{
		Model:       registeredProvider.Config.Model,
		Messages:    messages,
		Temperature: 0.7,
		Stream:      true,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	var streamedText strings.Builder

	// Stream response
	err = providerReg.SendChatCompletionStream(ctx, providerID, providerReq, func(chunk *provider.StreamChunk) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if len(chunk.Choices) > 0 {
			streamedText.WriteString(chunk.Choices[0].Delta.Content)
		}

		data, err := json.Marshal(chunk)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", data)
		flusher.Flush()

		return nil
	})

	if err != nil {
		errorData, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errorData)
		flusher.Flush()
		return
	}

	// Save assistant response to conversation
	responseText := streamedText.String()
	conversationCtx.AddMessage("assistant", responseText, len(responseText)/4)
	if err := db.UpdateConversationContext(conversationCtx); err != nil {
		log.Printf("Warning: Failed to save assistant response: %v", err)
	}

	// Try lenient action parsing (optional — no error if no actions)
	if router := s.app.GetActionRouter(); router != nil {
		actx := actions.ActionContext{
			AgentID:   req.AgentID,
			BeadID:    req.BeadID,
			ProjectID: conversationCtx.ProjectID,
		}
		env, parseErr := actions.DecodeLenient([]byte(responseText))
		if parseErr == nil && env != nil && len(env.Actions) > 0 {
			results, _ := router.Execute(ctx, env, actx)
			actionData, _ := json.Marshal(map[string]any{
				"actions": env.Actions,
				"results": results,
			})
			fmt.Fprintf(w, "event: actions\ndata: %s\n\n", actionData)
			flusher.Flush()
		}
		// No error event for parse failures — pair mode is conversational
	}

	// Send done event
	doneData, _ := json.Marshal(map[string]any{
		"tokens_used": conversationCtx.TokenCount,
	})
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", doneData)
	flusher.Flush()
}

// buildPairSystemPrompt builds a system prompt for pair-programming mode from agent persona
func buildPairSystemPrompt(agent *models.Agent) string {
	var prompt string

	if agent.Persona != nil {
		persona := agent.Persona

		if persona.Character != "" {
			prompt += fmt.Sprintf("# Your Character\n%s\n\n", persona.Character)
		}
		if persona.Mission != "" {
			prompt += fmt.Sprintf("# Your Mission\n%s\n\n", persona.Mission)
		}
		if persona.Personality != "" {
			prompt += fmt.Sprintf("# Your Personality\n%s\n\n", persona.Personality)
		}
		if len(persona.Capabilities) > 0 {
			prompt += "# Your Capabilities\n"
			for _, cap := range persona.Capabilities {
				prompt += fmt.Sprintf("- %s\n", cap)
			}
			prompt += "\n"
		}
		if persona.AutonomyInstructions != "" {
			prompt += fmt.Sprintf("# Autonomy Guidelines\n%s\n\n", persona.AutonomyInstructions)
		}
		if persona.DecisionInstructions != "" {
			prompt += fmt.Sprintf("# Decision Making\n%s\n\n", persona.DecisionInstructions)
		}
	} else {
		prompt = fmt.Sprintf("You are %s, an AI agent.\n\n", agent.Name)
	}

	// Pair-mode instruction instead of strict action prompt
	prompt += `# Pair-Programming Mode
You are in pair-programming mode with a human. Respond conversationally.
When you want to take actions (read/write files, run commands), include
an action block in a fenced JSON block. You may include explanation
before and after. If just discussing, no action block needed.
`

	return prompt
}

// applyTokenLimits implements sliding window token truncation
func applyTokenLimits(messages []provider.ChatMessage, model string) []provider.ChatMessage {
	modelLimit := getModelTokenLimit(model)
	maxTokens := int(float64(modelLimit) * 0.8)

	totalTokens := 0
	for _, msg := range messages {
		totalTokens += len(msg.Content) / 4
	}

	if totalTokens <= maxTokens {
		return messages
	}

	if len(messages) == 0 {
		return messages
	}

	systemMsg := messages[0]
	systemTokens := len(systemMsg.Content) / 4

	recentTokens := 0
	startIndex := len(messages)

	for i := len(messages) - 1; i > 0; i-- {
		msgTokens := len(messages[i].Content) / 4
		if systemTokens+recentTokens+msgTokens > maxTokens {
			startIndex = i + 1
			break
		}
		recentTokens += msgTokens
	}

	if startIndex > 1 {
		truncatedCount := startIndex - 1
		noticeMsg := provider.ChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("[Note: %d older messages truncated to stay within token limit]", truncatedCount),
		}
		result := []provider.ChatMessage{systemMsg, noticeMsg}
		result = append(result, messages[startIndex:]...)
		return result
	}

	return messages
}

// lookupAgentFromDB reads agent info directly from the database,
// bypassing the agent manager's in-memory map (which can block under lock contention).
func lookupAgentFromDB(db *database.Database, agentID string) (*models.Agent, error) {
	query := `SELECT id, name, role, persona_name, provider_id, status, project_id FROM agents WHERE id = ?`
	agent := &models.Agent{}
	var role, providerID, projectID *string
	err := db.DB().QueryRow(query, agentID).Scan(
		&agent.ID, &agent.Name, &role, &agent.PersonaName,
		&providerID, &agent.Status, &projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	if role != nil {
		agent.Role = *role
	}
	if providerID != nil {
		agent.ProviderID = *providerID
	}
	if projectID != nil {
		agent.ProjectID = *projectID
	}
	return agent, nil
}

// getModelTokenLimit returns the token limit for a given model
func getModelTokenLimit(model string) int {
	limits := map[string]int{
		"gpt-4":             8192,
		"gpt-4-32k":         32768,
		"gpt-4-turbo":       128000,
		"gpt-3.5-turbo":     4096,
		"gpt-3.5-turbo-16k": 16384,
		"claude-3-opus":     200000,
		"claude-3-sonnet":   200000,
		"claude-3-haiku":    200000,
	}
	if limit, ok := limits[model]; ok {
		return limit
	}
	return 100000
}
