package database

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jordanhubbard/loom/internal/memory"
	"github.com/jordanhubbard/loom/pkg/models"
)

// migrateLessons creates the lessons table if it doesn't exist
// and adds the embedding column for semantic search.
func (d *Database) migrateLessons() error {
	schema := `
	CREATE TABLE IF NOT EXISTS lessons (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		detail TEXT NOT NULL,
		source_bead_id TEXT,
		source_agent_id TEXT,
		relevance_score REAL NOT NULL DEFAULT 1.0,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_lessons_project ON lessons(project_id);
	CREATE INDEX IF NOT EXISTS idx_lessons_category ON lessons(category);
	`
	if _, err := d.db.Exec(schema); err != nil {
		return err
	}

	// Add embedding column if it doesn't exist (migration)
	_, err := d.db.Exec(`ALTER TABLE lessons ADD COLUMN embedding BLOB`)
	if err != nil {
		// Column already exists — ignore the error
		if !isAlterColumnExistsError(err) {
			return err
		}
	}
	return nil
}

// isAlterColumnExistsError checks if an ALTER TABLE error is "column already exists".
func isAlterColumnExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return len(msg) >= 9 && msg[:9] == "duplicate"
}

// CreateLesson inserts a new lesson record.
func (d *Database) CreateLesson(lesson *models.Lesson) error {
	if lesson == nil {
		return fmt.Errorf("lesson cannot be nil")
	}
	if lesson.CreatedAt.IsZero() {
		lesson.CreatedAt = time.Now()
	}
	if lesson.RelevanceScore == 0 {
		lesson.RelevanceScore = 1.0
	}

	_, err := d.db.Exec(`
		INSERT INTO lessons (id, project_id, category, title, detail, source_bead_id, source_agent_id, relevance_score, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		lesson.ID, lesson.ProjectID, lesson.Category, lesson.Title, lesson.Detail,
		lesson.SourceBeadID, lesson.SourceAgentID, lesson.RelevanceScore, lesson.CreatedAt,
	)
	return err
}

// GetLessonsForProject retrieves recent lessons for a project, up to limit count
// and maxChars total detail characters. Lessons are scored with time decay.
func (d *Database) GetLessonsForProject(projectID string, limit int, maxChars int) ([]*models.Lesson, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := d.db.Query(`
		SELECT id, project_id, category, title, detail, source_bead_id, source_agent_id, relevance_score, created_at
		FROM lessons
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT ?`,
		projectID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []*models.Lesson
	totalChars := 0
	now := time.Now()

	for rows.Next() {
		l := &models.Lesson{}
		err := rows.Scan(&l.ID, &l.ProjectID, &l.Category, &l.Title, &l.Detail,
			&l.SourceBeadID, &l.SourceAgentID, &l.RelevanceScore, &l.CreatedAt)
		if err != nil {
			return lessons, err
		}

		// Apply time decay: halve relevance every 7 days
		ageDays := now.Sub(l.CreatedAt).Hours() / 24
		l.RelevanceScore = l.RelevanceScore * math.Pow(0.5, ageDays/7.0)

		totalChars += len(l.Detail)
		if maxChars > 0 && totalChars > maxChars {
			break
		}

		lessons = append(lessons, l)
	}

	return lessons, rows.Err()
}

// StoreLessonWithEmbedding inserts a lesson along with its vector embedding.
func (d *Database) StoreLessonWithEmbedding(lesson *models.Lesson, embedding []float32) error {
	if lesson == nil {
		return fmt.Errorf("lesson cannot be nil")
	}
	if lesson.CreatedAt.IsZero() {
		lesson.CreatedAt = time.Now()
	}
	if lesson.RelevanceScore == 0 {
		lesson.RelevanceScore = 1.0
	}

	embBytes := memory.EncodeEmbedding(embedding)

	_, err := d.db.Exec(`
		INSERT INTO lessons (id, project_id, category, title, detail, source_bead_id, source_agent_id, relevance_score, created_at, embedding)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		lesson.ID, lesson.ProjectID, lesson.Category, lesson.Title, lesson.Detail,
		lesson.SourceBeadID, lesson.SourceAgentID, lesson.RelevanceScore, lesson.CreatedAt, embBytes,
	)
	return err
}

// SearchLessonsBySimilarity retrieves lessons for a project ranked by cosine
// similarity to the query embedding. Returns the top-K most similar lessons.
// Similarity is computed in Go — for typical lesson counts (<100) this is fast.
func (d *Database) SearchLessonsBySimilarity(projectID string, queryEmbedding []float32, topK int) ([]*models.Lesson, error) {
	if topK <= 0 {
		topK = 5
	}

	rows, err := d.db.Query(`
		SELECT id, project_id, category, title, detail, source_bead_id, source_agent_id, relevance_score, created_at, embedding
		FROM lessons
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT 200`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scored struct {
		lesson     *models.Lesson
		similarity float32
	}

	var candidates []scored
	now := time.Now()

	for rows.Next() {
		l := &models.Lesson{}
		var embBytes []byte
		err := rows.Scan(&l.ID, &l.ProjectID, &l.Category, &l.Title, &l.Detail,
			&l.SourceBeadID, &l.SourceAgentID, &l.RelevanceScore, &l.CreatedAt, &embBytes)
		if err != nil {
			return nil, err
		}

		// Apply time decay
		ageDays := now.Sub(l.CreatedAt).Hours() / 24
		l.RelevanceScore = l.RelevanceScore * math.Pow(0.5, ageDays/7.0)

		embedding := memory.DecodeEmbedding(embBytes)
		if len(embedding) == 0 || len(queryEmbedding) == 0 {
			// No embedding — use a low default similarity so unembedded
			// lessons still appear if there aren't enough embedded ones
			candidates = append(candidates, scored{lesson: l, similarity: 0.1})
			continue
		}

		sim := memory.CosineSimilarity(queryEmbedding, embedding)
		// Combine cosine similarity with time-decayed relevance score
		combined := float32(l.RelevanceScore)*0.3 + sim*0.7
		candidates = append(candidates, scored{lesson: l, similarity: combined})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort by combined similarity score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].similarity > candidates[j].similarity
	})

	// Return top-K
	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	result := make([]*models.Lesson, len(candidates))
	for i, c := range candidates {
		result[i] = c.lesson
	}
	return result, nil
}
