package models

import (
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OwnerID     uuid.UUID `json:"owner_id" db:"owner_id"`
	Slug        string    `json:"slug" db:"slug"`
	Title       string    `json:"title" db:"title"`
	Description *string   `json:"description,omitempty" db:"description"`
	Language    *string   `json:"language,omitempty" db:"language"`
	Tags        []string  `json:"tags,omitempty" db:"tags"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CreateChannelRequest struct {
	Title       string   `json:"title" binding:"required"`
	Slug        string   `json:"slug" binding:"required"`
	Description *string  `json:"description,omitempty"`
	Language    *string  `json:"language,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
