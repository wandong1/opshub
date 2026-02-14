package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisLock is a simple distributed lock backed by Redis SET NX.
type RedisLock struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewRedisLock creates a new RedisLock.
func NewRedisLock(client *redis.Client, prefix string, ttl time.Duration) *RedisLock {
	return &RedisLock{client: client, prefix: prefix, ttl: ttl}
}

// TryLock attempts to acquire a lock for the given key.
// Returns a release function and true if the lock was acquired.
func (l *RedisLock) TryLock(ctx context.Context, key string) (release func(), ok bool) {
	lockKey := fmt.Sprintf("%s:%s", l.prefix, key)
	value := uuid.New().String()

	acquired, err := l.client.SetNX(ctx, lockKey, value, l.ttl).Result()
	if err != nil || !acquired {
		return nil, false
	}

	return func() {
		// Only delete if we still own the lock (compare value).
		script := `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
		l.client.Eval(context.Background(), script, []string{lockKey}, value)
	}, true
}
