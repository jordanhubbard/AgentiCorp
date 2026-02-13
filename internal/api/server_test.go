package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jordanhubbard/loom/pkg/config"
)

func TestServer_respondJSON(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "simple struct",
			status:     http.StatusOK,
			data:       map[string]string{"key": "value"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty object",
			status:     http.StatusOK,
			data:       map[string]interface{}{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "created status",
			status:     http.StatusCreated,
			data:       map[string]string{"id": "123"},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			server.respondJSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", ct)
			}

			// Verify JSON is valid
			var result interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Errorf("Invalid JSON response: %v", err)
			}
		})
	}
}

func TestServer_respondError(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			message:    "Invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			message:    "Resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error",
			status:     http.StatusInternalServerError,
			message:    "Server error",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			server.respondError(w, tt.status, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", ct)
			}

			// Verify error response structure
			var errResp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v", err)
			}

			if errResp["error"] != tt.message {
				t.Errorf("Error message = %s, want %s", errResp["error"], tt.message)
			}
		})
	}
}

func TestServer_parseJSON(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid JSON",
			body:    `{"name":"test","value":123}`,
			wantErr: false,
		},
		{
			name:    "empty JSON",
			body:    `{}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			body:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			var result map[string]interface{}
			err := server.parseJSON(req, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result for valid JSON")
			}
		})
	}
}

func TestServer_extractID(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name    string
		path    string
		prefix  string
		want    string
	}{
		{
			name:   "simple ID",
			path:   "/api/v1/agents/agent-123",
			prefix: "/api/v1/agents",
			want:   "agent-123",
		},
		{
			name:   "complex ID",
			path:   "/api/v1/projects/proj-456",
			prefix: "/api/v1/projects",
			want:   "proj-456",
		},
		{
			name:   "ID with dashes",
			path:   "/api/v1/beads/bead-test-123",
			prefix: "/api/v1/beads",
			want:   "bead-test-123",
		},
		{
			name:   "ID with slashes",
			path:   "/api/v1/personas/default/qa-engineer",
			prefix: "/api/v1/personas",
			want:   "default", // extractID only returns first part
		},
		{
			name:   "no ID",
			path:   "/api/v1/agents",
			prefix: "/api/v1/agents",
			want:   "",
		},
		{
			name:   "trailing slash",
			path:   "/api/v1/agents/",
			prefix: "/api/v1/agents",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := server.extractID(tt.path, tt.prefix)
			if got != tt.want {
				t.Errorf("extractID() = %v, want %v", got, tt.want)
			}
		})
	}
}


func TestNewServer(t *testing.T) {
	cfg := &config.Config{}

	// Test with nil app (minimal initialization)
	server := NewServer(nil, nil, nil, cfg)

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	if server.config != cfg {
		t.Error("Config not set correctly")
	}

	if server.apiFailureLast == nil {
		t.Error("apiFailureLast map not initialized")
	}
}

func TestServer_handleHealth(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		wantStatus  int
		wantSuccess bool
	}{
		{
			name:        "GET request",
			method:      http.MethodGet,
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "POST request",
			method:      http.MethodPost,
			wantStatus:  http.StatusMethodNotAllowed,
			wantSuccess: false,
		},
		{
			name:        "HEAD request",
			method:      http.MethodHead,
			wantStatus:  http.StatusMethodNotAllowed,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(nil, nil, nil, &config.Config{})
			req := httptest.NewRequest(tt.method, "/api/v1/health", nil)
			w := httptest.NewRecorder()

			server.handleHealth(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			// Verify response structure for successful requests
			if tt.wantSuccess {
				var health map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
					t.Fatalf("Failed to unmarshal health response: %v", err)
				}

				if health["status"] != "ok" {
					t.Errorf("Status = %v, want ok", health["status"])
				}
			}
		})
	}
}


func TestServer_SetupRoutes(t *testing.T) {
	cfg := &config.Config{
		WebUI: config.WebUIConfig{
			Enabled:    false, // Disable to avoid file system dependencies
			StaticPath: "",
		},
	}

	server := NewServer(nil, nil, nil, cfg)
	handler := server.SetupRoutes()

	if handler == nil {
		t.Fatal("SetupRoutes() returned nil handler")
	}

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health endpoint status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestServer_ConcurrentResponses(t *testing.T) {
	server := &Server{}
	done := make(chan bool)

	// Test concurrent respondJSON calls (no data races)
	for i := 0; i < 10; i++ {
		go func(id int) {
			w := httptest.NewRecorder()
			server.respondJSON(w, http.StatusOK, map[string]int{"id": id})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent respondError calls
	for i := 0; i < 10; i++ {
		go func(id int) {
			w := httptest.NewRecorder()
			server.respondError(w, http.StatusBadRequest, "error")
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
