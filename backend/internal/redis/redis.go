package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func Connect(ctx context.Context, redisURL, password string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	if password != "" {
		opts.Password = password
	}

	rdb := redis.NewClient(opts)

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctxTimeout).Err(); err != nil {
		// Log warning rather than hard crash, so local dev or offline redis doesn't break everything
		log.Printf("Warning: failed to connect to Redis (%v). Operating with caching disabled.", err)
	} else {
		log.Println("Redis client connected successfully")
	}

	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() {
	if c != nil && c.rdb != nil {
		c.rdb.Close()
	}
}

// 1. Caching

func (c *Client) GetProject(ctx context.Context, projectID uuid.UUID) (*models.Project, error) {
	if c == nil || c.rdb == nil {
		return nil, fmt.Errorf("redis unavailable")
	}
	key := fmt.Sprintf("proj:%s", projectID.String())
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var proj models.Project
	if err := json.Unmarshal(data, &proj); err != nil {
		return nil, err
	}
	return &proj, nil
}

func (c *Client) SetProject(ctx context.Context, project *models.Project, ttl time.Duration) error {
	if c == nil || c.rdb == nil || project == nil {
		return nil
	}
	key := fmt.Sprintf("proj:%s", project.ID.String())
	data, err := json.Marshal(project)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *Client) InvalidateProject(ctx context.Context, projectID uuid.UUID) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	keyProj := fmt.Sprintf("proj:%s", projectID.String())
	keyMembers := fmt.Sprintf("proj:%s:members", projectID.String())
	return c.rdb.Del(ctx, keyProj, keyMembers).Err()
}

func (c *Client) GetMembers(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectMember, error) {
	if c == nil || c.rdb == nil {
		return nil, fmt.Errorf("redis unavailable")
	}
	key := fmt.Sprintf("proj:%s:members", projectID.String())
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var members []*models.ProjectMember
	if err := json.Unmarshal(data, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (c *Client) SetMembers(ctx context.Context, projectID uuid.UUID, members []*models.ProjectMember, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	key := fmt.Sprintf("proj:%s:members", projectID.String())
	data, err := json.Marshal(members)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// 2. Online Presence

func (c *Client) UpdatePresence(ctx context.Context, projectID, userID uuid.UUID, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	leaseKey := fmt.Sprintf("presence:%s:%s", projectID.String(), userID.String())
	setKey := fmt.Sprintf("presence:%s:set", projectID.String())
	now := float64(time.Now().Unix())

	pipe := c.rdb.Pipeline()
	pipe.Set(ctx, leaseKey, now, ttl)
	pipe.ZAdd(ctx, setKey, redis.Z{
		Score:  now,
		Member: userID.String(),
	})
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Client) RemovePresence(ctx context.Context, projectID, userID uuid.UUID) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	leaseKey := fmt.Sprintf("presence:%s:%s", projectID.String(), userID.String())
	setKey := fmt.Sprintf("presence:%s:set", projectID.String())

	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, leaseKey)
	pipe.ZRem(ctx, setKey, userID.String())
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Client) GetOnlineUsers(ctx context.Context, projectID uuid.UUID, thresholdSeconds int64) ([]uuid.UUID, error) {
	if c == nil || c.rdb == nil {
		return nil, nil
	}
	setKey := fmt.Sprintf("presence:%s:set", projectID.String())
	minScore := fmt.Sprintf("%d", time.Now().Unix()-thresholdSeconds)
	maxScore := "+inf"

	// Fetch active members within threshold
	userIDsStr, err := c.rdb.ZRangeByScore(ctx, setKey, &redis.ZRangeBy{
		Min: minScore,
		Max: maxScore,
	}).Result()
	if err != nil {
		return nil, err
	}

	// Clean up stale members from sorted set
	c.rdb.ZRemRangeByScore(ctx, setKey, "-inf", fmt.Sprintf("(%d", time.Now().Unix()-thresholdSeconds))

	var userIDs []uuid.UUID
	for _, idStr := range userIDsStr {
		if uid, err := uuid.Parse(idStr); err == nil {
			userIDs = append(userIDs, uid)
		}
	}

	return userIDs, nil
}

// 3. Pub/Sub for Real-Time Event Distribution across instances

const EventChannel = "taskmgr:events"

func (c *Client) PublishEvent(ctx context.Context, event *models.RealtimeEvent) error {
	if c == nil || c.rdb == nil || event == nil {
		return nil
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return c.rdb.Publish(ctx, EventChannel, data).Err()
}

func (c *Client) SubscribeEvents(ctx context.Context) *redis.PubSub {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Subscribe(ctx, EventChannel)
}
