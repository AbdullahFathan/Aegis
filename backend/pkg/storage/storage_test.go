package storage_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"aegis/pkg/storage"

	"github.com/stretchr/testify/require"
)

func TestMemoryPresignUsesExpiryAndClock(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	mem := &storage.Memory{Now: func() time.Time { return now }}
	url, err := mem.PresignGet(context.Background(), "abc", time.Hour)
	require.NoError(t, err)
	require.Contains(t, url, "memory://abc")
	require.Contains(t, url, fmt.Sprintf("exp=%d", now.Add(time.Hour).Unix()))
}

func TestMemoryPutGet(t *testing.T) {
	mem := storage.NewMemory()
	require.NoError(t, mem.Put(context.Background(), "k", strings.NewReader("hi"), 2, "text/plain"))
	b, ok := mem.Get("k")
	require.True(t, ok)
	require.Equal(t, []byte("hi"), b)
}

func TestOpenFallsBackToMemoryWithoutKeys(t *testing.T) {
	store, err := storage.Open(context.Background(), "http://localhost:9000", "", "", "aegis")
	require.NoError(t, err)
	_, ok := store.(*storage.Memory)
	require.True(t, ok)
}

func TestEnsureBucketIdempotent(t *testing.T) {
	endpoint := os.Getenv("RUSTFS_ENDPOINT")
	ak := os.Getenv("RUSTFS_ACCESS_KEY")
	sk := os.Getenv("RUSTFS_SECRET_KEY")
	if endpoint == "" || ak == "" || sk == "" {
		t.Skip("RUSTFS_* not set")
	}
	bucket := os.Getenv("RUSTFS_BUCKET")
	if bucket == "" {
		bucket = "aegis"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s := storage.NewS3(endpoint, ak, sk, bucket)
	require.NoError(t, s.EnsureBucket(ctx))
	require.NoError(t, s.EnsureBucket(ctx))
	require.NoError(t, s.Put(ctx, "ensure-test/ok.txt", strings.NewReader("ok"), 2, "text/plain"))
}
