package location_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	"aegis/internal/location"
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
	u := database.User{
		Email: email, PasswordHash: hash, Name: email,
		Role: role, Status: database.UserStatusActive,
	}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func TestLocationDeactivateNotHardDeleteAndUniqueCode(t *testing.T) {
	db := testDB(t)
	sup := seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	off := seedUser(t, db, "hse@example.com", database.RoleHSEOfficer)
	admin := seedUser(t, db, "admin@example.com", database.RoleAdmin)
	svc := &location.Service{Repo: &location.Repository{DB: db}, Audit: &auditlog.Repository{DB: db}}
	actor := authctx.Principal{ID: admin.ID, Role: admin.Role}

	loc, err := svc.CreateLocation(context.Background(), location.LocationInput{
		Name: "Tambang A", Code: "TMB-A", Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID,
	}, actor, "127.0.0.1")
	require.NoError(t, err)

	_, err = svc.CreateLocation(context.Background(), location.LocationInput{
		Name: "Tambang B", Code: "TMB-A", Type: database.LocationTambang,
		SupervisorID: sup.ID, HSEOfficerID: off.ID,
	}, actor, "127.0.0.1")
	require.ErrorIs(t, err, location.ErrConflict)

	_, err = svc.Deactivate(context.Background(), loc.ID, actor, "127.0.0.1")
	require.NoError(t, err)

	var row database.Location
	require.NoError(t, db.First(&row, "id = ?", loc.ID).Error)
	require.False(t, row.IsActive)
	require.Equal(t, loc.ID, row.ID)

	var createdN int64
	require.NoError(t, db.Model(&database.AuditLog{}).Where("action = ? AND entity_type = ?", database.AuditCreated, "Location").Count(&createdN).Error)
	require.Equal(t, int64(1), createdN)
	var updatedN int64
	require.NoError(t, db.Model(&database.AuditLog{}).Where("action = ? AND entity_type = ?", database.AuditUpdated, "Location").Count(&updatedN).Error)
	require.Equal(t, int64(1), updatedN)
}

func TestLocationDTOValidation(t *testing.T) {
	db := testDB(t)
	admin := seedUser(t, db, "admin@example.com", database.RoleAdmin)
	svc := &location.Service{Repo: &location.Repository{DB: db}, Audit: &auditlog.Repository{DB: db}}
	actor := authctx.Principal{ID: admin.ID, Role: admin.Role}

	_, err := svc.CreateLocation(context.Background(), location.LocationInput{
		Name: "", Code: "X", Type: database.LocationTambang,
		SupervisorID: uuid.New(), HSEOfficerID: uuid.New(),
	}, actor, "127.0.0.1")
	require.ErrorIs(t, err, location.ErrValidation)

	_, err = svc.CreateLocation(context.Background(), location.LocationInput{
		Name: "Site", Code: "SITE-1", Type: "INVALID",
		SupervisorID: admin.ID, HSEOfficerID: admin.ID,
	}, actor, "127.0.0.1")
	require.ErrorIs(t, err, location.ErrValidation)
}

func TestReporterCannotWriteLocation(t *testing.T) {
	db := testDB(t)
	rep := seedUser(t, db, "rep@example.com", database.RoleReporter)
	tok := auth.Tokens{Secret: []byte("s"), AccessTTL: time.Hour}
	h := &location.Handler{Service: &location.Service{Repo: &location.Repository{DB: db}, Audit: auditlog.Noop{}}}
	r := chi.NewRouter()
	r.Use(middleware.RequireAuth(tok))
	r.With(middleware.RequirePermission(rbac.LocationsWrite)).Post("/locations", h.CreateLocation)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	authSvc := &auth.Service{
		Users: &auth.Repository{DB: db}, Store: auth.NewMemoryRefreshStore(),
		Tokens: tok, TTL: time.Hour,
	}
	out, _, err := authSvc.Login(context.Background(), rep.Email, "password12")
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"name": "X", "code": "X", "type": "TAMBANG"})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/locations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+out.AccessToken)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusForbidden, res.StatusCode)
}
