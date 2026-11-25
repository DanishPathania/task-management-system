package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/example/taskmgr/internal/auth"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
)

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString == "" {
				respondUnauthorized(w, "missing authentication token")
				return
			}

			claims, err := auth.ValidateToken(tokenString, jwtSecret)
			if err != nil {
				respondUnauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) string {
	// 1. Check Authorization Bearer header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// 2. Check HttpOnly cookie
	if cookie, err := r.Cookie("task_jwt"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 3. Check query param (e.g. for browser WebSocket handshake)
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	return ""
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return id, ok
}

func GetUserEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}

func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "unauthorized",
		"message": message,
	})
}
