package auditlog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"aegis/internal/auditlog"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/middleware"
	"aegis/pkg/rbac"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuditLogsReporterForbidden(t *testing.T) {
	h := &auditlog.Handler{Repo: &auditlog.Repository{}}
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := authctx.WithPrincipal(req.Context(), authctx.Principal{
				ID: uuid.New(), Role: database.RoleReporter, Email: "r@example.com",
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.With(middleware.RequirePermission(rbac.AuditLogsRead)).Get("/audit-logs", h.List)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/audit-logs", nil)
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}
