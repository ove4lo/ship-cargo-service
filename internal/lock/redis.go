package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisLock implements distributed locking using Redis.
type RedisLock struct {
	client *redis.Client
}

// NewRedisLock creates a new instance of RedisLock.
func NewRedisLock(client *redis.Client) *RedisLock {
	return &RedisLock{client: client}
}

// Acquire attempts to acquire the lock. Returns a token for release.
// Returns an error if the lock is held.
func (l *RedisLock) Acquire(ctx context.Context, key string, ttl time.Duration) (string, error) {
	token := uuid.New().String()

	ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return "", fmt.Errorf("redis setnx: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("lock is already held: %s", key)
	}

	return token, nil
}

// Release releases the lock, but only if the token matches.
// This protects against the scenario where the lock has expired due to TTL
// and been acquired by another process, while the first process attempts
// to release a lock that no longer belongs to it.
func (l *RedisLock) Release(ctx context.Context, key string, token string) error {
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		end
		return 0
	`

	_, err := l.client.Eval(ctx, script, []string{key}, token).Result()
	return err
}