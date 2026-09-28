package user_test

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
	"aegis/internal/user"
	"aegis/pkg/database"
	"aegis/pkg/middleware"
	"aegis/pkg/rbac"
	"aegis/pkg/response"

	"github.com/go-chi/chi/v5"
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

func seed(t *testing.T, db *gorm.DB, email, password string, role database.Role) database.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	u := database.User{
		Email: email, PasswordHash: hash, Name: email,
		Role: role, Status: database.UserStatusActive,
	}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func mountUsers(db *gorm.DB, tok auth.Tokens) http.Handler {
	h := &user.Handler{Service: &user.Service{
		Users: &user.Repository{DB: db},
		Audit: &auditlog.Repository{DB: db},
	}}
	r := chi.NewRouter()
	r.Use(middleware.RequireAuth(tok))
	r.With(middleware.RequirePermission(rbac.UsersRead)).Get("/users", h.List)
	r.With(middleware.RequirePermission(rbac.UsersWrite)).Post("/users", h.Create)
	r.With(middleware.RequirePermission(rbac.UsersWrite)).Patch("/users/{id}", h.Patch)
	return r
}

func bearer(t *testing.T, db *gorm.DB, tok auth.Tokens, email, password string) string {
	t.Helper()
	svc := &auth.Service{
		Users:  &auth.Repository{DB: db},
		Store:  auth.NewMemoryRefreshStore(),
		Tokens: tok,
		TTL:    time.Hour,
	}
	out, _, err := svc.Login(context.Background(), email, password)
	require.NoError(t, err)
	return out.AccessToken
}

func TestCreateUserUniqueEmailAndPatchRole(t *testing.T) {
	db := testDB(t)
	seed(t, db, "admin@example.com", "password12", database.RoleAdmin)
	tok := auth.Tokens{Secret: []byte("s"), AccessTTL: time.Hour}
	srv := httptest.NewServer(mountUsers(db, tok))
	t.Cleanup(srv.Close)
	token := bearer(t, db, tok, "admin@example.com", "password12")

	create := func() *http.Response {
		body, _ := json.Marshal(map[string]string{
			"email": "dup@example.com", "password": "password12",
			"name": "Dup", "role": "REPORTER",
		})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/users", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		return res
	}

	res := create()
	require.Equal(t, http.StatusCreated, res.StatusCode)
	var env response.Envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&env))
	res.Body.Close()
	raw, _ := json.Marshal(env.Data)
	var created auth.UserView
	require.NoError(t, json.Unmarshal(raw, &created))

	dup := create()
	require.Equal(t, http.StatusConflict, dup.StatusCode)
	dup.Body.Close()

	patch, _ := json.Marshal(map[string]string{"role": "SUPERVISOR"})
	preq, _ := http.NewRequest(http.MethodPatch, srv.URL+"/users/"+created.ID, bytes.NewReader(patch))
	preq.Header.Set("Authorization", "Bearer "+token)
	preq.Header.Set("Content-Type", "application/json")
	pres, err := http.DefaultClient.Do(preq)
	require.NoError(t, err)
	defer pres.Body.Close()
	require.Equal(t, http.StatusOK, pres.StatusCode)
	var penv response.Envelope
	require.NoError(t, json.NewDecoder(pres.Body).Decode(&penv))
	praw, _ := json.Marshal(penv.Data)
	var updated auth.UserView
	require.NoError(t, json.Unmarshal(praw, &updated))
	require.Equal(t, "SUPERVISOR", updated.Role)

	var createdN int64
	require.NoError(t, db.Model(&database.AuditLog{}).Where("action = ? AND entity_type = ?", database.AuditCreated, "User").Count(&createdN).Error)
	require.Equal(t, int64(1), createdN)
	var updatedN int64
	require.NoError(t, db.Model(&database.AuditLog{}).Where("action = ? AND entity_type = ?", database.AuditUpdated, "User").Count(&updatedN).Error)
	require.Equal(t, int64(1), updatedN)
}

func TestNonAdminCannotCreateUser(t *testing.T) {
	db := testDB(t)
	seed(t, db, "rep@example.com", "password12", database.RoleReporter)
	tok := auth.Tokens{Secret: []byte("s"), AccessTTL: time.Hour}
	srv := httptest.NewServer(mountUsers(db, tok))
	t.Cleanup(srv.Close)
	token := bearer(t, db, tok, "rep@example.com", "password12")

	body, _ := json.Marshal(map[string]string{
		"email": "x@example.com", "password": "password12", "name": "X", "role": "REPORTER",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/users", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusForbidden, res.StatusCode)
}
