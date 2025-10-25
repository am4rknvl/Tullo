package websocket

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tullo/backend/internal/auth"
	"github.com/tullo/backend/internal/cache"
	"github.com/tullo/backend/internal/repository"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate origin properly
		return true
	},
}

// Handler handles WebSocket connections
type Handler struct {
	hub        *Hub
	jwtService *auth.JWTService
	msgRepo    *repository.MessageRepository
	convRepo   *repository.ConversationRepository
	redis      *cache.RedisClient
}

// NewHandler creates a new WebSocket handler
func NewHandler(
	hub *Hub,
	jwtService *auth.JWTService,
	msgRepo *repository.MessageRepository,
	convRepo *repository.ConversationRepository,
	redis *cache.RedisClient,
) *Handler {
	return &Handler{
		hub:        hub,
		jwtService: jwtService,
		msgRepo:    msgRepo,
		convRepo:   convRepo,
		redis:      redis,
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
		return
	}

	// Validate token
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client
	client := NewClient(
		h.hub,
		conn,
		claims.UserID,
		claims.Email,
		h.msgRepo,
		h.convRepo,
		h.redis,
	)

	// Register client
	h.hub.register <- client

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()
}

// GetOnlineUsers returns online users (for testing/admin)
func (h *Handler) GetOnlineUsers(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	_ = userID.(uuid.UUID)

	onlineUsers := h.hub.GetOnlineUsers()
	c.JSON(http.StatusOK, gin.H{
		"online_users": onlineUsers,
		"count":        len(onlineUsers),
	})
}
