package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tullo/backend/internal/models"
	"github.com/tullo/backend/internal/repository"
)

type ConversationHandler struct {
	convRepo *repository.ConversationRepository
	userRepo *repository.UserRepository
	msgRepo  *repository.MessageRepository
}

func NewConversationHandler(
	convRepo *repository.ConversationRepository,
	userRepo *repository.UserRepository,
	msgRepo *repository.MessageRepository,
) *ConversationHandler {
	return &ConversationHandler{
		convRepo: convRepo,
		userRepo: userRepo,
		msgRepo:  msgRepo,
	}
}

// CreateConversation creates a new conversation
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	var req models.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	// For 1:1 conversations, check if it already exists
	if !req.IsGroup && len(req.Members) == 1 {
		conv, err := h.convRepo.GetOrCreateDirectConversation(uid, req.Members[0])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
			return
		}

		// Load members
		members, _ := h.convRepo.GetMembers(conv.ID)
		conv.Members = members

		c.JSON(http.StatusOK, conv)
		return
	}

	// Create group conversation
	conversation := &models.Conversation{
		ID:        uuid.New(),
		IsGroup:   req.IsGroup,
		Name:      req.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.convRepo.Create(conversation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	// Add creator as admin
	creatorMember := &models.ConversationMember{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		UserID:         uid,
		Role:           "admin",
		JoinedAt:       time.Now(),
	}
	h.convRepo.AddMember(creatorMember)

	// Add other members
	for _, memberID := range req.Members {
		if memberID == uid {
			continue
		}
		member := &models.ConversationMember{
			ID:             uuid.New(),
			ConversationID: conversation.ID,
			UserID:         memberID,
			Role:           "member",
			JoinedAt:       time.Now(),
		}
		h.convRepo.AddMember(member)
	}

	// Load members
	members, _ := h.convRepo.GetMembers(conversation.ID)
	conversation.Members = members

	c.JSON(http.StatusCreated, conversation)
}

// GetConversations returns all conversations for the current user
func (h *ConversationHandler) GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	conversations, err := h.convRepo.GetByUserID(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversations"})
		return
	}

	// Load members and last message for each conversation
	for i := range conversations {
		members, _ := h.convRepo.GetMembers(conversations[i].ID)
		conversations[i].Members = members

		// Get last message
		messages, _ := h.msgRepo.GetByConversationID(conversations[i].ID, 1, 0)
		if len(messages) > 0 {
			conversations[i].LastMessage = &messages[0]
		}
	}

	c.JSON(http.StatusOK, conversations)
}

// GetConversation returns a specific conversation
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	// Check if user is a member
	isMember, err := h.convRepo.IsMember(conversationID, uid)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	conversation, err := h.convRepo.GetByID(conversationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	// Load members
	members, _ := h.convRepo.GetMembers(conversation.ID)
	conversation.Members = members

	c.JSON(http.StatusOK, conversation)
}

// AddMembers adds members to a group conversation
func (h *ConversationHandler) AddMembers(c *gin.Context) {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req models.AddMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	// Check if user is a member
	isMember, err := h.convRepo.IsMember(conversationID, uid)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if it's a group conversation
	conversation, err := h.convRepo.GetByID(conversationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	if !conversation.IsGroup {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add members to 1:1 conversation"})
		return
	}

	// Add members
	for _, memberID := range req.Members {
		member := &models.ConversationMember{
			ID:             uuid.New(),
			ConversationID: conversationID,
			UserID:         memberID,
			Role:           "member",
			JoinedAt:       time.Now(),
		}
		h.convRepo.AddMember(member)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Members added successfully"})
}

// RemoveMember removes a member from a conversation
func (h *ConversationHandler) RemoveMember(c *gin.Context) {
	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	memberID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	// Check if user is a member
	isMember, err := h.convRepo.IsMember(conversationID, uid)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Remove member
	if err := h.convRepo.RemoveMember(conversationID, memberID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}
