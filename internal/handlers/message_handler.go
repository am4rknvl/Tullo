package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tullo/backend/internal/cache"
	"github.com/tullo/backend/internal/models"
	"github.com/tullo/backend/internal/repository"
)

type MessageHandler struct {
	msgRepo  *repository.MessageRepository
	convRepo *repository.ConversationRepository
	redis    *cache.RedisClient
}

func NewMessageHandler(
	msgRepo *repository.MessageRepository,
	convRepo *repository.ConversationRepository,
	redis *cache.RedisClient,
) *MessageHandler {
	return &MessageHandler{
		msgRepo:  msgRepo,
		convRepo: convRepo,
		redis:    redis,
	}
}

// GetMessages returns messages for a conversation
func (h *MessageHandler) GetMessages(c *gin.Context) {
	var req models.GetMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	// Check if user is a member
	isMember, err := h.convRepo.IsMember(req.ConversationID, uid)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 50
	}

	messages, err := h.msgRepo.GetByConversationID(req.ConversationID, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// SendMessage sends a new message (REST endpoint)
func (h *MessageHandler) SendMessage(c *gin.Context) {
	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	// Check if user is a member
	isMember, err := h.convRepo.IsMember(req.ConversationID, uid)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Create message
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: req.ConversationID,
		SenderID:       uid,
		Body:           req.Body,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.msgRepo.Create(message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Publish to Redis for WebSocket broadcast
	h.redis.PublishMessage(models.WSMessage{
		Event:   models.EventMessageNew,
		Payload: message,
	})

	c.JSON(http.StatusCreated, message)
}

// MarkMessageAsRead marks a message as read
func (h *MessageHandler) MarkMessageAsRead(c *gin.Context) {
	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	// Get message to verify conversation membership
	message, err := h.msgRepo.GetByID(messageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	// Check if user is a member
	isMember, err := h.convRepo.IsMember(message.ConversationID, uid)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Mark as read
	if err := h.msgRepo.MarkAsRead(messageID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark message as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message marked as read"})
}
