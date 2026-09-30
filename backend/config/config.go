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
	DBAPITimeout       time.Duration
	DBReportTimeout    time.Duration
	DBUploadTimeout    time.Duration
	DBJobTimeout       time.Duration
	DBPingTimeout      time.Duration
	DBStatementTimeout time.Duration
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration
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
		DBAPITimeout:       envDuration("DB_API_TIMEOUT", 5*time.Second),
		DBReportTimeout:    envDuration("DB_REPORT_TIMEOUT", 30*time.Second),
		DBUploadTimeout:    envDuration("DB_UPLOAD_TIMEOUT", 60*time.Second),
		DBJobTimeout:       envDuration("DB_JOB_TIMEOUT", 50*time.Second),
		DBPingTimeout:      envDuration("DB_PING_TIMEOUT", 2*time.Second),
		DBStatementTimeout: envDuration("DB_STATEMENT_TIMEOUT", 60*time.Second),
		DBMaxOpenConns:     envInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:     envInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime:  envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
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

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
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
