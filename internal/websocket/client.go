package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tullo/backend/internal/cache"
	"github.com/tullo/backend/internal/models"
	"github.com/tullo/backend/internal/repository"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 10240 // 10KB
)

// Client represents a WebSocket client
type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	userID      uuid.UUID
	email       string
	connectedAt time.Time

	// Repositories
	msgRepo  *repository.MessageRepository
	convRepo *repository.ConversationRepository
	redis    *cache.RedisClient
	// simple token-bucket rate limiter
	tokens       int
	maxTokens    int
	refillPeriod time.Duration
	lastRefill   time.Time
}

// NewClient creates a new WebSocket client
func NewClient(
	hub *Hub,
	conn *websocket.Conn,
	userID uuid.UUID,
	email string,
	msgRepo *repository.MessageRepository,
	convRepo *repository.ConversationRepository,
	redis *cache.RedisClient,
) *Client {
	return &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, 256),
		userID:       userID,
		email:        email,
		connectedAt:  time.Now(),
		msgRepo:      msgRepo,
		convRepo:     convRepo,
		redis:        redis,
		tokens:       20,
		maxTokens:    20,
		refillPeriod: time.Second,
		lastRefill:   time.Now(),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Rate limit: simple token bucket (in-memory). If Redis present, you may implement a global limiter.
		now := time.Now()
		elapsed := now.Sub(c.lastRefill)
		if elapsed >= c.refillPeriod {
			// add tokens proportional to elapsed seconds
			add := int(elapsed / c.refillPeriod)
			c.tokens += add
			if c.tokens > c.maxTokens {
				c.tokens = c.maxTokens
			}
			c.lastRefill = now
		}

		if c.tokens <= 0 {
			// drop the message and optionally send a rate limit error
			c.sendError("rate_limited")
			continue
		}
		c.tokens--

		// Handle incoming message
		c.handleMessage(message)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages
func (c *Client) handleMessage(data []byte) {
	var wsMsg models.WSMessage
	if err := json.Unmarshal(data, &wsMsg); err != nil {
		c.sendError("Invalid message format")
		return
	}

	switch wsMsg.Event {
	case models.EventMessageSend:
		c.handleMessageSend(wsMsg.Payload)

	case models.EventMessageRead:
		c.handleMessageRead(wsMsg.Payload)

	case models.EventTypingStart:
		c.handleTypingStart(wsMsg.Payload)

	case models.EventTypingStop:
		c.handleTypingStop(wsMsg.Payload)

	default:
		c.sendError("Unknown event type")
	}
}

// handleMessageSend handles sending a message
func (c *Client) handleMessageSend(payload interface{}) {
	data, _ := json.Marshal(payload)
	var req models.WSMessageSendPayload
	if err := json.Unmarshal(data, &req); err != nil {
		c.sendError("Invalid message payload")
		return
	}

	// Check if user is a member of the conversation
	isMember, err := c.convRepo.IsMember(req.ConversationID, c.userID)
	if err != nil || !isMember {
		c.sendError("Access denied")
		return
	}

	// Create message
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: req.ConversationID,
		SenderID:       c.userID,
		Body:           req.Body,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := c.msgRepo.Create(message); err != nil {
		c.sendError("Failed to send message")
		return
	}

	// Publish to Redis for broadcast
	c.redis.PublishMessage(models.WSMessage{
		Event:   models.EventMessageNew,
		Payload: message,
	})
}

// handleMessageRead handles marking a message as read
func (c *Client) handleMessageRead(payload interface{}) {
	data, _ := json.Marshal(payload)
	var req models.WSMessageReadPayload
	if err := json.Unmarshal(data, &req); err != nil {
		c.sendError("Invalid read payload")
		return
	}

	// Mark message as read
	if err := c.msgRepo.MarkAsRead(req.MessageID, c.userID); err != nil {
		c.sendError("Failed to mark message as read")
		return
	}

	// Publish read receipt
	c.redis.PublishMessage(models.WSMessage{
		Event: models.EventMessageRead,
		Payload: map[string]interface{}{
			"message_id":      req.MessageID,
			"conversation_id": req.ConversationID,
			"user_id":         c.userID,
			"read_at":         time.Now(),
		},
	})
}

// handleTypingStart handles typing start event
func (c *Client) handleTypingStart(payload interface{}) {
	data, _ := json.Marshal(payload)
	var req models.WSTypingPayload
	if err := json.Unmarshal(data, &req); err != nil {
		c.sendError("Invalid typing payload")
		return
	}

	// Check if user is a member
	isMember, err := c.convRepo.IsMember(req.ConversationID, c.userID)
	if err != nil || !isMember {
		return
	}

	// Set typing in Redis
	c.redis.SetTyping(req.ConversationID, c.userID)

	// Publish typing indicator
	c.redis.PublishTyping(models.TypingIndicator{
		ConversationID: req.ConversationID,
		UserID:         c.userID,
		IsTyping:       true,
	})
}

// handleTypingStop handles typing stop event
func (c *Client) handleTypingStop(payload interface{}) {
	data, _ := json.Marshal(payload)
	var req models.WSTypingPayload
	if err := json.Unmarshal(data, &req); err != nil {
		c.sendError("Invalid typing payload")
		return
	}

	// Remove typing from Redis
	c.redis.RemoveTyping(req.ConversationID, c.userID)

	// Publish typing indicator
	c.redis.PublishTyping(models.TypingIndicator{
		ConversationID: req.ConversationID,
		UserID:         c.userID,
		IsTyping:       false,
	})
}

// sendError sends an error message to the client
func (c *Client) sendError(message string) {
	errorMsg := models.WSMessage{
		Event: models.EventError,
		Payload: models.WSErrorPayload{
			Message: message,
		},
	}

	data, _ := json.Marshal(errorMsg)
	select {
	case c.send <- data:
	default:
	}
}
