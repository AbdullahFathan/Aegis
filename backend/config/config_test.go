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
	t.Setenv("DB_API_TIMEOUT", "")
	t.Setenv("DB_REPORT_TIMEOUT", "")
	t.Setenv("DB_UPLOAD_TIMEOUT", "")
	t.Setenv("DB_JOB_TIMEOUT", "")
	t.Setenv("DB_PING_TIMEOUT", "")
	t.Setenv("DB_STATEMENT_TIMEOUT", "")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME", "")
	cfg := config.Load()
	require.Equal(t, "8080", cfg.Port)
	require.Equal(t, "development", cfg.AppEnv)
	require.Empty(t, cfg.DatabaseDSN)
	require.Equal(t, "dev-insecure-change-me", cfg.JWTAccessSecret)
	require.Equal(t, 15*time.Minute, cfg.JWTAccessTTL)
	require.Equal(t, 5*time.Second, cfg.DBAPITimeout)
	require.Equal(t, 30*time.Second, cfg.DBReportTimeout)
	require.Equal(t, 60*time.Second, cfg.DBUploadTimeout)
	require.Equal(t, 50*time.Second, cfg.DBJobTimeout)
	require.Equal(t, 2*time.Second, cfg.DBPingTimeout)
	require.Equal(t, 60*time.Second, cfg.DBStatementTimeout)
	require.Equal(t, 25, cfg.DBMaxOpenConns)
	require.Equal(t, 5, cfg.DBMaxIdleConns)
	require.Equal(t, 30*time.Minute, cfg.DBConnMaxLifetime)
}
