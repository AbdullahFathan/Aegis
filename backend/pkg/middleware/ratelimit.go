package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"aegis/pkg/httputil"
	"aegis/pkg/response"

	"github.com/redis/go-redis/v9"
)

type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type MemoryLimiter struct {
	Limit  int
	Window time.Duration
	Now    func() time.Time
	hits   map[string][]time.Time
}

func (m *MemoryLimiter) Allow(_ context.Context, key string) (bool, error) {
	now := time.Now()
	if m.Now != nil {
		now = m.Now()
	}
	if m.hits == nil {
		m.hits = map[string][]time.Time{}
	}
	cut := now.Add(-m.Window)
	kept := make([]time.Time, 0, len(m.hits[key]))
	for _, ts := range m.hits[key] {
		if ts.After(cut) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= m.Limit {
		m.hits[key] = kept
		return false, nil
	}
	m.hits[key] = append(kept, now)
	return true, nil
}

type RedisLimiter struct {
	Client *redis.Client
	Limit  int
	Window time.Duration
	Prefix string
}

func (l RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := l.Prefix + key
	n, err := l.Client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		if err := l.Client.Expire(ctx, redisKey, l.Window).Err(); err != nil {
			return false, err
		}
	}
	return n <= int64(l.Limit), nil
}

func LoginRateLimit(limiter Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := httputil.ClientIP(r)
			ok, err := limiter.Allow(r.Context(), ip)
			if err != nil {
				_ = response.Error(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
				return
			}
			if !ok {
				w.Header().Set("Retry-After", strconv.Itoa(60))
				_ = response.Error(w, http.StatusTooManyRequests, "RATE_LIMIT", "too many login attempts")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
