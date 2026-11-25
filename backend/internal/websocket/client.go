package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/redis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536 // 64KB
)

type Client struct {
	Hub         *Hub
	Conn        *websocket.Conn
	UserID      uuid.UUID
	Send        chan []byte
	Projects    map[uuid.UUID]bool
	projectsMu  sync.RWMutex
	redisClient *redis.Client
}

type ClientAction struct {
	Action    string    `json:"action"` // "subscribe" or "unsubscribe"
	ProjectID uuid.UUID `json:"projectId"`
}

func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, redisClient *redis.Client) *Client {
	return &Client{
		Hub:         hub,
		Conn:        conn,
		UserID:      userID,
		Send:        make(chan []byte, 256),
		Projects:    make(map[uuid.UUID]bool),
		redisClient: redisClient,
	}
}

func (c *Client) IsSubscribed(projectID uuid.UUID) bool {
	c.projectsMu.RLock()
	defer c.projectsMu.RUnlock()
	return c.Projects[projectID]
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		// Renew presence for subscribed projects
		c.refreshPresence()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS read error for user %s: %v", c.UserID, err)
			}
			break
		}

		var action ClientAction
		if err := json.Unmarshal(message, &action); err == nil {
			switch action.Action {
			case "subscribe":
				c.Hub.Subscribe(c, action.ProjectID)
			case "unsubscribe":
				c.Hub.Unsubscribe(c, action.ProjectID)
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed by hub
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain queued messages to reduce syscall overhead
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) refreshPresence() {
	if c.redisClient == nil {
		return
	}
	c.projectsMu.RLock()
	defer c.projectsMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for projectID := range c.Projects {
		c.redisClient.UpdatePresence(ctx, projectID, c.UserID, 35*time.Second)
	}
}

func (c *Client) CleanPresence() {
	if c.redisClient == nil {
		return
	}
	c.projectsMu.RLock()
	defer c.projectsMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for projectID := range c.Projects {
		c.redisClient.RemovePresence(ctx, projectID, c.UserID)
		// Broadcast offline presence event
		event := &models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventPresenceUpdated,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload: map[string]interface{}{
				"user_id":   c.UserID,
				"is_online": false,
			},
		}
		c.Hub.Broadcast(event)
	}
}
