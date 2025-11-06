package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/tullo/backend/internal/models"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient creates a new Redis client
func NewRedisClient(addr, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Presence Management

// SetUserOnline sets a user as online
func (r *RedisClient) SetUserOnline(userID uuid.UUID) error {
	key := fmt.Sprintf("presence:user:%s", userID.String())
	presence := models.UserPresence{
		UserID:   userID,
		Status:   "online",
		LastSeen: time.Now(),
	}

	data, err := json.Marshal(presence)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, data, 5*time.Minute).Err()
}

// SetUserOffline sets a user as offline
func (r *RedisClient) SetUserOffline(userID uuid.UUID) error {
	key := fmt.Sprintf("presence:user:%s", userID.String())
	presence := models.UserPresence{
		UserID:   userID,
		Status:   "offline",
		LastSeen: time.Now(),
	}

	data, err := json.Marshal(presence)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, data, 24*time.Hour).Err()
}

// GetUserPresence gets a user's presence
func (r *RedisClient) GetUserPresence(userID uuid.UUID) (*models.UserPresence, error) {
	key := fmt.Sprintf("presence:user:%s", userID.String())
	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return &models.UserPresence{
			UserID:   userID,
			Status:   "offline",
			LastSeen: time.Now(),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var presence models.UserPresence
	if err := json.Unmarshal([]byte(data), &presence); err != nil {
		return nil, err
	}

	return &presence, nil
}

// Typing Indicators

// SetTyping sets a user as typing in a conversation
func (r *RedisClient) SetTyping(conversationID, userID uuid.UUID) error {
	key := fmt.Sprintf("typing:%s", conversationID.String())
	return r.client.SAdd(r.ctx, key, userID.String()).Err()
}

// RemoveTyping removes a user from typing in a conversation
func (r *RedisClient) RemoveTyping(conversationID, userID uuid.UUID) error {
	key := fmt.Sprintf("typing:%s", conversationID.String())
	return r.client.SRem(r.ctx, key, userID.String()).Err()
}

// GetTypingUsers gets all users typing in a conversation
func (r *RedisClient) GetTypingUsers(conversationID uuid.UUID) ([]uuid.UUID, error) {
	key := fmt.Sprintf("typing:%s", conversationID.String())
	members, err := r.client.SMembers(r.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		userID, err := uuid.Parse(member)
		if err != nil {
			continue
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}

// Pub/Sub

// PublishMessage publishes a message to the messages channel
func (r *RedisClient) PublishMessage(message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return r.client.Publish(r.ctx, "messages", data).Err()
}

// SubscribeToMessages subscribes to the messages channel
func (r *RedisClient) SubscribeToMessages() *redis.PubSub {
	return r.client.Subscribe(r.ctx, "messages")
}

// PublishPresence publishes a presence update
func (r *RedisClient) PublishPresence(presence models.UserPresence) error {
	data, err := json.Marshal(presence)
	if err != nil {
		return err
	}

	return r.client.Publish(r.ctx, "presence", data).Err()
}

// SubscribeToPresence subscribes to presence updates
func (r *RedisClient) SubscribeToPresence() *redis.PubSub {
	return r.client.Subscribe(r.ctx, "presence")
}

// PublishTyping publishes a typing indicator
func (r *RedisClient) PublishTyping(typing models.TypingIndicator) error {
	data, err := json.Marshal(typing)
	if err != nil {
		return err
	}

	return r.client.Publish(r.ctx, "typing", data).Err()
}

// SubscribeToTyping subscribes to typing indicators
func (r *RedisClient) SubscribeToTyping() *redis.PubSub {
	return r.client.Subscribe(r.ctx, "typing")
}

// GetClient returns the underlying Redis client
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// AllowAction implements a Redis-backed token-bucket limiter per key (user+action).
// Returns true if the action is allowed, false if rate-limited.
func (r *RedisClient) AllowAction(userID uuid.UUID, action string, rate int, burst int) (bool, error) {
	key := fmt.Sprintf("rl:%s:%s", action, userID.String())
	// Lua script: manage tokens and last timestamp
	script := `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local vals = redis.call('HMGET', key, 'tokens', 'last')
local tokens = tonumber(vals[1])
local last = tonumber(vals[2])
if tokens == nil then tokens = burst end
if last == nil then last = now end
local delta = math.max(0, now - last)
local new_tokens = math.min(burst, tokens + (delta * rate / 1000))
if new_tokens >= 1 then
	new_tokens = new_tokens - 1
	redis.call('HMSET', key, 'tokens', new_tokens, 'last', now)
	redis.call('PEXPIRE', key, 60000)
	return 1
else
	redis.call('HMSET', key, 'tokens', new_tokens, 'last', now)
	redis.call('PEXPIRE', key, 60000)
	return 0
end
`

	now := time.Now().UnixNano() / int64(time.Millisecond)
	res, err := r.client.Eval(r.ctx, script, []string{key}, rate, burst, now).Result()
	if err != nil {
		return false, err
	}
	// Eval returns int64 (1 or 0)
	switch v := res.(type) {
	case int64:
		return v == 1, nil
	case int:
		return v == 1, nil
	default:
		return false, fmt.Errorf("unexpected result from rate limiter: %T %v", res, res)
	}
}
