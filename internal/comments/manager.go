package comments

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jordanhubbard/agenticorp/internal/database"
	"github.com/jordanhubbard/agenticorp/internal/notifications"
	"github.com/jordanhubbard/agenticorp/internal/temporal/eventbus"
)

// Manager handles comment operations
type Manager struct {
	db              *database.Database
	notificationMgr *notifications.Manager
	eventBus        *eventbus.EventBus
}

// Comment represents a bead comment with replies
type Comment struct {
	ID             string     `json:"id"`
	BeadID         string     `json:"bead_id"`
	ParentID       string     `json:"parent_id,omitempty"`
	AuthorID       string     `json:"author_id"`
	AuthorUsername string     `json:"author_username"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Edited         bool       `json:"edited"`
	Replies        []*Comment `json:"replies,omitempty"`
	Mentions       []string   `json:"mentions,omitempty"`
}

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_-]+)`)

// NewManager creates a new comments manager
func NewManager(db *database.Database, notificationMgr *notifications.Manager, eventBus *eventbus.EventBus) *Manager {
	return &Manager{
		db:              db,
		notificationMgr: notificationMgr,
		eventBus:        eventBus,
	}
}

// CreateComment creates a new comment
func (m *Manager) CreateComment(beadID, authorID, authorUsername, content, parentID string) (*Comment, error) {
	now := time.Now()
	comment := &Comment{
		ID:             uuid.New().String(),
		BeadID:         beadID,
		ParentID:       parentID,
		AuthorID:       authorID,
		AuthorUsername: authorUsername,
		Content:        content,
		CreatedAt:      now,
		UpdatedAt:      now,
		Edited:         false,
	}

	// Save to database
	dbComment := &database.BeadComment{
		ID:             comment.ID,
		BeadID:         comment.BeadID,
		ParentID:       comment.ParentID,
		AuthorID:       comment.AuthorID,
		AuthorUsername: comment.AuthorUsername,
		Content:        comment.Content,
		CreatedAt:      comment.CreatedAt,
		UpdatedAt:      comment.UpdatedAt,
		Edited:         comment.Edited,
		Deleted:        false,
	}

	if err := m.db.CreateComment(dbComment); err != nil {
		return nil, err
	}

	// Parse and process mentions
	mentions := m.parseMentions(content)
	comment.Mentions = mentions

	if err := m.processMentions(comment.ID, mentions); err != nil {
		// Log error but don't fail comment creation
		fmt.Printf("Failed to process mentions: %v\n", err)
	}

	// Publish event to EventBus
	if m.eventBus != nil {
		m.publishCommentEvent("comment.created", comment)
	}

	return comment, nil
}

// GetComments retrieves all comments for a bead with threading
func (m *Manager) GetComments(beadID string) ([]*Comment, error) {
	dbComments, err := m.db.GetCommentsByBeadID(beadID)
	if err != nil {
		return nil, err
	}

	// Build comment map
	commentMap := make(map[string]*Comment)
	var topLevel []*Comment

	// First pass: create Comment objects
	for _, dbComment := range dbComments {
		comment := &Comment{
			ID:             dbComment.ID,
			BeadID:         dbComment.BeadID,
			ParentID:       dbComment.ParentID,
			AuthorID:       dbComment.AuthorID,
			AuthorUsername: dbComment.AuthorUsername,
			Content:        dbComment.Content,
			CreatedAt:      dbComment.CreatedAt,
			UpdatedAt:      dbComment.UpdatedAt,
			Edited:         dbComment.Edited,
			Replies:        []*Comment{},
		}

		// Parse mentions from content
		comment.Mentions = m.parseMentions(comment.Content)

		commentMap[comment.ID] = comment

		if comment.ParentID == "" {
			topLevel = append(topLevel, comment)
		}
	}

	// Second pass: build thread structure
	for _, comment := range commentMap {
		if comment.ParentID != "" {
			if parent, exists := commentMap[comment.ParentID]; exists {
				parent.Replies = append(parent.Replies, comment)
			}
		}
	}

	return topLevel, nil
}

// UpdateComment updates a comment's content
func (m *Manager) UpdateComment(commentID, authorID, content string) error {
	// Verify ownership
	dbComment, err := m.db.GetComment(commentID)
	if err != nil {
		return err
	}

	if dbComment.AuthorID != authorID {
		return fmt.Errorf("unauthorized: only the author can edit their comment")
	}

	if dbComment.Deleted {
		return fmt.Errorf("cannot edit deleted comment")
	}

	// Update comment
	if err := m.db.UpdateComment(commentID, content); err != nil {
		return err
	}

	// Parse new mentions (note: we don't re-notify for edits)
	mentions := m.parseMentions(content)

	// Publish event
	if m.eventBus != nil {
		comment := &Comment{
			ID:        commentID,
			BeadID:    dbComment.BeadID,
			AuthorID:  authorID,
			Content:   content,
			UpdatedAt: time.Now(),
			Mentions:  mentions,
		}
		m.publishCommentEvent("comment.updated", comment)
	}

	return nil
}

// DeleteComment soft deletes a comment
func (m *Manager) DeleteComment(commentID, authorID string) error {
	// Verify ownership
	dbComment, err := m.db.GetComment(commentID)
	if err != nil {
		return err
	}

	if dbComment.AuthorID != authorID {
		return fmt.Errorf("unauthorized: only the author can delete their comment")
	}

	// Delete comment
	if err := m.db.DeleteComment(commentID); err != nil {
		return err
	}

	// Publish event
	if m.eventBus != nil {
		comment := &Comment{
			ID:       commentID,
			BeadID:   dbComment.BeadID,
			AuthorID: authorID,
		}
		m.publishCommentEvent("comment.deleted", comment)
	}

	return nil
}

// parseMentions extracts @mentions from content
func (m *Manager) parseMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	var mentions []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			username := match[1]
			if !seen[username] {
				mentions = append(mentions, username)
				seen[username] = true
			}
		}
	}

	return mentions
}

// processMentions creates mention records and notifications
func (m *Manager) processMentions(commentID string, mentions []string) error {
	if len(mentions) == 0 {
		return nil
	}

	// Get all users to find mentioned users
	users, err := m.db.ListUsers()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Build username to user ID map
	userMap := make(map[string]string)
	for _, user := range users {
		userMap[user.Username] = user.ID
	}

	// Create mentions and notifications
	for _, username := range mentions {
		userID, exists := userMap[username]
		if !exists {
			// Skip if user doesn't exist
			continue
		}

		// Create mention record
		mention := &database.CommentMention{
			ID:                uuid.New().String(),
			CommentID:         commentID,
			MentionedUserID:   userID,
			MentionedUsername: username,
			CreatedAt:         time.Now(),
		}

		if err := m.db.CreateMention(mention); err != nil {
			return fmt.Errorf("failed to create mention: %w", err)
		}

		// Create notification via notification manager
		// This will be picked up by the notification rules
		if m.eventBus != nil {
			event := &eventbus.Event{
				ID:        uuid.New().String(),
				Type:      "mention.created",
				Timestamp: time.Now(),
				Source:    "comments",
				Data: map[string]interface{}{
					"mention_id":   mention.ID,
					"comment_id":   commentID,
					"mentioned_id": userID,
					"username":     username,
				},
			}
			_ = m.eventBus.Publish(event)
		}

		// Mark as notified
		if err := m.db.MarkMentionNotified(mention.ID); err != nil {
			// Log but don't fail
			fmt.Printf("Failed to mark mention as notified: %v\n", err)
		}
	}

	return nil
}

// publishCommentEvent publishes a comment event to the EventBus
func (m *Manager) publishCommentEvent(eventType string, comment *Comment) {
	event := &eventbus.Event{
		ID:        uuid.New().String(),
		Type:      eventbus.EventType(eventType),
		Timestamp: time.Now(),
		Source:    "comments",
		Data: map[string]interface{}{
			"comment_id":      comment.ID,
			"bead_id":         comment.BeadID,
			"author_id":       comment.AuthorID,
			"author_username": comment.AuthorUsername,
			"content":         comment.Content,
			"mentions":        comment.Mentions,
		},
	}

	_ = m.eventBus.Publish(event)
}
