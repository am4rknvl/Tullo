package handlers

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tullo/backend/internal/cache"
	"github.com/tullo/backend/internal/models"
	"github.com/tullo/backend/internal/repository"
)

type ChannelChatHandler struct {
	channelRepo *repository.ChannelRepository
	convRepo    *repository.ConversationRepository
	msgRepo     *repository.MessageRepository
	redis       *cache.RedisClient
	// in-memory limiter fallback (token-bucket per user)
	buckets   map[uuid.UUID]*tokenBucket
	bucketsMu sync.Mutex
	// bucket params (configurable)
	localRate  float64 // tokens per second
	localBurst float64 // capacity
}

func NewChannelChatHandler(chRepo *repository.ChannelRepository, convRepo *repository.ConversationRepository, msgRepo *repository.MessageRepository, redis *cache.RedisClient, localRate float64, localBurst float64) *ChannelChatHandler {
	h := &ChannelChatHandler{
		channelRepo: chRepo,
		convRepo:    convRepo,
		msgRepo:     msgRepo,
		redis:       redis,
		buckets:     make(map[uuid.UUID]*tokenBucket),
		localRate:   localRate,
		localBurst:  localBurst,
	}

	// start a background cleanup/refill goroutine
	go h.runRefillLoop()

	return h
}

// tokenBucket is a simple in-memory token bucket
type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
	rate       float64
	capacity   float64
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.rate
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		b.lastRefill = now
	}

	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}
	return false
}

func (h *ChannelChatHandler) runRefillLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		h.bucketsMu.Lock()
		now := time.Now()
		// refill each bucket; also remove stale buckets
		for uid, b := range h.buckets {
			b.mu.Lock()
			elapsed := now.Sub(b.lastRefill).Seconds()
			if elapsed > 0 {
				b.tokens += elapsed * b.rate
				if b.tokens > b.capacity {
					b.tokens = b.capacity
				}
				b.lastRefill = now
			}
			// remove bucket if unused for > 10 minutes to prevent leaks
			if now.Sub(b.lastRefill) > 10*time.Minute && b.tokens == b.capacity {
				delete(h.buckets, uid)
			}
			b.mu.Unlock()
		}
		h.bucketsMu.Unlock()
	}
}

// Get chat messages for channel
func (h *ChannelChatHandler) GetChat(c *gin.Context) {
	slug := c.Param("slug")
	ch, err := h.channelRepo.GetBySlug(slug)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "Channel not found")
		return
	}

	// find conversation id
	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get conversation")
		return
	}

	// parse query params
	limit := 50
	if l := c.Query("limit"); l != "" {
		if li, err := strconv.Atoi(l); err == nil {
			limit = li
		}
	}

	// support opaque cursor via message IDs: before_id / after_id
	var beforePtr *time.Time
	var afterPtr *time.Time
	if bs := c.Query("before_id"); bs != "" {
		if id, err := uuid.Parse(bs); err == nil {
			if m, err := h.msgRepo.GetByID(id); err == nil {
				beforePtr = &m.CreatedAt
			} else {
				ErrorResponse(c, http.StatusBadRequest, "invalid before_id")
				return
			}
		} else {
			ErrorResponse(c, http.StatusBadRequest, "invalid before_id")
			return
		}
	}
	if as := c.Query("after_id"); as != "" {
		if id, err := uuid.Parse(as); err == nil {
			if m, err := h.msgRepo.GetByID(id); err == nil {
				afterPtr = &m.CreatedAt
			} else {
				ErrorResponse(c, http.StatusBadRequest, "invalid after_id")
				return
			}
		} else {
			ErrorResponse(c, http.StatusBadRequest, "invalid after_id")
			return
		}
	}

	messages, err := h.msgRepo.GetByConversationIDCursor(convID, limit, beforePtr, afterPtr)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get messages")
		return
	}

	c.JSON(http.StatusOK, messages)
}

// Post chat message to channel
func (h *ChannelChatHandler) PostChat(c *gin.Context) {
	slug := c.Param("slug")
	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	convID, err := h.channelRepo.GetOrCreateConversation(ch.ID)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get conversation")
		return
	}

	// Moderation check: ensure user isn't muted/banned
	muted, banned, err := h.convRepo.IsUserMutedOrBanned(convID, uid)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to check moderation")
		return
	}
	if banned {
		ErrorResponse(c, http.StatusForbidden, "banned")
		return
	}
	if muted {
		ErrorResponse(c, http.StatusForbidden, "muted")
		return
	}

	// Rate limit: try Redis first
	allowed := true
	if h.redis != nil {
		ok, err := h.redis.AllowAction(uid, "channel_chat", int(h.localRate), int(h.localBurst))
		if err != nil {
			// fallback to local limiter if Redis errors
			allowed = false
		} else {
			allowed = ok
		}
	}

	if h.redis == nil || !allowed {
		// use in-memory token bucket fallback
		h.bucketsMu.Lock()
		b, ok := h.buckets[uid]
		if !ok {
			b = &tokenBucket{
				tokens:     h.localBurst,
				lastRefill: time.Now(),
				rate:       h.localRate,
				capacity:   h.localBurst,
			}
			h.buckets[uid] = b
		}
		h.bucketsMu.Unlock()

		if !b.allow() {
			ErrorResponse(c, http.StatusTooManyRequests, "rate_limited")
			return
		}
	}

	// create message
	message := &models.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		SenderID:       uid,
		Body:           req.Body,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.msgRepo.Create(message); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to send message")
		return
	}

	// publish via Redis (if available) for real-time broadcast
	if h.redis != nil {
		h.redis.PublishMessage(models.WSMessage{Event: models.EventMessageNew, Payload: message})
	}

	c.JSON(http.StatusCreated, message)
}
