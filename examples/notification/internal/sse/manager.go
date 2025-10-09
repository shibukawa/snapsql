package sse

import (
	"encoding/json"
	"log"
	"sync"
)

// Client represents a connected SSE client.
type Client struct {
	UserID string
	Chan   chan []byte
	Done   chan struct{}
}

// Manager manages SSE client connections and broadcasts events.
type Manager struct {
	clients map[string][]*Client
	mu      sync.RWMutex
}

// NewManager creates a new SSE manager.
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string][]*Client),
	}
}

// AddClient registers a new client connection.
func (m *Manager) AddClient(userID string, client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[userID] = append(m.clients[userID], client)
	log.Printf("SSE: client connected for user %s (total: %d)", userID, len(m.clients[userID]))
}

// RemoveClient unregisters a client connection.
func (m *Manager) RemoveClient(userID string, client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	clients := m.clients[userID]
	for i, c := range clients {
		if c == client {
			m.clients[userID] = append(clients[:i], clients[i+1:]...)
			close(client.Chan)
			close(client.Done)
			log.Printf("SSE: client disconnected for user %s (remaining: %d)", userID, len(m.clients[userID]))
			break
		}
	}

	if len(m.clients[userID]) == 0 {
		delete(m.clients, userID)
	}
}

// Broadcast sends an event to all connected clients for a specific user.
func (m *Manager) Broadcast(userID string, event Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := m.clients[userID]
	if len(clients) == 0 {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("SSE: failed to marshal event: %v", err)
		return
	}

	message := append([]byte("data: "), data...)
	message = append(message, []byte("\n\n")...)

	for _, client := range clients {
		select {
		case client.Chan <- message:
		case <-client.Done:
			// Client is disconnected, skip
		default:
			// Channel is full, skip to avoid blocking
			log.Printf("SSE: skipping broadcast to user %s (channel full)", userID)
		}
	}

	log.Printf("SSE: broadcasted %s event to %d clients for user %s", event.Type, len(clients), userID)
}

// BroadcastToMultiple sends an event to all specified users.
func (m *Manager) BroadcastToMultiple(userIDs []string, event Event) {
	for _, userID := range userIDs {
		m.Broadcast(userID, event)
	}
}

// Event represents an SSE event.
type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
