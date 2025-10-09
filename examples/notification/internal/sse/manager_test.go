package sse

import (
	"encoding/json"
	"testing"
	"time"
)

func TestManager_AddRemoveClient(t *testing.T) {
	manager := NewManager()

	client := &Client{
		UserID: "user1",
		Chan:   make(chan []byte, 10),
		Done:   make(chan struct{}),
	}

	// Add client
	manager.AddClient("user1", client)

	manager.mu.RLock()
	if len(manager.clients["user1"]) != 1 {
		t.Errorf("expected 1 client, got %d", len(manager.clients["user1"]))
	}
	manager.mu.RUnlock()

	// Remove client
	manager.RemoveClient("user1", client)

	manager.mu.RLock()
	if len(manager.clients["user1"]) != 0 {
		t.Errorf("expected 0 clients, got %d", len(manager.clients["user1"]))
	}
	manager.mu.RUnlock()
}

func TestManager_Broadcast(t *testing.T) {
	manager := NewManager()

	client := &Client{
		UserID: "user1",
		Chan:   make(chan []byte, 10),
		Done:   make(chan struct{}),
	}

	manager.AddClient("user1", client)
	defer manager.RemoveClient("user1", client)

	// Broadcast event
	event := Event{
		Type: "notification",
		Payload: map[string]interface{}{
			"id":    1,
			"title": "Test",
		},
	}

	manager.Broadcast("user1", event)

	// Receive message
	select {
	case msg := <-client.Chan:
		// Verify message format
		if len(msg) < 6 || string(msg[:6]) != "data: " {
			t.Errorf("expected message to start with 'data: ', got %s", string(msg))
		}

		// Verify JSON payload
		jsonData := msg[6 : len(msg)-2] // Remove "data: " prefix and "\n\n" suffix
		var receivedEvent Event
		if err := json.Unmarshal(jsonData, &receivedEvent); err != nil {
			t.Errorf("failed to unmarshal event: %v", err)
		}

		if receivedEvent.Type != "notification" {
			t.Errorf("expected type 'notification', got %s", receivedEvent.Type)
		}

	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for message")
	}
}

func TestManager_BroadcastToMultiple(t *testing.T) {
	manager := NewManager()

	client1 := &Client{
		UserID: "user1",
		Chan:   make(chan []byte, 10),
		Done:   make(chan struct{}),
	}

	client2 := &Client{
		UserID: "user2",
		Chan:   make(chan []byte, 10),
		Done:   make(chan struct{}),
	}

	manager.AddClient("user1", client1)
	manager.AddClient("user2", client2)
	defer manager.RemoveClient("user1", client1)
	defer manager.RemoveClient("user2", client2)

	// Broadcast to multiple users
	event := Event{
		Type:    "notification",
		Payload: map[string]interface{}{"id": 1},
	}

	manager.BroadcastToMultiple([]string{"user1", "user2"}, event)

	// Verify both clients received the message
	select {
	case <-client1.Chan:
		// OK
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for message on client1")
	}

	select {
	case <-client2.Chan:
		// OK
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for message on client2")
	}
}
