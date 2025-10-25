package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "mySecurePassword123"
	
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if hash == "" {
		t.Fatal("Expected hash to be generated")
	}

	if hash == password {
		t.Fatal("Hash should not equal plain password")
	}
}

func TestCheckPassword_Valid(t *testing.T) {
	password := "mySecurePassword123"
	
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	err = CheckPassword(hash, password)
	if err != nil {
		t.Errorf("Expected password to match, got error: %v", err)
	}
}

func TestCheckPassword_Invalid(t *testing.T) {
	password := "mySecurePassword123"
	wrongPassword := "wrongPassword"
	
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	err = CheckPassword(hash, wrongPassword)
	if err == nil {
		t.Error("Expected error for wrong password")
	}
}

func TestHashPassword_EmptyString(t *testing.T) {
	password := ""
	
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Expected no error for empty password, got %v", err)
	}

	if hash == "" {
		t.Fatal("Expected hash to be generated even for empty password")
	}
}
