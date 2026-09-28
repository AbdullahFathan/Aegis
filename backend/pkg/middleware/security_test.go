package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aegis/internal/auth"
	"aegis/pkg/database"
	"aegis/pkg/middleware"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSecurityLoginRateLimitSixthAttempt(t *testing.T) {
	limiter := &middleware.MemoryLimiter{Limit: 5, Window: time.Minute}
	h := middleware.LoginRateLimit(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "203.0.113.9:9"
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "203.0.113.9:9"
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestSecurityJWTExpiredAndBadSignature(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	issuer := auth.Tokens{Secret: []byte("correct-secret"), AccessTTL: time.Minute, Now: func() time.Time { return now }}
	u := database.User{ID: uuid.New(), Email: "a@b.com", Role: database.RoleHSEOfficer}
	raw, err := issuer.IssueAccess(u)
	require.NoError(t, err)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	expired := auth.Tokens{Secret: []byte("correct-secret"), AccessTTL: time.Minute, Now: func() time.Time { return now.Add(2 * time.Minute) }}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/incidents", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	middleware.RequireAuth(expired)(okHandler).ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	wrong := auth.Tokens{Secret: []byte("other-secret"), AccessTTL: time.Minute, Now: func() time.Time { return now }}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/incidents", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	middleware.RequireAuth(wrong)(okHandler).ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/incidents", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	middleware.RequireAuth(issuer)(okHandler).ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
