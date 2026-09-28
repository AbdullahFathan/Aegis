package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrRefreshNotFound = errors.New("refresh token not found")

const refreshKeyPrefix = "refresh:"

type RefreshStore interface {
	Save(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error
	Peek(ctx context.Context, token string) (uuid.UUID, error)
	Delete(ctx context.Context, token string) error
}

func HashRefresh(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type RedisRefreshStore struct {
	Client *redis.Client
}

func (s RedisRefreshStore) Save(ctx context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	return s.Client.Set(ctx, refreshKeyPrefix+HashRefresh(token), userID.String(), ttl).Err()
}

func (s RedisRefreshStore) Peek(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := s.Client.Get(ctx, refreshKeyPrefix+HashRefresh(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, ErrRefreshNotFound
		}
		return uuid.Nil, err
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid refresh payload")
	}
	return id, nil
}

func (s RedisRefreshStore) Delete(ctx context.Context, token string) error {
	return s.Client.Del(ctx, refreshKeyPrefix+HashRefresh(token)).Err()
}

type MemoryRefreshStore struct {
	items map[string]memoryRefresh
}

type memoryRefresh struct {
	userID uuid.UUID
	exp    time.Time
}

func NewMemoryRefreshStore() *MemoryRefreshStore {
	return &MemoryRefreshStore{items: map[string]memoryRefresh{}}
}

func (s *MemoryRefreshStore) Save(_ context.Context, token string, userID uuid.UUID, ttl time.Duration) error {
	s.items[HashRefresh(token)] = memoryRefresh{userID: userID, exp: time.Now().Add(ttl)}
	return nil
}

func (s *MemoryRefreshStore) Peek(_ context.Context, token string) (uuid.UUID, error) {
	item, ok := s.items[HashRefresh(token)]
	if !ok || time.Now().After(item.exp) {
		delete(s.items, HashRefresh(token))
		return uuid.Nil, ErrRefreshNotFound
	}
	return item.userID, nil
}

func (s *MemoryRefreshStore) Delete(_ context.Context, token string) error {
	delete(s.items, HashRefresh(token))
	return nil
}
