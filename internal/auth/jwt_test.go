package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTService_GenerateToken(t *testing.T) {
	secret := "test-secret-key"
	expiryHours := 24
	service := NewJWTService(secret, expiryHours)

	userID := uuid.New()
	token, err := service.GenerateToken(userID, "test@example.com")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Fatal("Expected token to be generated")
	}
}

func TestJWTService_ValidateToken(t *testing.T) {
	secret := "test-secret-key"
	expiryHours := 24
	service := NewJWTService(secret, expiryHours)

	userID := uuid.New()
	token, err := service.GenerateToken(userID, "test@example.com")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID.String(), claims.UserID.String())
	}
}

func TestJWTService_ValidateToken_Invalid(t *testing.T) {
	secret := "test-secret-key"
	expiryHours := 24
	service := NewJWTService(secret, expiryHours)

	invalidToken := "invalid.token.here"
	_, err := service.ValidateToken(invalidToken)

	if err == nil {
		t.Fatal("Expected error for invalid token")
	}
}

func TestJWTService_ValidateToken_Expired(t *testing.T) {
	secret := "test-secret-key"
	expiryHours := -1 // Expired token
	service := NewJWTService(secret, expiryHours)

	userID := uuid.New()
	token, err := service.GenerateToken(userID, "test@example.com")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait a moment to ensure expiry
	time.Sleep(time.Millisecond * 100)

	_, err = service.ValidateToken(token)
	if err == nil {
		t.Fatal("Expected error for expired token")
	}
}
