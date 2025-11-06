package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/tullo/backend/internal/cache"
	"github.com/tullo/backend/internal/models"
	"github.com/tullo/backend/internal/repository"
)

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients
	clients map[uuid.UUID]*Client

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Redis client for pub/sub
	redis *cache.RedisClient

	// Conversation repository to resolve members for conversation-scoped broadcasts
	convRepo *repository.ConversationRepository

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewHub creates a new Hub
func NewHub(redis *cache.RedisClient, convRepo *repository.ConversationRepository) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		redis:      redis,
		convRepo:   convRepo,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	// Subscribe to Redis channels
	go h.subscribeToRedis()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client
			h.mu.Unlock()

			// Set user online in Redis
			h.redis.SetUserOnline(client.userID)

			// Broadcast presence update
			presence := models.UserPresence{
				UserID:   client.userID,
				Status:   "online",
				LastSeen: client.connectedAt,
			}
			h.redis.PublishPresence(presence)

			log.Printf("Client registered: %s", client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
			h.mu.Unlock()

			// Set user offline in Redis
			h.redis.SetUserOffline(client.userID)

			// Broadcast presence update
			presence := models.UserPresence{
				UserID: client.userID,
				Status: "offline",
			}
			h.redis.PublishPresence(presence)

			log.Printf("Client unregistered: %s", client.userID)

		case message := <-h.broadcast:
			// Broadcast to all connected clients
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client.userID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// subscribeToRedis subscribes to Redis pub/sub channels
func (h *Hub) subscribeToRedis() {
	// Subscribe to messages channel
	msgPubSub := h.redis.SubscribeToMessages()
	defer msgPubSub.Close()

	msgChan := msgPubSub.Channel()

	// Subscribe to presence channel
	presencePubSub := h.redis.SubscribeToPresence()
	defer presencePubSub.Close()

	presenceChan := presencePubSub.Channel()

	// Subscribe to typing channel
	typingPubSub := h.redis.SubscribeToTyping()
	defer typingPubSub.Close()

	typingChan := typingPubSub.Channel()

	for {
		select {
		case msg := <-msgChan:
			// Try to unmarshal into WSMessage and handle conversation-scoped delivery
			var wsMsg models.WSMessage
			if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err == nil {
				// If it's a message event with a Message payload, attempt scoped delivery
				if wsMsg.Event == models.EventMessageNew {
					// payload may be a nested object; marshal/unmarshal to Message
					raw, _ := json.Marshal(wsMsg.Payload)
					var m models.Message
					if err := json.Unmarshal(raw, &m); err == nil {
						// resolve members for conversation
						members, err := h.convRepo.GetMembers(m.ConversationID)
						if err == nil {
							ids := make([]uuid.UUID, 0, len(members))
							for _, u := range members {
								ids = append(ids, u.ID)
							}
							// send to only conversation members
							h.SendToConversation(ids, wsMsg)
							continue
						}
					}
				}
			}

			// fallback: broadcast raw message to everyone
			h.broadcast <- []byte(msg.Payload)

		case presence := <-presenceChan:
			// Broadcast presence update
			h.broadcast <- []byte(presence.Payload)

		case typing := <-typingChan:
			// Broadcast typing indicator
			h.broadcast <- []byte(typing.Payload)
		}
	}
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID uuid.UUID, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		select {
		case client.send <- data:
		default:
			// Client's send channel is full, skip
		}
	}

	return nil
}

// SendToConversation sends a message to all members of a conversation
func (h *Hub) SendToConversation(memberIDs []uuid.UUID, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, memberID := range memberIDs {
		if client, ok := h.clients[memberID]; ok {
			select {
			case client.send <- data:
			default:
				// Client's send channel is full, skip
			}
		}
	}

	return nil
}

// GetOnlineUsers returns the list of online user IDs
func (h *Hub) GetOnlineUsers() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := make([]uuid.UUID, 0, len(h.clients))
	for userID := range h.clients {
		userIDs = append(userIDs, userID)
	}

	return userIDs
}

// IsUserOnline checks if a user is online
func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, ok := h.clients[userID]
	return ok
}
