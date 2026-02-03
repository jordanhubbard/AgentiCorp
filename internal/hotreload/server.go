package hotreload

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Server manages hot-reload WebSocket connections
type Server struct {
	watcher   *Watcher
	clients   map[*websocket.Conn]bool
	mu        sync.RWMutex
	upgrader  websocket.Upgrader
	broadcast chan FileChangeEvent
}

// NewServer creates a new hot-reload server
func NewServer(watcher *Watcher) *Server {
	s := &Server{
		watcher: watcher,
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		broadcast: make(chan FileChangeEvent, 10),
	}

	// Subscribe to file changes
	if watcher != nil {
		go s.watchFileChanges()
	}

	// Start broadcast loop
	go s.broadcastLoop()

	return s
}

// HandleWebSocket handles WebSocket connections for hot-reload
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[HotReload] Failed to upgrade connection: %v", err)
		return
	}

	// Register client
	s.mu.Lock()
	s.clients[conn] = true
	clientCount := len(s.clients)
	s.mu.Unlock()

	log.Printf("[HotReload] Client connected (total: %d)", clientCount)

	// Send initial connection message
	msg := map[string]interface{}{
		"type":    "connected",
		"message": "Hot-reload enabled",
	}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("[HotReload] Failed to send welcome message: %v", err)
	}

	// Keep connection alive and handle client messages
	go s.handleClient(conn)
}

// handleClient handles messages from a specific client
func (s *Server) handleClient(conn *websocket.Conn) {
	defer func() {
		// Unregister client
		s.mu.Lock()
		delete(s.clients, conn)
		clientCount := len(s.clients)
		s.mu.Unlock()

		conn.Close()
		log.Printf("[HotReload] Client disconnected (remaining: %d)", clientCount)
	}()

	for {
		// Read message (mostly for ping/pong keepalive)
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[HotReload] Unexpected close: %v", err)
			}
			break
		}
	}
}

// watchFileChanges subscribes to file change events from watcher
func (s *Server) watchFileChanges() {
	if s.watcher == nil {
		return
	}

	eventCh := s.watcher.Subscribe()
	defer s.watcher.Unsubscribe(eventCh)

	for event := range eventCh {
		// Forward to broadcast channel
		select {
		case s.broadcast <- event:
		default:
			// Broadcast channel full, skip
		}
	}
}

// broadcastLoop sends file change events to all connected clients
func (s *Server) broadcastLoop() {
	for event := range s.broadcast {
		s.mu.RLock()
		clients := make([]*websocket.Conn, 0, len(s.clients))
		for client := range s.clients {
			clients = append(clients, client)
		}
		s.mu.RUnlock()

		// Build message
		msg := map[string]interface{}{
			"type":      "file_changed",
			"path":      event.Path,
			"operation": event.Operation,
			"timestamp": event.Timestamp.Unix(),
		}

		// Broadcast to all clients
		for _, client := range clients {
			if err := client.WriteJSON(msg); err != nil {
				log.Printf("[HotReload] Failed to send to client: %v", err)
				// Client will be removed by handleClient goroutine
			}
		}

		log.Printf("[HotReload] Broadcasted change to %d clients", len(clients))
	}
}

// GetStats returns current hot-reload statistics
func (s *Server) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"connected_clients": len(s.clients),
		"enabled":           s.watcher != nil,
	}
}

// Close shuts down the hot-reload server
func (s *Server) Close() error {
	close(s.broadcast)

	s.mu.Lock()
	defer s.mu.Unlock()

	for client := range s.clients {
		client.Close()
	}
	s.clients = make(map[*websocket.Conn]bool)

	return nil
}

// HandleStatus returns hot-reload status as JSON
func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.GetStats()); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
