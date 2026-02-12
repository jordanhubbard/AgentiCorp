package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jordanhubbard/loom/internal/logging"
)

// HandleLogsRecent returns recent log entries
func (s *Server) HandleLogsRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	level := r.URL.Query().Get("level")
	source := r.URL.Query().Get("source")
	sinceStr := r.URL.Query().Get("since")
	untilStr := r.URL.Query().Get("until")
	agentID := r.URL.Query().Get("agent_id")
	beadID := r.URL.Query().Get("bead_id")
	projectID := r.URL.Query().Get("project_id")

	var logs []logging.LogEntry
	var err error

	// If 'since' is provided, query from database
	var since time.Time
	var until time.Time
	if sinceStr != "" {
		var parseErr error
		since, parseErr = time.Parse(time.RFC3339, sinceStr)
		if parseErr != nil {
			http.Error(w, fmt.Sprintf("Invalid 'since' parameter: %v", parseErr), http.StatusBadRequest)
			return
		}
	}
	if untilStr != "" {
		var parseErr error
		until, parseErr = time.Parse(time.RFC3339, untilStr)
		if parseErr != nil {
			http.Error(w, fmt.Sprintf("Invalid 'until' parameter: %v", parseErr), http.StatusBadRequest)
			return
		}
	}

	logs, err = s.logManager.Query(limit, level, source, agentID, beadID, projectID, since, until)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query logs: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	}

	s.respondJSON(w, http.StatusOK, response)
}

// HandleLogsStream streams log entries via Server-Sent Events (SSE)
func (s *Server) HandleLogsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Disable write timeout for SSE - the server's WriteTimeout (30s default)
	// would kill long-running streams.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse filters from query params
	levelFilter := r.URL.Query().Get("level")
	sourceFilter := r.URL.Query().Get("source")
	agentFilter := r.URL.Query().Get("agent_id")
	beadFilter := r.URL.Query().Get("bead_id")
	projectFilter := r.URL.Query().Get("project_id")

	// Create a channel for this client
	logChan := make(chan logging.LogEntry, 100)

	// Register handler for new logs
	handler := func(entry logging.LogEntry) {
		// Apply filters
		if levelFilter != "" && entry.Level != levelFilter {
			return
		}
		if sourceFilter != "" && entry.Source != sourceFilter {
			return
		}
		if agentFilter != "" && getLogMeta(entry.Metadata, "agent_id") != agentFilter {
			return
		}
		if beadFilter != "" && getLogMeta(entry.Metadata, "bead_id") != beadFilter {
			return
		}
		if projectFilter != "" && getLogMeta(entry.Metadata, "project_id") != projectFilter {
			return
		}

		select {
		case logChan <- entry:
		default:
			// Channel full, skip
		}
	}

	s.logManager.AddHandler(handler)

	// Send logs to client
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial recent logs
	recentLogs := s.logManager.GetRecent(50, levelFilter, sourceFilter, agentFilter, beadFilter, projectFilter, time.Time{}, time.Time{})
	for _, entry := range recentLogs {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
	}
	flusher.Flush()

	// Stream new logs
	ctx := r.Context()
	ticker := time.NewTicker(30 * time.Second) // Heartbeat
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case entry := <-logChan:
			data, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			// Send heartbeat comment
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// HandleLogsExport exports logs as JSON or CSV
func (s *Server) HandleLogsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	agentID := r.URL.Query().Get("agent_id")
	beadID := r.URL.Query().Get("bead_id")
	projectID := r.URL.Query().Get("project_id")
	level := r.URL.Query().Get("level")
	source := r.URL.Query().Get("source")

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'start_time' parameter: %v", err), http.StatusBadRequest)
			return
		}
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid 'end_time' parameter: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Query logs
	logs, err := s.logManager.Query(0, level, source, agentID, beadID, projectID, startTime, endTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export logs: %v", err), http.StatusInternalServerError)
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"logs-%s.json\"", time.Now().Format("2006-01-02")))
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"logs":  logs,
			"count": len(logs),
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"logs-%s.csv\"", time.Now().Format("2006-01-02")))

		// Write CSV header
		fmt.Fprintln(w, "Timestamp,Level,Source,Message,Metadata")

		// Write CSV rows
		for _, log := range logs {
			metadataJSON := ""
			if log.Metadata != nil {
				data, _ := json.Marshal(log.Metadata)
				metadataJSON = string(data)
			}
			// Escape CSV fields
			message := strings.ReplaceAll(log.Message, "\"", "\"\"")
			metadataJSON = strings.ReplaceAll(metadataJSON, "\"", "\"\"")
			fmt.Fprintf(w, "%s,%s,%s,\"%s\",\"%s\"\n",
				log.Timestamp.Format(time.RFC3339),
				log.Level,
				log.Source,
				message,
				metadataJSON,
			)
		}
	default:
		http.Error(w, "Unsupported format. Use 'json' or 'csv'", http.StatusBadRequest)
	}
}

func getLogMeta(meta map[string]interface{}, key string) string {
	if meta == nil {
		return ""
	}
	if val, ok := meta[key].(string); ok {
		return val
	}
	return ""
}
