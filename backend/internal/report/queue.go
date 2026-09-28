package report

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const reportQueueKey = "report:jobs"

type JobQueue interface {
	Enqueue(ctx context.Context, reportID uuid.UUID) error
	Pop(ctx context.Context, n int) ([]uuid.UUID, error)
}

type MemoryQueue struct {
	mu    sync.Mutex
	items []uuid.UUID
}

func (m *MemoryQueue) Enqueue(_ context.Context, reportID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, reportID)
	return nil
}

func (m *MemoryQueue) Pop(_ context.Context, n int) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n <= 0 || len(m.items) == 0 {
		return nil, nil
	}
	if n > len(m.items) {
		n = len(m.items)
	}
	out := append([]uuid.UUID(nil), m.items[:n]...)
	m.items = m.items[n:]
	return out, nil
}

type RedisQueue struct {
	Client *redis.Client
}

func (r RedisQueue) Enqueue(ctx context.Context, reportID uuid.UUID) error {
	if r.Client == nil {
		return nil
	}
	return r.Client.LPush(ctx, reportQueueKey, reportID.String()).Err()
}

func (r RedisQueue) Pop(ctx context.Context, n int) ([]uuid.UUID, error) {
	if r.Client == nil || n <= 0 {
		return nil, nil
	}
	out := make([]uuid.UUID, 0, n)
	for i := 0; i < n; i++ {
		s, err := r.Client.RPop(ctx, reportQueueKey).Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return out, err
		}
		id, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}
