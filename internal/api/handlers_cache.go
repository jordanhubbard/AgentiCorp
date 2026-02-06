package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jordanhubbard/loom/internal/auth"
	"github.com/jordanhubbard/loom/internal/cache"
)

// handleGetCacheStats handles GET /api/v1/cache/stats
func (s *Server) handleGetCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Authentication required
	userID := auth.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get cache stats
	if s.cache == nil {
		http.Error(w, "Cache not initialized", http.StatusInternalServerError)
		return
	}

	stats := s.cache.GetStats(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetCacheConfig handles GET /api/v1/cache/config
func (s *Server) handleGetCacheConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Authentication required (admin only for config)
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	// Return current cache configuration
	if s.config == nil || s.cache == nil {
		http.Error(w, "Cache not configured", http.StatusInternalServerError)
		return
	}

	cacheConfig := s.config.Cache

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":        cacheConfig.Enabled,
		"default_ttl":    cacheConfig.DefaultTTL.String(),
		"max_size":       cacheConfig.MaxSize,
		"max_memory_mb":  cacheConfig.MaxMemoryMB,
		"cleanup_period": cacheConfig.CleanupPeriod.String(),
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleClearCache handles POST /api/v1/cache/clear
func (s *Server) handleClearCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Authentication required (admin only)
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	if s.cache == nil {
		http.Error(w, "Cache not initialized", http.StatusInternalServerError)
		return
	}

	// Clear the cache
	s.cache.Clear(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cache cleared successfully",
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleInvalidateCache handles POST /api/v1/cache/invalidate
func (s *Server) handleInvalidateCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Authentication required (admin only)
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		return
	}

	if s.cache == nil {
		http.Error(w, "Cache not initialized", http.StatusInternalServerError)
		return
	}

	// Parse invalidation request
	var req struct {
		Type  string `json:"type"`  // "provider", "model", "age", "pattern"
		Value string `json:"value"` // Provider ID, model name, age duration, or pattern
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var removed int
	var invalidationType string

	switch req.Type {
	case "provider":
		removed = s.cache.InvalidateByProvider(r.Context(), req.Value)
		invalidationType = "provider: " + req.Value
	case "model":
		removed = s.cache.InvalidateByModel(r.Context(), req.Value)
		invalidationType = "model: " + req.Value
	case "age":
		duration, err := time.ParseDuration(req.Value)
		if err != nil {
			http.Error(w, "Invalid age duration: "+err.Error(), http.StatusBadRequest)
			return
		}
		removed = s.cache.InvalidateByAge(r.Context(), duration)
		invalidationType = "age: older than " + req.Value
	case "pattern":
		removed = s.cache.InvalidateByPattern(r.Context(), req.Value)
		invalidationType = "pattern: " + req.Value
	default:
		http.Error(w, "Invalid invalidation type. Use: provider, model, age, or pattern", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"removed":        removed,
		"type":           invalidationType,
		"invalidated_at": time.Now().Format(time.RFC3339),
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// CacheToCacheConfig converts cache.Config to a format suitable for API responses
func CacheToCacheConfig(c *cache.Config) map[string]interface{} {
	return map[string]interface{}{
		"enabled":        c.Enabled,
		"default_ttl":    c.DefaultTTL.String(),
		"max_size":       c.MaxSize,
		"max_memory_mb":  c.MaxMemoryMB,
		"cleanup_period": c.CleanupPeriod.String(),
	}
}
