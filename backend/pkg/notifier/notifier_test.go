package notifier_test

import (
	"context"
	"testing"

	"aegis/pkg/notifier"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNoopDoesNotPanic(t *testing.T) {
	require.NoError(t, notifier.NoopEmergency{}.NotifyFatality(context.Background(), uuid.New(), "x"))
	require.NoError(t, notifier.NoopMailer{}.Send(context.Background(), "a@b.com", "s", "b"))
	require.NoError(t, notifier.LogEmergency{}.NotifyFatality(context.Background(), uuid.New(), "x"))
}
