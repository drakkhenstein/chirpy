package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWT(t *testing.T) {
	// Test data
	userID := "123e4567-e89b-12d3-a456-426614174000"
	tokenSecret := "mysecretkey"
	expiresIn := 1 * time.Hour

	// Create JWT
	token, err := MakeJWT(uuid.MustParse(userID), tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Validate JWT
	returnedUserID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if returnedUserID.String() != userID {
		t.Errorf("Expected userID %s, got %s", userID, returnedUserID.String())
	}
}

func TestJWTExpired(t *testing.T) {
	// Test data
	userID := "123e4567-e89b-12d3-a456-426614174000"
	tokenSecret := "mysecretkey"
	expiresIn := -1 * time.Hour // Expired token

	// Create JWT
	token, err := MakeJWT(uuid.MustParse(userID), tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Validate JWT
	_, err = ValidateJWT(token, tokenSecret)
	if err == nil {
		t.Fatal("Expected error for expired token, got nil")
	}
}

func TestJWTInvalidSecret(t *testing.T) {
	// Test data
	userID := "123e4567-e89b-12d3-a456-426614174000"
	tokenSecret := "mysecretkey"
	invalidSecret := "invalidsecretkey"
	expiresIn := 1 * time.Hour

	// Create JWT
	token, err := MakeJWT(uuid.MustParse(userID), tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Validate JWT with invalid secret
	_, err = ValidateJWT(token, invalidSecret)
	if err == nil {
		t.Fatal("Expected error for invalid secret, got nil")
	}
}