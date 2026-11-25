package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/example/taskmgr/internal/auth"
	"github.com/example/taskmgr/internal/config"
	"github.com/example/taskmgr/internal/redis"
	"github.com/example/taskmgr/internal/websocket"
	ws "github.com/gorilla/websocket"
)

type WSHandler struct {
	hub         *websocket.Hub
	cfg         *config.Config
	redisClient *redis.Client
	upgrader    ws.Upgrader
}

func NewWSHandler(hub *websocket.Hub, cfg *config.Config, redisClient *redis.Client) *WSHandler {
	return &WSHandler{
		hub:         hub,
		cfg:         cfg,
		redisClient: redisClient,
		upgrader: ws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				for _, allowed := range cfg.CORSAllowedOrigins {
					if strings.EqualFold(allowed, "*") || strings.EqualFold(allowed, origin) {
						return true
					}
				}
				return false // deny unrecognised origins
			},
		},
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticate the connection
	tokenString := extractWSToken(r)
	if tokenString == "" {
		RespondError(w, http.StatusUnauthorized, "unauthorized", "missing authentication token for websocket handshake")
		return
	}

	claims, err := auth.ValidateToken(tokenString, h.cfg.JWTSecret)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade websocket connection: %v", err)
		return
	}

	client := websocket.NewClient(h.hub, conn, claims.UserID, h.redisClient)
	h.hub.Register <- client

	// Start goroutines for reading and writing
	go client.WritePump()
	go client.ReadPump()
}

func extractWSToken(r *http.Request) string {
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	if cookie, err := r.Cookie("task_jwt"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}
