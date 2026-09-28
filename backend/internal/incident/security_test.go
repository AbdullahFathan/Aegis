package incident_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	"aegis/internal/incident"
	"aegis/pkg/authctx"
	"aegis/pkg/database"
	"aegis/pkg/middleware"
	"aegis/pkg/rbac"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestSecurityOfficerCannotReadOtherSiteIncident(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	supA := seedUser(t, db, "supa@example.com", database.RoleSupervisor)
	offA := seedUser(t, db, "offa@example.com", database.RoleHSEOfficer)
	supB := seedUser(t, db, "supb@example.com", database.RoleSupervisor)
	offB := seedUser(t, db, "offb@example.com", database.RoleHSEOfficer)
	locB := seedLoc(t, db, "TMB-B", supB, offB)
	_ = seedLoc(t, db, "TMB-A", supA, offA)
	s := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: auditlog.Noop{}}
	row := createDraft(t, s, authctx.Principal{ID: rep.ID, Role: rep.Role}, locB.ID)
	_, err := s.Submit(t.Context(), row.ID, authctx.Principal{ID: rep.ID, Role: rep.Role}, "ip")
	require.NoError(t, err)

	tok := auth.Tokens{Secret: []byte("idor"), AccessTTL: time.Hour}
	h := &incident.Handler{Service: s}
	r := chi.NewRouter()
	r.Use(middleware.RequireAuth(tok))
	r.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents/{id}", h.Get)

	raw, err := tok.IssueAccess(offA)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/incidents/"+row.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.NotContains(t, rec.Body.String(), "Slip")
	require.NotContains(t, rec.Body.String(), row.ID.String())
}
