package config_test

import (
	"testing"
	"time"

	"aegis/config"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("JWT_ACCESS_SECRET", "")
	t.Setenv("JWT_ACCESS_TTL", "")
	cfg := config.Load()
	require.Equal(t, "8080", cfg.Port)
	require.Equal(t, "development", cfg.AppEnv)
	require.Empty(t, cfg.DatabaseDSN)
	require.Equal(t, "dev-insecure-change-me", cfg.JWTAccessSecret)
	require.Equal(t, 15*time.Minute, cfg.JWTAccessTTL)
}
