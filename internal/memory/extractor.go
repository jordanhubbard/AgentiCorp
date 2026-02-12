package memory

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/loom/pkg/models"
)

// LessonStore is the subset of database.Database that the extractor needs.
type LessonStore interface {
	StoreLessonWithEmbedding(lesson *models.Lesson, embedding []float32) error
	CreateLesson(lesson *models.Lesson) error
}

// ActionEntry is a flattened action result for extraction analysis.
// The caller converts from their internal representation.
type ActionEntry struct {
	Iteration  int
	ActionType string
	Status     string
	Message    string
	Path       string
}

// Extractor processes action logs from completed loops and extracts
// durable lessons — patterns, errors, and decisions — as new lessons
// with category "conversation_insight".
type Extractor struct {
	store    LessonStore
	embedder Embedder
}

// NewExtractor creates an Extractor backed by the given store and embedder.
func NewExtractor(store LessonStore, embedder Embedder) *Extractor {
	return &Extractor{store: store, embedder: embedder}
}

// ExtractFromLoop scans action entries for extractable patterns and stores
// new lessons. Designed to be called at the end of ExecuteTaskWithLoop.
func (e *Extractor) ExtractFromLoop(projectID, beadID string, entries []ActionEntry, terminalReason string) {
	if e == nil || e.store == nil || len(entries) == 0 {
		return
	}

	var lessons []extractedLesson

	lessons = append(lessons, extractBuildPatterns(entries)...)
	lessons = append(lessons, extractTestPatterns(entries)...)
	lessons = append(lessons, extractEditPatterns(entries)...)

	if insight := extractTerminalInsight(terminalReason, len(entries)); insight != nil {
		lessons = append(lessons, *insight)
	}

	for _, l := range lessons {
		lesson := &models.Lesson{
			ID:             uuid.New().String(),
			ProjectID:      projectID,
			Category:       "conversation_insight",
			Title:          l.title,
			Detail:         l.detail,
			SourceBeadID:   beadID,
			CreatedAt:      time.Now(),
			RelevanceScore: 1.0,
		}

		// Embed and store
		if e.embedder != nil {
			text := l.title + " " + l.detail
			ctx := context.Background()
			embeddings, err := e.embedder.Embed(ctx, []string{text})
			if err == nil && len(embeddings) > 0 && len(embeddings[0]) > 0 {
				if err := e.store.StoreLessonWithEmbedding(lesson, embeddings[0]); err != nil {
					log.Printf("[Extractor] Failed to store lesson with embedding: %v", err)
				} else {
					log.Printf("[Extractor] Extracted lesson: %s", l.title)
				}
				continue
			}
		}

		if err := e.store.CreateLesson(lesson); err != nil {
			log.Printf("[Extractor] Failed to store lesson: %v", err)
		} else {
			log.Printf("[Extractor] Extracted lesson (no embedding): %s", l.title)
		}
	}
}

type extractedLesson struct {
	title  string
	detail string
}

func extractBuildPatterns(entries []ActionEntry) []extractedLesson {
	var failures []string
	for _, e := range entries {
		if e.ActionType == "build_project" && e.Status == "error" {
			failures = append(failures, truncateStr(e.Message, 200))
		}
	}
	if len(failures) < 2 {
		return nil
	}
	return []extractedLesson{{
		title:  fmt.Sprintf("Repeated build failures (%d times)", len(failures)),
		detail: "Build failed multiple times: " + strings.Join(failures[:min(len(failures), 3)], "; "),
	}}
}

func extractTestPatterns(entries []ActionEntry) []extractedLesson {
	var failures []string
	for _, e := range entries {
		if e.ActionType == "run_tests" && e.Status == "error" {
			failures = append(failures, truncateStr(e.Message, 200))
		}
	}
	if len(failures) < 2 {
		return nil
	}
	return []extractedLesson{{
		title:  fmt.Sprintf("Repeated test failures (%d times)", len(failures)),
		detail: "Tests failed multiple times: " + strings.Join(failures[:min(len(failures), 3)], "; "),
	}}
}

func extractEditPatterns(entries []ActionEntry) []extractedLesson {
	pathFailures := make(map[string]int)
	for _, e := range entries {
		if (e.ActionType == "apply_patch" || e.ActionType == "edit_code") && e.Status == "error" {
			if e.Path != "" {
				pathFailures[e.Path]++
			}
		}
	}
	var lessons []extractedLesson
	for path, count := range pathFailures {
		if count >= 2 {
			lessons = append(lessons, extractedLesson{
				title:  fmt.Sprintf("Repeated edit failures on %s", path),
				detail: fmt.Sprintf("File %s had %d edit failures — may need different approach", path, count),
			})
		}
	}
	return lessons
}

func extractTerminalInsight(reason string, totalActions int) *extractedLesson {
	switch reason {
	case "max_iterations":
		return &extractedLesson{
			title:  "Task hit max iterations",
			detail: fmt.Sprintf("Task exhausted all iterations with %d total actions — may be too complex for single bead", totalActions),
		}
	case "inner_loop":
		return &extractedLesson{
			title:  "Agent stuck in action loop",
			detail: "Agent repeated the same actions — needs clearer guidance or different approach",
		}
	case "parse_failures":
		return &extractedLesson{
			title:  "Agent produced unparseable responses",
			detail: "Agent failed to produce valid JSON actions — may need simpler prompt or different model",
		}
	}
	return nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
