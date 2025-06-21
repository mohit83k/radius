package redisclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mohit83k/radius/internal/model"
	"github.com/redis/go-redis/v9"
)

const defaultTTL = 24 * time.Hour

// Store defines the interface for saving accounting data to Redis.
type Store interface {
	Save(ctx context.Context, record model.AccountingRecord) error
}

// RedisStore implements the Store interface using go-redis.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore returns a new RedisStore with auto-reconnect and retry.
func NewRedisStore(addr, password string, db int) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		MaxRetries:      5,
		MinRetryBackoff: 100 * time.Millisecond,
		MaxRetryBackoff: 1 * time.Second,
	})
	return &RedisStore{client: client}
}

// Save stores the accounting record in Redis with a TTL.
func (r *RedisStore) Save(ctx context.Context, record model.AccountingRecord) error {
	key := fmt.Sprintf("radius:acct:%s:%s:%s", record.Username, record.AcctSessionID, record.Timestamp.Format("20060102T150405"))

	value, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	err = r.client.Set(ctx, key, string(value), defaultTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to save record in redis: %w", err)
	}

	return nil
}
