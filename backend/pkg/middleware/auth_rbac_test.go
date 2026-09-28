package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aegis/internal/auth"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/middleware"
	"aegis/pkg/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLoginRateLimitSixthAttempt429(t *testing.T) {
	limiter := &middleware.MemoryLimiter{Limit: 5, Window: time.Minute}
	h := middleware.LoginRateLimit(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, "attempt %d", i+1)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Contains(t, rec.Body.String(), "RATE_LIMIT")
}

func TestLoginRateLimitWindowReset(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	limiter := &middleware.MemoryLimiter{
		Limit:  5,
		Window: time.Minute,
		Now:    func() time.Time { return now },
	}
	h := middleware.LoginRateLimit(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		r.RemoteAddr = "10.0.0.2:9"
		return r
	}
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req())
		require.Equal(t, http.StatusOK, rec.Code)
	}
	now = now.Add(61 * time.Second)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req())
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermissionReporterForbidden(t *testing.T) {
	inner := middleware.RequirePermission(rbac.UsersWrite)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := authctx.Principal{ID: uuid.New(), Role: database.RoleReporter, Email: "r@x.com"}
		inner.ServeHTTP(w, r.WithContext(authctx.WithPrincipal(r.Context(), p)))
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/users", nil))
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermissionSuperAdminAllowed(t *testing.T) {
	inner := middleware.RequirePermission(rbac.IncidentsClose)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := authctx.Principal{ID: uuid.New(), Role: database.RoleSuperAdmin, Email: "sa@x.com"}
		inner.ServeHTTP(w, r.WithContext(authctx.WithPrincipal(r.Context(), p)))
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/incidents/x/close", nil))
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRequireAuthRejectsExpired(t *testing.T) {
	now := time.Now()
	tok := auth.Tokens{Secret: []byte("secret"), AccessTTL: time.Minute, Now: func() time.Time { return now }}
	u := database.User{ID: uuid.New(), Email: "a@b.com", Role: database.RoleAdmin}
	raw, err := tok.IssueAccess(u)
	require.NoError(t, err)

	expired := auth.Tokens{Secret: []byte("secret"), AccessTTL: time.Minute, Now: func() time.Time { return now.Add(2 * time.Minute) }}
	h := middleware.RequireAuth(expired)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
