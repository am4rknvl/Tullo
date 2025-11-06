package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tullo/backend/internal/models"
	"github.com/tullo/backend/internal/repository"
)

type ChannelHandler struct {
	channelRepo *repository.ChannelRepository
	streamRepo  *repository.StreamRepository
	convRepo    *repository.ConversationRepository
	userRepo    *repository.UserRepository
	modRepo     *repository.ModerationRepository
}

func NewChannelHandler(chRepo *repository.ChannelRepository, sRepo *repository.StreamRepository, convRepo *repository.ConversationRepository, userRepo *repository.UserRepository, modRepo *repository.ModerationRepository) *ChannelHandler {
	return &ChannelHandler{channelRepo: chRepo, streamRepo: sRepo, convRepo: convRepo, userRepo: userRepo, modRepo: modRepo}
}

// Create channel
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req models.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch := &models.Channel{
		ID:          uuid.New(),
		OwnerID:     uid,
		Slug:        req.Slug,
		Title:       req.Title,
		Description: req.Description,
		Language:    req.Language,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.channelRepo.Create(ch); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create channel")
		return
	}

	// Ensure a conversation exists for the channel and add owner as moderator
	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create or get channel conversation")
		return
	}

	// Add owner as conversation member with role 'moderator'
	member := &models.ConversationMember{
		ID:             uuid.New(),
		ConversationID: convID,
		UserID:         uid,
		Role:           "moderator",
		JoinedAt:       time.Now(),
	}
	if err := h.convRepo.AddMember(member); err != nil {
		// Log but do not fail channel creation
		// use ErrorResponse? keep channel creation successful
	}

	// Add TulloBot as moderator if available
	if h.userRepo != nil {
		botEmail := "tullo-bot@tullo.local"
		if bot, err := h.userRepo.GetByEmail(botEmail); err == nil {
			botMember := &models.ConversationMember{
				ID:             uuid.New(),
				ConversationID: convID,
				UserID:         bot.ID,
				Role:           "moderator",
				JoinedAt:       time.Now(),
			}
			_ = h.convRepo.AddMember(botMember)
		}
	}

	c.JSON(http.StatusCreated, ch)
}

// Get channel by slug
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	slug := c.Param("slug")
	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}

	// attach latest stream info if any
	stream, _ := h.streamRepo.GetByChannel(ch.ID)
	c.JSON(http.StatusOK, gin.H{"channel": ch, "stream": stream})
}

// StartStream starts a new stream for the channel. Only owner can start.
func (h *ChannelHandler) StartStream(c *gin.Context) {
	slug := c.Param("slug")
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}
	if ch.OwnerID != uid {
		ErrorResponse(c, http.StatusForbidden, "only owner can start stream")
		return
	}

	now := time.Now()
	key := uuid.New().String()
	s := &models.Stream{
		ID:        uuid.New(),
		ChannelID: ch.ID,
		Status:    "live",
		StreamKey: &key,
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.streamRepo.Create(s); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to start stream")
		return
	}

	c.JSON(http.StatusCreated, s)
}

// EndStream ends the active stream. Owner or moderator can end.
func (h *ChannelHandler) EndStream(c *gin.Context) {
	slug := c.Param("slug")
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}

	// allow owner or moderator
	if ch.OwnerID != uid {
		// check moderator role via conversation
		convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
		if err != nil {
			ErrorResponse(c, http.StatusInternalServerError, "failed to check permissions")
			return
		}
		role, err := h.convRepo.GetMemberRole(convID, uid)
		if err != nil || (role != "moderator" && role != "admin") {
			ErrorResponse(c, http.StatusForbidden, "only owner/moderator can end stream")
			return
		}
	}

	stream, err := h.streamRepo.GetByChannel(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "no active stream found")
		return
	}
	now := time.Now()
	if err := h.streamRepo.EndStream(stream.ID, now); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to end stream")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "stream ended"})
}

// GetActiveStreams returns currently live streams for the explore page
func (h *ChannelHandler) GetActiveStreams(c *gin.Context) {
	limit := 50
	streams, err := h.streamRepo.GetActiveStreams(limit)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to get active streams")
		return
	}
	c.JSON(http.StatusOK, streams)
}

// FollowChannel: authenticated user follows a channel
func (h *ChannelHandler) FollowChannel(c *gin.Context) {
	slug := c.Param("slug")
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}
	if err := h.channelRepo.AddFollower(ch.ID, uid); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to follow channel")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "followed"})
}

// UnfollowChannel: authenticated user unfollows a channel
func (h *ChannelHandler) UnfollowChannel(c *gin.Context) {
	slug := c.Param("slug")
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}
	if err := h.channelRepo.RemoveFollower(ch.ID, uid); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to unfollow channel")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}

// AssignModerator: owner assigns a moderator role to a user for channel
func (h *ChannelHandler) AssignModerator(c *gin.Context) {
	slug := c.Param("slug")
	var body struct {
		UserID uuid.UUID `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}
	if ch.OwnerID != uid {
		ErrorResponse(c, http.StatusForbidden, "only owner can assign moderators")
		return
	}

	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to get conversation")
		return
	}
	if err := h.convRepo.UpdateMemberRole(convID, body.UserID, "moderator"); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to assign moderator")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "moderator assigned"})
}

// RemoveModerator: owner removes moderator role (demote to member)
func (h *ChannelHandler) RemoveModerator(c *gin.Context) {
	slug := c.Param("slug")
	userIDParam := c.Param("user_id")
	targetID, err := uuid.Parse(userIDParam)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid user id")
		return
	}
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}
	if ch.OwnerID != uid {
		ErrorResponse(c, http.StatusForbidden, "only owner can remove moderators")
		return
	}

	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to get conversation")
		return
	}
	if err := h.convRepo.UpdateMemberRole(convID, targetID, "member"); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to remove moderator")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "moderator removed"})
}

// BanUser bans a user from the channel (owner/mod)
func (h *ChannelHandler) BanUser(c *gin.Context) {
	slug := c.Param("slug")
	targetIDParam := c.Param("user_id")
	targetID, err := uuid.Parse(targetIDParam)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var body struct {
		DurationMin int    `json:"duration_min"`
		Reason      string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		// allow empty body
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}

	// check owner or moderator
	allowed := false
	if ch.OwnerID == uid {
		allowed = true
	} else {
		convID, _ := h.channelRepo.GetOrCreateConversation(ch.ID)
		role, _ := h.convRepo.GetMemberRole(convID, uid)
		if role == "moderator" || role == "admin" {
			allowed = true
		}
	}
	if !allowed {
		ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to get conversation")
		return
	}

	var expires *time.Time
	if body.DurationMin > 0 {
		t := time.Now().Add(time.Duration(body.DurationMin) * time.Minute)
		expires = &t
	}
	if err := h.convRepo.AddModeration(convID, targetID, "ban", expires, body.Reason); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to ban user")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user banned"})
}

// UnbanUser removes ban (owner/mod)
func (h *ChannelHandler) UnbanUser(c *gin.Context) {
	slug := c.Param("slug")
	targetIDParam := c.Param("user_id")
	targetID, err := uuid.Parse(targetIDParam)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid user id")
		return
	}

	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}

	// check owner or moderator
	allowed := false
	if ch.OwnerID == uid {
		allowed = true
	} else {
		convID, _ := h.channelRepo.GetOrCreateConversation(ch.ID)
		role, _ := h.convRepo.GetMemberRole(convID, uid)
		if role == "moderator" || role == "admin" {
			allowed = true
		}
	}
	if !allowed {
		ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to get conversation")
		return
	}
	if err := h.convRepo.RemoveModeration(convID, targetID, "ban"); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to unban user")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user unbanned"})
}

// Banned words management
// AddBannedWord: owner/mod can add a custom banned word for the channel
func (h *ChannelHandler) AddBannedWord(c *gin.Context) {
	slug := c.Param("slug")
	var body struct {
		Word string `json:"word"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}

	// only owner or moderator can add
	allowed := false
	if ch.OwnerID == uid {
		allowed = true
	} else {
		convID, _ := h.channelRepo.GetOrCreateConversation(ch.ID)
		role, _ := h.convRepo.GetMemberRole(convID, uid)
		if role == "moderator" || role == "admin" {
			allowed = true
		}
	}
	if !allowed {
		ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to get conversation")
		return
	}
	if err := h.modRepo.AddBannedWord(convID, body.Word); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to add banned word")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "banned word added"})
}

// RemoveBannedWord removes a banned word
func (h *ChannelHandler) RemoveBannedWord(c *gin.Context) {
	slug := c.Param("slug")
	word := c.Param("word")
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}

	// only owner can remove
	if ch.OwnerID != uid {
		ErrorResponse(c, http.StatusForbidden, "only owner can remove banned words")
		return
	}
	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to get conversation")
		return
	}
	if err := h.modRepo.RemoveBannedWord(convID, word); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to remove banned word")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "banned word removed"})
}

// ListBannedWords lists custom banned words for a channel
func (h *ChannelHandler) ListBannedWords(c *gin.Context) {
	slug := c.Param("slug")
	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}
	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to get conversation")
		return
	}
	words, err := h.modRepo.GetBannedWords(convID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "failed to list banned words")
		return
	}
	c.JSON(http.StatusOK, words)
}
