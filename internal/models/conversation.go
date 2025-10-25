package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	IsGroup   bool       `json:"is_group" db:"is_group"`
	Name      *string    `json:"name,omitempty" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	Members   []User     `json:"members,omitempty"`
	LastMessage *Message `json:"last_message,omitempty"`
}

type ConversationMember struct {
	ID             uuid.UUID `json:"id" db:"id"`
	ConversationID uuid.UUID `json:"conversation_id" db:"conversation_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Role           string    `json:"role" db:"role"` // member, admin
	JoinedAt       time.Time `json:"joined_at" db:"joined_at"`
}

type CreateConversationRequest struct {
	IsGroup bool        `json:"is_group"`
	Name    *string     `json:"name,omitempty"`
	Members []uuid.UUID `json:"members" binding:"required,min=1"`
}

type AddMembersRequest struct {
	Members []uuid.UUID `json:"members" binding:"required,min=1"`
}

type ConversationWithDetails struct {
	Conversation
	UnreadCount int `json:"unread_count"`
}
