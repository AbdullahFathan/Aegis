package notification

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Once interface {
	Claim(ctx context.Context, key string, ttl time.Duration) bool
}

type MemoryOnce struct {
	mu   sync.Mutex
	keys map[string]time.Time
	Now  func() time.Time
}

func (m *MemoryOnce) Claim(_ context.Context, key string, ttl time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keys == nil {
		m.keys = map[string]time.Time{}
	}
	now := time.Now().UTC()
	if m.Now != nil {
		now = m.Now()
	}
	if exp, ok := m.keys[key]; ok && exp.After(now) {
		return false
	}
	m.keys[key] = now.Add(ttl)
	return true
}

type RedisOnce struct {
	Client *redis.Client
}

func (r RedisOnce) Claim(ctx context.Context, key string, ttl time.Duration) bool {
	if r.Client == nil {
		return true
	}
	ok, err := r.Client.SetNX(ctx, key, "1", ttl).Result()
	return err == nil && ok
}
