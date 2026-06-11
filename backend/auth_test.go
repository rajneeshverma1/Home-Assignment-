package main

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// TestPasswordHashing verifies bcrypt password hashing and comparison
func TestPasswordHashing(t *testing.T) {
	password := "SecretP@ssword123"

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate password hash: %v", err)
	}

	// Verify correct password matches
	err = bcrypt.CompareHashAndPassword(hashed, []byte(password))
	if err != nil {
		t.Errorf("Expected password to match hash, got error: %v", err)
	}

	// Verify wrong password does not match
	err = bcrypt.CompareHashAndPassword(hashed, []byte("WrongPassword"))
	if err == nil {
		t.Errorf("Expected wrong password to fail matching")
	}
}

// TestJWTTokenGenerationAndValidation verifies token claims extraction and validation
func TestJWTTokenGenerationAndValidation(t *testing.T) {
	// Setup jwtKey manually for test environment
	jwtKey = []byte("my_testing_secret_key_for_jwt_test")

	userID := 42
	email := "test@example.com"
	role := "admin"

	token, err := generateToken(userID, email, role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if len(token) == 0 {
		t.Fatalf("Generated token is empty")
	}

	// Parse and validate the token
	claims := &Claims{}
	parsedToken, err := jwtParseHelper(token, claims)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !parsedToken.Valid {
		t.Errorf("Expected token to be valid")
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID to be %d, got %d", userID, claims.UserID)
	}

	if claims.Email != email {
		t.Errorf("Expected Email to be %s, got %s", email, claims.Email)
	}

	if claims.Role != role {
		t.Errorf("Expected Role to be %s, got %s", role, claims.Role)
	}

	// Check expiration is set in the future
	if claims.ExpiresAt.Before(time.Now()) {
		t.Errorf("Expected expiration to be in the future")
	}
}

// TestEmailRegexValidation verifies the email validator matches standard patterns
func TestEmailRegexValidation(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"john.doe@company.org",
		"a@b.co",
		"user_name.123@sub.domain.edu",
	}

	invalidEmails := []string{
		"",
		"plainaddress",
		"#@%^%#$@#$@#.com",
		"@example.com",
		"Joe Smith <email@example.com>",
		"email.example.com",
		"email@example@example.com",
		"email@example.123",
	}

	for _, email := range validEmails {
		if !emailRegex.MatchString(email) {
			t.Errorf("Expected email to be valid: %s", email)
		}
	}

	for _, email := range invalidEmails {
		if emailRegex.MatchString(email) {
			t.Errorf("Expected email to be invalid: %s", email)
		}
	}
}

// jwtParseHelper mocks jwt.ParseWithClaims parser logic for testing
func jwtParseHelper(tokenStr string, claims *Claims) (*ClaimsToken, error) {
	token, err := jwtParseFuncMock(tokenStr, claims)
	return &ClaimsToken{Valid: token.Valid}, err
}

type ClaimsToken struct {
	Valid bool
}

func jwtParseFuncMock(tokenStr string, claims *Claims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
}
