package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/redis"
	"github.com/example/taskmgr/internal/repository"
	"github.com/google/uuid"
)

type Hub struct {
	// Registered clients
	clients   map[*Client]bool
	clientsMu sync.RWMutex

	// Project subscriptions: ProjectID -> set of Clients
	projectSubs   map[uuid.UUID]map[*Client]bool
	projectSubsMu sync.RWMutex

	// User subscriptions: UserID -> set of Clients (for private notifications)
	userSubs   map[uuid.UUID]map[*Client]bool
	userSubsMu sync.RWMutex

	Register      chan *Client
	Unregister    chan *Client
	BroadcastChan chan *models.RealtimeEvent

	projectRepo *repository.ProjectRepository
	redisClient *redis.Client
	stopChan    chan struct{}
}

func NewHub(projectRepo *repository.ProjectRepository, redisClient *redis.Client) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		projectSubs:   make(map[uuid.UUID]map[*Client]bool),
		userSubs:      make(map[uuid.UUID]map[*Client]bool),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		BroadcastChan: make(chan *models.RealtimeEvent, 1024),
		projectRepo:   projectRepo,
		redisClient:   redisClient,
		stopChan:      make(chan struct{}),
	}
}

func (h *Hub) Run() {
	// Start Redis Pub/Sub listener if Redis client is available
	if h.redisClient != nil {
		go h.listenRedisPubSub()
	}

	for {
		select {
		case <-h.stopChan:
			h.cleanupAll()
			return

		case client := <-h.Register:
			h.clientsMu.Lock()
			h.clients[client] = true
			h.clientsMu.Unlock()

			h.userSubsMu.Lock()
			if h.userSubs[client.UserID] == nil {
				h.userSubs[client.UserID] = make(map[*Client]bool)
			}
			h.userSubs[client.UserID][client] = true
			h.userSubsMu.Unlock()

		case client := <-h.Unregister:
			h.clientsMu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.clientsMu.Unlock()

			h.userSubsMu.Lock()
			if subs, ok := h.userSubs[client.UserID]; ok {
				delete(subs, client)
				if len(subs) == 0 {
					delete(h.userSubs, client.UserID)
				}
			}
			h.userSubsMu.Unlock()

			// Remove client from all project subscriptions
			h.projectSubsMu.Lock()
			for projectID, subs := range h.projectSubs {
				delete(subs, client)
				if len(subs) == 0 {
					delete(h.projectSubs, projectID)
				}
			}
			h.projectSubsMu.Unlock()

			// Clean presence in Redis and notify other members
			client.CleanPresence()

		case event := <-h.BroadcastChan:
			h.deliverLocal(event)
		}
	}
}

func (h *Hub) Stop() {
	close(h.stopChan)
}

func (h *Hub) cleanupAll() {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	for client := range h.clients {
		close(client.Send)
		client.Conn.Close()
	}
	h.clients = make(map[*Client]bool)
}

// Subscribe authorizes project membership before attaching client to project events
func (h *Hub) Subscribe(client *Client, projectID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Verify project membership against database
	_, err := h.projectRepo.GetMemberRole(ctx, projectID, client.UserID)
	if err != nil {
		// Not authorized to subscribe to this project
		return
	}

	client.projectsMu.Lock()
	client.Projects[projectID] = true
	client.projectsMu.Unlock()

	h.projectSubsMu.Lock()
	if h.projectSubs[projectID] == nil {
		h.projectSubs[projectID] = make(map[*Client]bool)
	}
	h.projectSubs[projectID][client] = true
	h.projectSubsMu.Unlock()

	// Update presence lease in Redis
	if h.redisClient != nil {
		h.redisClient.UpdatePresence(ctx, projectID, client.UserID, 35*time.Second)

		// Broadcast presence online event
		h.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventPresenceUpdated,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload: map[string]interface{}{
				"user_id":   client.UserID,
				"is_online": true,
			},
		})
	}
}

func (h *Hub) Unsubscribe(client *Client, projectID uuid.UUID) {
	client.projectsMu.Lock()
	delete(client.Projects, projectID)
	client.projectsMu.Unlock()

	h.projectSubsMu.Lock()
	if subs, ok := h.projectSubs[projectID]; ok {
		delete(subs, client)
		if len(subs) == 0 {
			delete(h.projectSubs, projectID)
		}
	}
	h.projectSubsMu.Unlock()

	if h.redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.redisClient.RemovePresence(ctx, projectID, client.UserID)
	}
}

// Broadcast broadcasts to local clients AND publishes to Redis for horizontal scaling
func (h *Hub) Broadcast(event *models.RealtimeEvent) {
	if event == nil {
		return
	}

	// 1. Deliver locally
	h.BroadcastChan <- event

	// 2. Publish to Redis Pub/Sub for other backend instances
	if h.redisClient != nil {
		go func(ev *models.RealtimeEvent) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			h.redisClient.PublishEvent(ctx, ev)
		}(event)
	}
}

func (h *Hub) deliverLocal(event *models.RealtimeEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	var targets []*Client
	if event.TargetUserID != nil {
		h.userSubsMu.RLock()
		for client := range h.userSubs[*event.TargetUserID] {
			targets = append(targets, client)
		}
		h.userSubsMu.RUnlock()
	} else {
		h.projectSubsMu.RLock()
		for client := range h.projectSubs[event.ProjectID] {
			targets = append(targets, client)
		}
		h.projectSubsMu.RUnlock()
	}

	for _, client := range targets {
		select {
		case client.Send <- data:
		default:
			// Bounded queue full: drop client and cleanup
			go func(c *Client) {
				h.Unregister <- c
			}(client)
		}
	}
}

func (h *Hub) listenRedisPubSub() {
	ctx := context.Background()
	pubsub := h.redisClient.SubscribeEvents(ctx)
	if pubsub == nil {
		return
	}
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var event models.RealtimeEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err == nil {
			h.deliverLocal(&event)
		}
	}
}
