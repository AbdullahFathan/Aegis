package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"aegis/internal/auth"
	"aegis/pkg/database"
	"aegis/pkg/middleware"
	"aegis/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	auth.SetBcryptCost(bcrypt.MinCost)
	os.Exit(m.Run())
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.OpenSQLite("file:" + t.Name() + "?mode=memory&cache=shared")
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(db))
	return db
}

func seedUser(t *testing.T, db *gorm.DB, email, password string, role database.Role, status database.UserStatus) database.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	u := database.User{
		Email:        email,
		PasswordHash: hash,
		Name:         "User",
		Role:         role,
		Status:       status,
	}
	require.NoError(t, db.Create(&u).Error)
	return u
}

func TestPasswordHashVerify(t *testing.T) {
	hash, err := auth.HashPassword("secret-pass")
	require.NoError(t, err)
	require.True(t, auth.CheckPassword(hash, "secret-pass"))
	require.False(t, auth.CheckPassword(hash, "wrong"))
}

func TestJWTIssueValidateExpire(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tok := auth.Tokens{Secret: []byte("k"), AccessTTL: time.Minute, Now: func() time.Time { return now }}
	u := database.User{Email: "a@b.com", Role: database.RoleAdmin}
	raw, err := tok.IssueAccess(u)
	require.NoError(t, err)
	p, err := tok.ParseAccess(raw)
	require.NoError(t, err)
	require.Equal(t, database.RoleAdmin, p.Role)

	tok.Now = func() time.Time { return now.Add(2 * time.Minute) }
	_, err = tok.ParseAccess(raw)
	require.Error(t, err)
}

func newAuthHandler(t *testing.T, db *gorm.DB) (*auth.Handler, auth.Tokens) {
	t.Helper()
	tok := auth.Tokens{Secret: []byte("test-secret"), AccessTTL: time.Hour}
	svc := &auth.Service{
		Users:  &auth.Repository{DB: db},
		Store:  auth.NewMemoryRefreshStore(),
		Tokens: tok,
		TTL:    24 * time.Hour,
	}
	return &auth.Handler{Service: svc, CookieMaxAge: 24 * time.Hour}, tok
}

func mountAuth(h *auth.Handler, tok auth.Tokens) http.Handler {
	r := chi.NewRouter()
	r.Post("/auth/login", h.Login)
	r.Post("/auth/refresh", h.Refresh)
	r.Post("/auth/logout", h.Logout)
	r.With(middleware.RequireAuth(tok)).Get("/auth/me", h.Me)
	return r
}

func TestLoginSuccessAndMe(t *testing.T) {
	db := testDB(t)
	seedUser(t, db, "admin@example.com", "password12", database.RoleAdmin, database.UserStatusActive)
	h, tok := newAuthHandler(t, db)
	srv := httptest.NewServer(mountAuth(h, tok))
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]string{"email": "admin@example.com", "password": "password12"})
	res, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	var env response.Envelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&env))
	require.True(t, env.Success)

	data, err := json.Marshal(env.Data)
	require.NoError(t, err)
	var payload auth.TokenResponse
	require.NoError(t, json.Unmarshal(data, &payload))
	require.NotEmpty(t, payload.AccessToken)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	meRes, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer meRes.Body.Close()
	require.Equal(t, http.StatusOK, meRes.StatusCode)
}

func TestLoginWrongPasswordDoesNotLeakEmail(t *testing.T) {
	db := testDB(t)
	seedUser(t, db, "admin@example.com", "password12", database.RoleAdmin, database.UserStatusActive)
	h, tok := newAuthHandler(t, db)
	srv := httptest.NewServer(mountAuth(h, tok))
	t.Cleanup(srv.Close)

	login := func(email, pass string) string {
		body, _ := json.Marshal(map[string]string{"email": email, "password": pass})
		res, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
		var b bytes.Buffer
		_, _ = b.ReadFrom(res.Body)
		return b.String()
	}

	known := login("admin@example.com", "nope")
	unknown := login("missing@example.com", "nope")
	require.Contains(t, known, "invalid email or password")
	require.Equal(t, known, unknown)
	require.NotContains(t, strings.ToLower(known), "not found")
}

func TestRefreshRotateAndInvalidateAfterLogout(t *testing.T) {
	db := testDB(t)
	seedUser(t, db, "admin@example.com", "password12", database.RoleAdmin, database.UserStatusActive)
	h, tok := newAuthHandler(t, db)
	srv := httptest.NewServer(mountAuth(h, tok))
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]string{"email": "admin@example.com", "password": "password12"})
	res, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "refresh_token" {
			cookie = c
		}
	}
	require.NotNil(t, cookie)

	client := &http.Client{}
	refreshReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/auth/refresh", nil)
	refreshReq.AddCookie(cookie)
	refRes, err := client.Do(refreshReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, refRes.StatusCode)
	var rotated *http.Cookie
	for _, c := range refRes.Cookies() {
		if c.Name == "refresh_token" {
			rotated = c
		}
	}
	refRes.Body.Close()
	require.NotNil(t, rotated)
	require.NotEqual(t, cookie.Value, rotated.Value)

	oldReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/auth/refresh", nil)
	oldReq.AddCookie(cookie)
	oldRes, err := client.Do(oldReq)
	require.NoError(t, err)
	oldRes.Body.Close()
	require.Equal(t, http.StatusUnauthorized, oldRes.StatusCode)

	logoutReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/auth/logout", nil)
	logoutReq.AddCookie(rotated)
	logoutRes, err := client.Do(logoutReq)
	require.NoError(t, err)
	logoutRes.Body.Close()
	require.Equal(t, http.StatusOK, logoutRes.StatusCode)

	after, _ := http.NewRequest(http.MethodPost, srv.URL+"/auth/refresh", nil)
	after.AddCookie(rotated)
	afterRes, err := client.Do(after)
	require.NoError(t, err)
	afterRes.Body.Close()
	require.Equal(t, http.StatusUnauthorized, afterRes.StatusCode)
}
