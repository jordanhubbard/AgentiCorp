package dispatch

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/loom/internal/database"
	"github.com/jordanhubbard/loom/internal/memory"
	"github.com/jordanhubbard/loom/pkg/models"
)

// LessonsProvider retrieves and records lessons from the database.
// It implements the worker.LessonsProvider interface.
type LessonsProvider struct {
	db       *database.Database
	embedder memory.Embedder
}

// NewLessonsProvider creates a new LessonsProvider backed by the given database.
// It uses a hash-based embedder by default for semantic search.
func NewLessonsProvider(db *database.Database) *LessonsProvider {
	if db == nil {
		return nil
	}
	return &LessonsProvider{
		db:       db,
		embedder: memory.NewHashEmbedder(),
	}
}

// SetEmbedder replaces the default hash embedder with a provider-backed one.
func (lp *LessonsProvider) SetEmbedder(e memory.Embedder) {
	if lp != nil && e != nil {
		lp.embedder = e
	}
}

// GetLessonsForPrompt retrieves lessons for a project and formats them as markdown
// suitable for injection into the system prompt.
func (lp *LessonsProvider) GetLessonsForPrompt(projectID string) string {
	if lp == nil || lp.db == nil || projectID == "" {
		return ""
	}

	lessons, err := lp.db.GetLessonsForProject(projectID, 15, 4000)
	if err != nil {
		log.Printf("[LessonsProvider] Failed to get lessons for project %s: %v", projectID, err)
		return ""
	}

	if len(lessons) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("The following lessons were learned from previous work on this project.\n")
	sb.WriteString("Avoid repeating these mistakes:\n\n")

	for _, l := range lessons {
		sb.WriteString(fmt.Sprintf("### %s: %s\n", strings.ToUpper(l.Category), l.Title))
		sb.WriteString(fmt.Sprintf("- %s\n", l.Detail))
		if l.RelevanceScore < 0.3 {
			sb.WriteString("- (older lesson, may be less relevant)\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetRelevantLessons retrieves the top-K lessons most semantically relevant
// to the given task context. Falls back to GetLessonsForPrompt on any error.
func (lp *LessonsProvider) GetRelevantLessons(projectID, taskContext string, topK int) string {
	if lp == nil || lp.db == nil || projectID == "" {
		return ""
	}

	if taskContext == "" || lp.embedder == nil {
		return lp.GetLessonsForPrompt(projectID)
	}

	if topK <= 0 {
		topK = 5
	}

	// Embed the task context
	ctx := context.Background()
	embeddings, err := lp.embedder.Embed(ctx, []string{taskContext})
	if err != nil {
		log.Printf("[LessonsProvider] Embedding failed, falling back to recency: %v", err)
		return lp.GetLessonsForPrompt(projectID)
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return lp.GetLessonsForPrompt(projectID)
	}

	queryEmb := embeddings[0]

	// Search by similarity
	lessons, err := lp.db.SearchLessonsBySimilarity(projectID, queryEmb, topK)
	if err != nil {
		log.Printf("[LessonsProvider] Similarity search failed, falling back to recency: %v", err)
		return lp.GetLessonsForPrompt(projectID)
	}

	if len(lessons) == 0 {
		return ""
	}

	// Format as markdown (max 2000 chars)
	var sb strings.Builder
	sb.WriteString("The following lessons are relevant to this task.\n")
	sb.WriteString("Apply them where appropriate:\n\n")

	totalChars := 0
	for _, l := range lessons {
		entry := fmt.Sprintf("### %s: %s\n- %s\n\n", strings.ToUpper(l.Category), l.Title, l.Detail)
		totalChars += len(entry)
		if totalChars > 2000 {
			break
		}
		sb.WriteString(entry)
	}

	return sb.String()
}

// RecordLesson creates a new lesson from observed agent behavior.
// It also embeds the lesson text for future semantic search.
func (lp *LessonsProvider) RecordLesson(projectID, category, title, detail, beadID, agentID string) error {
	if lp == nil || lp.db == nil {
		return nil
	}

	lesson := &models.Lesson{
		ID:             uuid.New().String(),
		ProjectID:      projectID,
		Category:       category,
		Title:          title,
		Detail:         detail,
		SourceBeadID:   beadID,
		SourceAgentID:  agentID,
		CreatedAt:      time.Now(),
		RelevanceScore: 1.0,
	}

	// Try to embed the lesson text for semantic search
	if lp.embedder != nil {
		text := title + " " + detail
		ctx := context.Background()
		embeddings, err := lp.embedder.Embed(ctx, []string{text})
		if err == nil && len(embeddings) > 0 && len(embeddings[0]) > 0 {
			if err := lp.db.StoreLessonWithEmbedding(lesson, embeddings[0]); err != nil {
				log.Printf("[LessonsProvider] Failed to record lesson with embedding: %v", err)
				return err
			}
			log.Printf("[LessonsProvider] Recorded lesson with embedding: [%s] %s", category, title)
			return nil
		}
		// Embedding failed — fall through to store without embedding
	}

	if err := lp.db.CreateLesson(lesson); err != nil {
		log.Printf("[LessonsProvider] Failed to record lesson: %v", err)
		return err
	}

	log.Printf("[LessonsProvider] Recorded lesson: [%s] %s", category, title)
	return nil
}
