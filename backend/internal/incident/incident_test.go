package incident_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const longDesc = "Worker slipped on wet surface near the loading area during morning shift."

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	auth.SetBcryptCost(bcrypt.MinCost)
	db, err := database.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))
	return db
}

func seedUser(t *testing.T, db *gorm.DB, email string, role database.Role) database.User {
	t.Helper()
	hash, err := auth.HashPassword("password12")
	require.NoError(t, err)
	u := database.User{Email: email, PasswordHash: hash, Name: email, Role: role, Status: database.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func seedLoc(t *testing.T, db *gorm.DB, code string, sup, off database.User) database.Location {
	t.Helper()
	loc := database.Location{
		Name: code, Code: code, Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID, IsActive: true,
	}
	require.NoError(t, db.Create(&loc).Error)
	return loc
}

func svc(db *gorm.DB) *incident.Service {
	return &incident.Service{Repo: &incident.Repository{DB: db}, Audit: &auditlog.Repository{DB: db}}
}

func mount(db *gorm.DB, tok auth.Tokens) http.Handler {
	h := &incident.Handler{Service: svc(db)}
	r := chi.NewRouter()
	r.Use(middleware.RequireAuth(tok))
	r.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents", h.List)
	r.With(middleware.RequirePermission(rbac.IncidentsWrite)).Post("/incidents", h.Create)
	r.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents/{id}", h.Get)
	r.With(middleware.RequirePermission(rbac.IncidentsWrite)).Patch("/incidents/{id}", h.Patch)
	r.With(middleware.RequirePermission(rbac.IncidentsWrite)).Post("/incidents/{id}/submit", h.Submit)
	return r
}

func bearer(t *testing.T, db *gorm.DB, tok auth.Tokens, email string) string {
	t.Helper()
	out, _, err := (&auth.Service{
		Users: &auth.Repository{DB: db}, Store: auth.NewMemoryRefreshStore(),
		Tokens: tok, TTL: time.Hour,
	}).Login(context.Background(), email, "password12")
	require.NoError(t, err)
	return out.AccessToken
}

func createDraft(t *testing.T, s *incident.Service, actor authctx.Principal, locID uuid.UUID) database.Incident {
	t.Helper()
	row, err := s.Create(context.Background(), incident.CreateInput{
		Title: "Slip", Description: longDesc, Category: database.CategoryNearMiss,
		Severity: database.SeverityLow, IncidentDatetime: time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC),
		LocationID: locID,
	}, actor, "127.0.0.1")
	require.NoError(t, err)
	return row
}

func TestCreateValidationAndPatchRules(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	other := seedUser(t, db, "o@example.com", database.RoleReporter)
	sup := seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "off@example.com", database.RoleHSEOfficer)
	loc := seedLoc(t, db, "TMB-A", sup, off)
	s := svc(db)
	actor := authctx.Principal{ID: rep.ID, Role: rep.Role}

	_, err := s.Create(context.Background(), incident.CreateInput{
		Title: "x", Description: "too short", Category: database.CategoryNearMiss,
		Severity: database.SeverityLow, IncidentDatetime: time.Now(), LocationID: loc.ID,
	}, actor, "1.1.1.1")
	require.ErrorIs(t, err, incident.ErrValidation)

	draft := createDraft(t, s, actor, loc.ID)
	title := "Updated title"
	_, err = s.Patch(context.Background(), draft.ID, incident.PatchInput{Title: &title}, actor, "1.1.1.1")
	require.NoError(t, err)

	_, err = s.Submit(context.Background(), draft.ID, actor, "1.1.1.1")
	require.NoError(t, err)

	title2 := "after submit"
	_, err = s.Patch(context.Background(), draft.ID, incident.PatchInput{Title: &title2}, actor, "1.1.1.1")
	require.ErrorIs(t, err, incident.ErrForbidden)

	st := "CLOSED"
	_, err = s.Patch(context.Background(), draft.ID, incident.PatchInput{Status: &st}, actor, "1.1.1.1")
	require.ErrorIs(t, err, incident.ErrValidation)

	_, err = s.Get(draft.ID, authctx.Principal{ID: other.ID, Role: other.Role})
	require.ErrorIs(t, err, incident.ErrNotFound)
}

func TestSubmitNumberPendingReviewAndIdempotency(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	other := seedUser(t, db, "o@example.com", database.RoleReporter)
	sup := seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "off@example.com", database.RoleHSEOfficer)
	loc := seedLoc(t, db, "TMB-A", sup, off)
	fixed := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	s := &incident.Service{Repo: &incident.Repository{DB: db}, Audit: &auditlog.Repository{DB: db}, Now: func() time.Time { return fixed }}
	actor := authctx.Principal{ID: rep.ID, Role: rep.Role}

	a := createDraft(t, s, actor, loc.ID)
	b := createDraft(t, s, actor, loc.ID)
	outA, err := s.Submit(context.Background(), a.ID, actor, "1.1.1.1")
	require.NoError(t, err)
	outB, err := s.Submit(context.Background(), b.ID, actor, "1.1.1.1")
	require.NoError(t, err)
	re := regexp.MustCompile(`^INC-\d{4}-\d{2}-\d{4}$`)
	require.True(t, re.MatchString(*outA.IncidentNumber))
	require.True(t, re.MatchString(*outB.IncidentNumber))
	require.Equal(t, "INC-2026-09-0001", *outA.IncidentNumber)
	require.Equal(t, "INC-2026-09-0002", *outB.IncidentNumber)
	require.Equal(t, database.StatusPendingReview, outA.Status)
	require.NotNil(t, outA.PendingReviewAt)
	require.Equal(t, fixed.UTC(), outA.PendingReviewAt.UTC())

	_, err = s.Submit(context.Background(), a.ID, actor, "1.1.1.1")
	require.ErrorIs(t, err, incident.ErrConflict)

	_, err = s.Submit(context.Background(), a.ID, authctx.Principal{ID: other.ID, Role: other.Role}, "1.1.1.1")
	require.ErrorIs(t, err, incident.ErrForbidden)
}

func TestListRBACHidesForeignDrafts(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	other := seedUser(t, db, "o@example.com", database.RoleReporter)
	sup := seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "off@example.com", database.RoleHSEOfficer)
	supB := seedUser(t, db, "supb@example.com", database.RoleSupervisor)
	offB := seedUser(t, db, "offb@example.com", database.RoleHSEOfficer)
	locA := seedLoc(t, db, "TMB-A", sup, off)
	locB := seedLoc(t, db, "TMB-B", supB, offB)
	s := svc(db)

	mine := createDraft(t, s, authctx.Principal{ID: rep.ID, Role: rep.Role}, locA.ID)
	theirs := createDraft(t, s, authctx.Principal{ID: other.ID, Role: other.Role}, locA.ID)
	_, err := s.Submit(context.Background(), theirs.ID, authctx.Principal{ID: other.ID, Role: other.Role}, "ip")
	require.NoError(t, err)
	otherSite := createDraft(t, s, authctx.Principal{ID: other.ID, Role: other.Role}, locB.ID)
	_, err = s.Submit(context.Background(), otherSite.ID, authctx.Principal{ID: other.ID, Role: other.Role}, "ip")
	require.NoError(t, err)

	items, _, err := s.List(authctx.Principal{ID: rep.ID, Role: rep.Role}, incident.ListFilter{Page: 1, PageSize: 50})
	require.NoError(t, err)
	ids := map[uuid.UUID]bool{}
	for _, it := range items {
		ids[it.ID] = true
	}
	require.True(t, ids[mine.ID])
	require.False(t, ids[theirs.ID])

	items, _, err = s.List(authctx.Principal{ID: sup.ID, Role: sup.Role}, incident.ListFilter{Page: 1, PageSize: 50})
	require.NoError(t, err)
	ids = map[uuid.UUID]bool{}
	for _, it := range items {
		ids[it.ID] = true
	}
	require.False(t, ids[mine.ID], "supervisor must not see others' drafts")
	require.True(t, ids[theirs.ID])
	require.False(t, ids[otherSite.ID])
}

func TestHTTPCreateAndSubmit(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	sup := seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "off@example.com", database.RoleHSEOfficer)
	loc := seedLoc(t, db, "TMB-A", sup, off)
	tok := auth.Tokens{Secret: []byte("s"), AccessTTL: time.Hour}
	srv := httptest.NewServer(mount(db, tok))
	t.Cleanup(srv.Close)
	token := bearer(t, db, tok, rep.Email)

	body := fmt.Sprintf(`{"title":"Slip","description":%q,"category":"NEAR_MISS","severity":"LOW","incidentDatetime":"2026-09-01T08:00:00Z","locationId":"%s","hasVictim":false}`, longDesc, loc.ID)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/incidents", bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusCreated, res.StatusCode)
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var env struct {
		Success bool `json:"success"`
		Data    struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &env))
	require.True(t, env.Success)

	req, err = http.NewRequest(http.MethodPost, srv.URL+"/incidents/"+env.Data.ID+"/submit", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
}
