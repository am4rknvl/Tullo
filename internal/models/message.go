package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ConversationID uuid.UUID  `json:"conversation_id" db:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id" db:"sender_id"`
	Body           string     `json:"body" db:"body"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	Sender         *User      `json:"sender,omitempty"`
}

type MessageRead struct {
	ID        uuid.UUID `json:"id" db:"id"`
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ReadAt    time.Time `json:"read_at" db:"read_at"`
}

type SendMessageRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" binding:"required"`
	Body           string    `json:"body" binding:"required,max=10000"`
}

type GetMessagesRequest struct {
	ConversationID uuid.UUID `form:"conversation_id" binding:"required"`
	Limit          int       `form:"limit"`
	Offset         int       `form:"offset"`
}

type MarkReadRequest struct {
	MessageID      uuid.UUID `json:"message_id" binding:"required"`
	ConversationID uuid.UUID `json:"conversation_id" binding:"required"`
}

type TypingIndicator struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	IsTyping       bool      `json:"is_typing"`
}
