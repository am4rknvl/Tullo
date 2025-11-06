package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeClient is a minimal client that exposes a send channel
type fakeClient struct {
	userID uuid.UUID
	send   chan []byte
}

func (f *fakeClient) start() {}

func TestHubSendToUserAndConversation(t *testing.T) {
	h := &Hub{
		clients:    make(map[uuid.UUID]*Client),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client, 1),
		unregister: make(chan *Client, 1),
	}

	// create fake clients
	id1 := uuid.New()
	id2 := uuid.New()

	// Use actual Client struct but only use the send channel for assertion
	c1 := &Client{userID: id1, send: make(chan []byte, 4)}
	c2 := &Client{userID: id2, send: make(chan []byte, 4)}

	h.clients[id1] = c1
	h.clients[id2] = c2

	// Send to single user
	msg := map[string]string{"hello": "world"}
	if err := h.SendToUser(id1, msg); err != nil {
		t.Fatalf("SendToUser error: %v", err)
	}

	select {
	case b := <-c1.send:
		var got map[string]string
		json.Unmarshal(b, &got)
		if got["hello"] != "world" {
			t.Fatalf("unexpected payload: %v", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timed out waiting for message to user 1")
	}

	// Send to conversation (both clients)
	msg2 := map[string]string{"ping": "pong"}
	memberIDs := []uuid.UUID{id1, id2}
	if err := h.SendToConversation(memberIDs, msg2); err != nil {
		t.Fatalf("SendToConversation error: %v", err)
	}

	for _, c := range []*Client{c1, c2} {
		select {
		case b := <-c.send:
			var got map[string]string
			json.Unmarshal(b, &got)
			if got["ping"] != "pong" {
				t.Fatalf("unexpected payload: %v", got)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timed out waiting for conversation message")
		}
	}
}
