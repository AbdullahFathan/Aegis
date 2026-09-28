package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	MaxBytes           int64 = 10 << 20
	MaxFilesPerRequest       = 5
	SignedURLTTL             = time.Hour
)

type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type Memory struct {
	mu      sync.Mutex
	Objects map[string][]byte
	Now     func() time.Time
}

func NewMemory() *Memory {
	return &Memory{Objects: make(map[string][]byte), Now: time.Now}
}

func (m *Memory) Put(_ context.Context, key string, body io.Reader, _ int64, _ string) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Objects == nil {
		m.Objects = make(map[string][]byte)
	}
	m.Objects[key] = b
	return nil
}

func (m *Memory) PresignGet(_ context.Context, key string, expiry time.Duration) (string, error) {
	now := time.Now()
	if m.Now != nil {
		now = m.Now()
	}
	exp := now.Add(expiry).UTC().Unix()
	return fmt.Sprintf("memory://%s?exp=%d", key, exp), nil
}

func (m *Memory) Get(key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.Objects[key]
	return b, ok
}

func Reader(b []byte) io.Reader {
	return bytes.NewReader(b)
}
