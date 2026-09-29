package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	Port               string
	DatabaseDSN        string
	RedisAddr          string
	RustFSEndpoint     string
	RustFSAccessKey    string
	RustFSSecretKey    string
	RustFSBucket       string
	JWTAccessSecret    string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	CookieSecure       bool
	ReportTZ           string
	CompanyName        string
	SuperadminUsername string
	SuperadminPassword string
}

func Load() Config {
	return Config{
		AppEnv:             env("APP_ENV", "development"),
		Port:               env("PORT", "8080"),
		DatabaseDSN:        os.Getenv("DATABASE_DSN"),
		RedisAddr:          env("REDIS_ADDR", "localhost:6379"),
		RustFSEndpoint:     env("RUSTFS_ENDPOINT", "http://localhost:9000"),
		RustFSAccessKey:    os.Getenv("RUSTFS_ACCESS_KEY"),
		RustFSSecretKey:    os.Getenv("RUSTFS_SECRET_KEY"),
		RustFSBucket:       env("RUSTFS_BUCKET", "aegis"),
		JWTAccessSecret:    env("JWT_ACCESS_SECRET", "dev-insecure-change-me"),
		JWTAccessTTL:       envDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:      envDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		CookieSecure:       envBool("COOKIE_SECURE", false),
		ReportTZ:           env("REPORT_TZ", "Asia/Jakarta"),
		CompanyName:        env("COMPANY_NAME", "Aegis"),
		SuperadminUsername: os.Getenv("SUPERADMIN_USERNAME"),
		SuperadminPassword: os.Getenv("SUPERADMIN_PASSWORD"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
