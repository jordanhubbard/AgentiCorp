package api

import (
	"context"
	"net/http"
	"strings"

	internalmodels "github.com/jordanhubbard/agenticorp/internal/models"
)

// handleProviders handles GET/POST /api/v1/providers
func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		providers, err := s.agenticorp.ListProviders()
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, providers)

	case http.MethodPost:
		var req internalmodels.Provider
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		created, err := s.agenticorp.RegisterProvider(context.Background(), &req)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.respondJSON(w, http.StatusCreated, created)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleProvider handles GET/DELETE /api/v1/providers/{id} and GET /api/v1/providers/{id}/models
func (s *Server) handleProvider(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/providers/")
	parts := strings.Split(path, "/")
	providerID := parts[0]

	if providerID == "" {
		s.respondError(w, http.StatusBadRequest, "Missing provider id")
		return
	}

	if len(parts) > 1 && parts[1] == "models" {
		if r.Method != http.MethodGet {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		models, err := s.agenticorp.GetProviderModels(context.Background(), providerID)
		if err != nil {
			s.respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, map[string]interface{}{"models": models})
		return
	}
	if len(parts) > 1 && parts[1] == "negotiate" {
		if r.Method != http.MethodPost {
			s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		updated, err := s.agenticorp.NegotiateProviderModel(context.Background(), providerID)
		if err != nil {
			s.respondError(w, http.StatusBadGateway, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, updated)
		return
	}

	switch r.Method {
	case http.MethodGet:
		providers, err := s.agenticorp.ListProviders()
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, p := range providers {
			if p.ID == providerID {
				s.respondJSON(w, http.StatusOK, p)
				return
			}
		}
		s.respondError(w, http.StatusNotFound, "Provider not found")

	case http.MethodDelete:
		if err := s.agenticorp.DeleteProvider(context.Background(), providerID); err != nil {
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodPut:
		var req internalmodels.Provider
		if err := s.parseJSON(r, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		req.ID = providerID
		updated, err := s.agenticorp.UpdateProvider(context.Background(), &req)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.respondJSON(w, http.StatusOK, updated)

	default:
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
