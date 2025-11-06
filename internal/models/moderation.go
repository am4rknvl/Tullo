package models

import (
	"time"

	"github.com/google/uuid"
)

// ModerationLog records actions taken by moderators or the bot
type ModerationLog struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	ConversationID *uuid.UUID     `json:"conversation_id,omitempty" db:"conversation_id"`
	MessageID      *uuid.UUID     `json:"message_id,omitempty" db:"message_id"`
	Action         string         `json:"action" db:"action"` // delete, warn, timeout, ban, mute
	ModeratorID    *uuid.UUID     `json:"moderator_id,omitempty" db:"moderator_id"`
	TargetUserID   *uuid.UUID     `json:"target_user_id,omitempty" db:"target_user_id"`
	Reason         *string        `json:"reason,omitempty" db:"reason"`
	Metadata       map[string]any `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}

// BannedWord represents a custom banned word for a conversation (channel)
type BannedWord struct {
	ID             uuid.UUID `json:"id" db:"id"`
	ConversationID uuid.UUID `json:"conversation_id" db:"conversation_id"`
	Word           string    `json:"word" db:"word"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
