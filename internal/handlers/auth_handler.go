package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tullo/backend/internal/auth"
	"github.com/tullo/backend/internal/models"
	"github.com/tullo/backend/internal/repository"
)

type AuthHandler struct {
	userRepo   *repository.UserRepository
	jwtService *auth.JWTService
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Create user
	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		AvatarURL:    req.AvatarURL,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.userRepo.Create(user); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate token
	token, err := h.jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	c.JSON(http.StatusCreated, models.LoginResponse{
		Token: token,
		User:  *user,
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get user by email
	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check password
	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate token
	token, err := h.jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  *user,
	})
}

// GetMe returns the current user
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uuid.UUID)

	user, err := h.userRepo.GetByID(uid)
	if err != nil {
		ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	c.JSON(http.StatusOK, user)
}
