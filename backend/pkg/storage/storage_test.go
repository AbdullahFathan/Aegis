package storage_test

import (
	"context"
	"fmt"
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
