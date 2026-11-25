package tests

import (
	"testing"
	"time"

	"github.com/example/taskmgr/internal/auth"
	"github.com/google/uuid"
)

func TestPasswordHashing(t *testing.T) {
	password := "SecretPass123!"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if hash == password {
		t.Fatalf("hash should not equal plain text password")
	}

	if !auth.CheckPasswordHash(password, hash) {
		t.Fatalf("expected password verification to succeed for correct password")
	}

	if auth.CheckPasswordHash("WrongPassword!", hash) {
		t.Fatalf("expected password verification to fail for incorrect password")
	}
}

func TestJWTGenerationAndValidation(t *testing.T) {
	userID := uuid.New()
	email := "test@example.com"
	secret := "test-super-secret-key-that-is-long-enough"
	duration := 1 * time.Hour

	token, err := auth.GenerateToken(userID, email, secret, duration)
	if err != nil {
		t.Fatalf("expected no error generating token, got %v", err)
	}

	claims, err := auth.ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Fatalf("expected email %s, got %s", email, claims.Email)
	}

	// Test invalid secret
	_, err = auth.ValidateToken(token, "wrong-secret-key-1234567890123456")
	if err == nil {
		t.Fatalf("expected error when validating with wrong secret")
	}

	// Test expired token
	expiredToken, err := auth.GenerateToken(userID, email, secret, -1*time.Minute)
	if err != nil {
		t.Fatalf("expected no error generating expired token, got %v", err)
	}

	_, err = auth.ValidateToken(expiredToken, secret)
	if err != auth.ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}
