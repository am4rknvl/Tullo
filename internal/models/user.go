package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	DisplayName  string    `json:"display_name" db:"display_name"`
	AvatarURL    *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Validate checks basic user fields
func (u *User) Validate() error {
	if u.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(u.Email, "@") {
		return fmt.Errorf("invalid email")
	}
	if u.DisplayName == "" {
		return fmt.Errorf("display name is required")
	}
	if len(u.DisplayName) < 2 || len(u.DisplayName) > 100 {
		return fmt.Errorf("display name length invalid")
	}
	return nil
}

type UserPresence struct {
	UserID   uuid.UUID `json:"user_id"`
	Status   string    `json:"status"` // online, offline
	LastSeen time.Time `json:"last_seen"`
}

type CreateUserRequest struct {
	Email       string  `json:"email" binding:"required,email"`
	Password    string  `json:"password" binding:"required,min=8"`
	DisplayName string  `json:"display_name" binding:"required"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
