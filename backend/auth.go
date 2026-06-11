package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	userIDKey    contextKey = "userID"
	userRoleKey  contextKey = "userRole"
	userEmailKey contextKey = "userEmail"
)

var jwtKey []byte

func InitJWT() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_fallback_jwt_secret_key_12345"
		log.Println("WARNING: JWT_SECRET not set, using default development fallback key")
	}
	jwtKey = []byte(secret)
}

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}
