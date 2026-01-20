package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jordanhubbard/agenticorp/pkg/models"
)

// handleProjectStateEndpoints handles project state management endpoints
func (s *Server) handleProjectStateEndpoints(w http.ResponseWriter, r *http.Request, id string, action string) {
	switch action {
	case "close":
		s.handleCloseProject(w, r, id)
	case "reopen":
		s.handleReopenProject(w, r, id)
	case "comments":
		s.handleProjectComments(w, r, id)
	case "state":
		s.handleProjectState(w, r, id)
	case "agents":
		s.handleProjectAgents(w, r, id)
	default:
		s.respondError(w, http.StatusNotFound, "Unknown action")
	}
}

// handleProjectAgents handles POST /api/v1/projects/{id}/agents
func (s *Server) handleProjectAgents(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
		Action  string `json:"action"`
	}
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.AgentID == "" {
		s.respondError(w, http.StatusBadRequest, "agent_id is required")
		return
	}
	if req.Action == "" {
		req.Action = "assign"
	}

	switch req.Action {
	case "assign":
		if err := s.agenticorp.AssignAgentToProject(req.AgentID, id); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	case "unassign":
		if err := s.agenticorp.UnassignAgentFromProject(req.AgentID, id); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	default:
		s.respondError(w, http.StatusBadRequest, "action must be assign or unassign")
		return
	}

	project, _ := s.agenticorp.GetProjectManager().GetProject(id)
	s.respondJSON(w, http.StatusOK, project)
}

// handleCloseProject handles POST /api/v1/projects/{id}/close
func (s *Server) handleCloseProject(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		AuthorID string `json:"author_id"`
		Comment  string `json:"comment"`
	}
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AuthorID == "" {
		s.respondError(w, http.StatusBadRequest, "author_id is required")
		return
	}

	// Check if project has open work
	openBeads, err := s.agenticorp.GetReadyBeads(id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	hasOpenWork := len(openBeads) > 0
	if hasOpenWork {
		// Create a decision bead for closure
		decision, err := s.agenticorp.CreateDecisionBead(
			fmt.Sprintf("Should project '%s' be closed despite having %d open beads?", id, len(openBeads)),
			"",
			req.AuthorID,
			[]string{"yes", "no"},
			"no",
			models.BeadPriorityP1,
			id,
		)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.respondJSON(w, http.StatusAccepted, map[string]interface{}{
			"status":        "decision_required",
			"message":       "Project has open work, decision required from agents",
			"decision_bead": decision,
		})
		return
	}

	// No open work, close directly
	if err := s.agenticorp.GetProjectManager().CloseProject(id, req.AuthorID, req.Comment); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.agenticorp.PersistProject(id)

	project, _ := s.agenticorp.GetProjectManager().GetProject(id)
	s.respondJSON(w, http.StatusOK, project)
}

// handleReopenProject handles POST /api/v1/projects/{id}/reopen
func (s *Server) handleReopenProject(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		AuthorID string `json:"author_id"`
		Comment  string `json:"comment"`
	}
	if err := s.parseJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.AuthorID == "" {
		s.respondError(w, http.StatusBadRequest, "author_id is required")
		return
	}

	if err := s.agenticorp.GetProjectManager().ReopenProject(id, req.AuthorID, req.Comment); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.agenticorp.PersistProject(id)

	project, _ := s.agenticorp.GetProjectManager().GetProject(id)
	s.respondJSON(w, http.StatusOK, project)
}

// handleProjectComments handles GET/POST /api/v1/projects/{id}/comments
func (s *Server) handleProjectComments(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		comments, err := s.agenticorp.GetProjectManager().GetComments(id)
		if err != nil {
			s.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, comments)

	case http.MethodPost:
		var req struct {
			AuthorID string `json:"author_id"`
			Comment  string `json:"comment"`
		}
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.AuthorID == "" || req.Comment == "" {
			s.respondError(w, http.StatusBadRequest, "author_id and comment are required")
			return
		}

		comment, err := s.agenticorp.GetProjectManager().AddComment(id, req.AuthorID, req.Comment)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.agenticorp.PersistProject(id)

		s.respondJSON(w, http.StatusCreated, comment)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleProjectState handles GET /api/v1/projects/{id}/state
func (s *Server) handleProjectState(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	project, err := s.agenticorp.GetProjectManager().GetProject(id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Project not found")
		return
	}

	// Check for open work
	openBeads, _ := s.agenticorp.GetReadyBeads(id)
	hasOpenWork := len(openBeads) > 0
	canClose := s.agenticorp.GetProjectManager().CanClose(id, hasOpenWork)

	state := map[string]interface{}{
		"id":             project.ID,
		"name":           project.Name,
		"status":         project.Status,
		"is_perpetual":   project.IsPerpetual,
		"is_sticky":      project.IsSticky,
		"open_beads":     len(openBeads),
		"can_close":      canClose,
		"created_at":     project.CreatedAt,
		"updated_at":     project.UpdatedAt,
		"closed_at":      project.ClosedAt,
		"comments_count": len(project.Comments),
	}

	s.respondJSON(w, http.StatusOK, state)
}

// updateHandleProject updates the existing handleProject to route to state management endpoints
func updateHandleProject(s *Server, w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/")
	parts := strings.Split(path, "/")
	id := parts[0]

	// Handle sub-endpoints for project state management
	if len(parts) > 1 {
		action := parts[1]
		s.handleProjectStateEndpoints(w, r, id, action)
		return
	}

	// Default GET behavior
	if r.Method == http.MethodGet {
		project, err := s.agenticorp.GetProjectManager().GetProject(id)
		if err != nil {
			s.respondError(w, http.StatusNotFound, "Project not found")
			return
		}
		s.respondJSON(w, http.StatusOK, project)
	} else {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
