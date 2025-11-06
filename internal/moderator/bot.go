package moderator

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tullo/backend/internal/cache"
	"github.com/tullo/backend/internal/models"
	"github.com/tullo/backend/internal/repository"
)

// Bot monitors messages and enforces moderation rules
type Bot struct {
	redis    *cache.RedisClient
	convRepo *repository.ConversationRepository
	msgRepo  *repository.MessageRepository
	modRepo  *repository.ModerationRepository
	userRepo *repository.UserRepository
	botUser  uuid.UUID

	// simple in-memory recent messages for spam detection
	recentMu sync.Mutex
	recent   map[uuid.UUID][]recentMsg // key: userID
}

type recentMsg struct {
	body string
	ts   time.Time
}

// NewBot creates a new moderation bot instance
func NewBot(redis *cache.RedisClient, convRepo *repository.ConversationRepository, msgRepo *repository.MessageRepository, modRepo *repository.ModerationRepository, userRepo *repository.UserRepository, botUser uuid.UUID) *Bot {
	return &Bot{
		redis:    redis,
		convRepo: convRepo,
		msgRepo:  msgRepo,
		modRepo:  modRepo,
		userRepo: userRepo,
		botUser:  botUser,
		recent:   make(map[uuid.UUID][]recentMsg),
	}
}

// Run starts listening for messages and processing them
func (b *Bot) Run() {
	if b.redis == nil {
		log.Println("Moderation bot requires Redis; not started")
		return
	}

	ps := b.redis.SubscribeToMessages()
	defer ps.Close()

	ch := ps.Channel()
	log.Println("Moderation bot started and listening to messages")
	for msg := range ch {
		var ws models.WSMessage
		if err := json.Unmarshal([]byte(msg.Payload), &ws); err != nil {
			continue
		}
		if ws.Event != models.EventMessageNew {
			continue
		}
		// payload -> message
		raw, _ := json.Marshal(ws.Payload)
		var m models.Message
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}

		go b.processMessage(&m)
	}
}

func (b *Bot) processMessage(m *models.Message) {
	// quick checks
	// 1. check banned words for conversation
	bannedWords, err := b.modRepo.GetBannedWords(m.ConversationID)
	if err == nil && len(bannedWords) > 0 {
		lower := strings.ToLower(m.Body)
		for _, bw := range bannedWords {
			if strings.Contains(lower, strings.ToLower(bw.Word)) {
				// delete message
				_ = b.msgRepo.Delete(m.ID)
				// log action
				logEntry := &models.ModerationLog{
					ID:             uuid.New(),
					ConversationID: &m.ConversationID,
					MessageID:      &m.ID,
					Action:         "delete_word",
					ModeratorID:    &b.botUser,
					TargetUserID:   &m.SenderID,
					Reason:         &bw.Word,
					CreatedAt:      time.Now(),
				}
				_ = b.modRepo.AddLog(logEntry)
				return
			}
		}
	}

	// 2. simple spam detection: repeated identical messages within 10s window
	b.recentMu.Lock()
	arr := b.recent[m.SenderID]
	now := time.Now()
	// prune old
	newArr := []recentMsg{}
	repeatCount := 0
	for _, rm := range arr {
		if now.Sub(rm.ts) <= 10*time.Second {
			newArr = append(newArr, rm)
			if rm.body == m.Body {
				repeatCount++
			}
		}
	}
	newArr = append(newArr, recentMsg{body: m.Body, ts: now})
	b.recent[m.SenderID] = newArr
	b.recentMu.Unlock()

	if repeatCount >= 3 {
		// timeout user for 5 minutes
		convID := m.ConversationID
		exp := time.Now().Add(5 * time.Minute)
		_ = b.convRepo.AddModeration(convID, m.SenderID, "mute", &exp, "spam: repeated messages")
		logEntry := &models.ModerationLog{
			ID:             uuid.New(),
			ConversationID: &convID,
			MessageID:      &m.ID,
			Action:         "timeout_spam",
			ModeratorID:    &b.botUser,
			TargetUserID:   &m.SenderID,
			Reason:         ptrString("spam repeated"),
			CreatedAt:      time.Now(),
		}
		_ = b.modRepo.AddLog(logEntry)
		// delete offending message
		_ = b.msgRepo.Delete(m.ID)
		return
	}

	// 3. placeholder for harmful language detection (future AI integration)
	// For now, simple profanity list can be global; omitted here.
}

func ptrString(s string) *string { return &s }
