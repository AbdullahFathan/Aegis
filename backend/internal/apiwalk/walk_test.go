package apiwalk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aegis/internal/auditlog"
	"aegis/internal/auth"
	correctiveaction "aegis/internal/corrective_action"
	"aegis/internal/dashboard"
	"aegis/internal/file"
	"aegis/internal/incident"
	"aegis/internal/location"
	"aegis/internal/notification"
	"aegis/internal/rca"
	"aegis/internal/report"
	"aegis/internal/user"
	"aegis/internal/workflow"
	"aegis/pkg/database"
	"aegis/pkg/logger"
	"aegis/pkg/middleware"
	"aegis/pkg/notifier"
	aegispdf "aegis/pkg/pdf"
	"aegis/pkg/rbac"
	"aegis/pkg/storage"

	"github.com/go-chi/chi/v5"
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

func seedUser(t *testing.T, db *gorm.DB, email string, role database.Role) {
	t.Helper()
	hash, err := auth.HashPassword("password12")
	require.NoError(t, err)
	u := database.User{Email: email, PasswordHash: hash, Name: email, Role: role, Status: database.UserStatusActive}
	require.NoError(t, db.Create(&u).Error)
}

type api struct {
	h      http.Handler
	tokens map[string]string
	rep    *report.Service
}

func mount(t *testing.T, db *gorm.DB) *api {
	t.Helper()
	log, err := logger.New("test")
	require.NoError(t, err)
	tok := auth.Tokens{Secret: []byte("walk-secret"), AccessTTL: time.Hour}
	store := auth.NewMemoryRefreshStore()
	authSvc := &auth.Service{Users: &auth.Repository{DB: db}, Store: store, Tokens: tok, TTL: time.Hour}
	authH := &auth.Handler{Service: authSvc, CookieSecure: false, CookieMaxAge: time.Hour}
	audit := &auditlog.Repository{DB: db}
	userH := &user.Handler{Service: &user.Service{Users: &user.Repository{DB: db}, Audit: audit}}
	locH := &location.Handler{Service: &location.Service{Repo: &location.Repository{DB: db}, Audit: audit}}
	incRepo := &incident.Repository{DB: db}
	notifSvc := &notification.Service{DB: db, Mailer: notifier.NoopMailer{}, Log: log, Once: &notification.MemoryOnce{}}
	emerg := notifier.LogEmergency{Log: log}
	incH := &incident.Handler{Service: &incident.Service{Repo: incRepo, Audit: audit, Notify: notifSvc, Emergency: emerg}}
	wfH := &workflow.Handler{Service: &workflow.Service{DB: db, Audit: audit, Notify: notifSvc, Emergency: emerg}}
	rcaH := &rca.Handler{Service: &rca.Service{DB: db, Audit: audit}}
	caH := &correctiveaction.Handler{Service: &correctiveaction.Service{DB: db, Audit: audit, Notify: notifSvc}}
	notifH := &notification.Handler{Service: notifSvc}
	mem := storage.NewMemory()
	fileH := &file.Handler{Service: &file.Service{Repo: incRepo, Store: mem, Audit: audit, Now: func() time.Time {
		return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	}}}
	repSvc := &report.Service{
		DB: db, Queue: &report.MemoryQueue{}, Store: mem, PDF: aegispdf.Fpdf{},
		Notify: notifSvc, Once: &notification.MemoryOnce{}, TZ: time.UTC, CompanyName: "Aegis",
	}
	repH := &report.Handler{Service: repSvc}
	dashH := &dashboard.Handler{Service: &dashboard.Service{DB: db}}
	auditH := &auditlog.Handler{Repo: audit}

	r := chi.NewRouter()
	limiter := &middleware.MemoryLimiter{Limit: 100, Window: time.Minute}
	r.With(middleware.LoginRateLimit(limiter)).Post("/auth/login", authH.Login)
	r.Post("/auth/refresh", authH.Refresh)
	r.Post("/auth/logout", authH.Logout)
	r.Group(func(ar chi.Router) {
		ar.Use(middleware.RequireAuth(tok))
		ar.Get("/auth/me", authH.Me)
		ar.With(middleware.RequirePermission(rbac.UsersRead)).Get("/users", userH.List)
		ar.With(middleware.RequirePermission(rbac.UsersWrite)).Post("/users", userH.Create)
		ar.With(middleware.RequirePermission(rbac.UsersWrite)).Patch("/users/{id}", userH.Patch)
		ar.With(middleware.RequirePermission(rbac.LocationsRead)).Get("/regions", locH.ListRegions)
		ar.With(middleware.RequirePermission(rbac.LocationsWrite)).Post("/regions", locH.CreateRegion)
		ar.With(middleware.RequirePermission(rbac.LocationsRead)).Get("/locations", locH.ListLocations)
		ar.With(middleware.RequirePermission(rbac.LocationsWrite)).Post("/locations", locH.CreateLocation)
		ar.With(middleware.RequirePermission(rbac.LocationsWrite)).Patch("/locations/{id}", locH.PatchLocation)
		ar.With(middleware.RequirePermission(rbac.LocationsWrite)).Delete("/locations/{id}", locH.DeleteLocation)
		ar.With(middleware.RequirePermission(rbac.LocationsRead)).Get("/locations/{id}/areas", locH.ListAreas)
		ar.With(middleware.RequirePermission(rbac.LocationsWrite)).Post("/locations/{id}/areas", locH.CreateArea)
		ar.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents", incH.List)
		ar.With(middleware.RequirePermission(rbac.IncidentsWrite)).Post("/incidents", incH.Create)
		ar.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents/{id}", incH.Get)
		ar.With(middleware.RequirePermission(rbac.IncidentsWrite)).Patch("/incidents/{id}", incH.Patch)
		ar.With(middleware.RequirePermission(rbac.IncidentsWrite)).Post("/incidents/{id}/submit", incH.Submit)
		ar.With(middleware.RequirePermission(rbac.IncidentsVerify)).Post("/incidents/{id}/verify", wfH.Verify)
		ar.With(middleware.RequirePermission(rbac.IncidentsReject)).Post("/incidents/{id}/reject", wfH.Reject)
		ar.With(middleware.RequirePermission(rbac.IncidentsClose)).Post("/incidents/{id}/close", wfH.Close)
		ar.With(middleware.RequirePermission(rbac.IncidentsWrite)).Post("/incidents/{id}/start-corrective-action", wfH.StartCorrectiveAction)
		ar.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents/{id}/timeline", wfH.Timeline)
		ar.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents/{id}/files", fileH.List)
		ar.With(middleware.RequirePermission(rbac.FilesWrite)).Post("/incidents/{id}/files", fileH.Upload)
		ar.With(middleware.RequirePermission(rbac.FilesWrite)).Delete("/incidents/{id}/files/{fileId}", fileH.Delete)
		ar.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents/{id}/rca", rcaH.Get)
		ar.With(middleware.RequirePermission(rbac.RCAWrite)).Put("/incidents/{id}/rca", rcaH.Upsert)
		ar.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/rca-templates/{category}", rcaH.GetTemplate)
		ar.With(middleware.RequirePermission(rbac.RCAWrite)).Put("/rca-templates/{category}", rcaH.PutTemplate)
		ar.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/incidents/{id}/corrective-actions", caH.ListByIncident)
		ar.With(middleware.RequirePermission(rbac.CAWrite)).Post("/incidents/{id}/corrective-actions", caH.Create)
		ar.With(middleware.RequirePermission(rbac.IncidentsRead)).Get("/corrective-actions", caH.Tracker)
		ar.With(middleware.RequirePermission(rbac.IncidentsWrite)).Patch("/corrective-actions/{id}", caH.Patch)
		ar.With(middleware.RequirePermission(rbac.CAVerify)).Post("/corrective-actions/{id}/verify", caH.Verify)
		ar.Get("/notifications", notifH.List)
		ar.Patch("/notifications/{id}/read", notifH.MarkRead)
		ar.Put("/notifications/preferences", notifH.PutPreference)
		ar.With(middleware.RequirePermission(rbac.DashboardRead)).Get("/dashboard/summary", dashH.Summary)
		ar.With(middleware.RequirePermission(rbac.DashboardRead)).Get("/dashboard/trends", dashH.Trends)
		ar.With(middleware.RequirePermission(rbac.DashboardRead)).Get("/dashboard/heatmap", dashH.Heatmap)
		ar.With(middleware.RequirePermission(rbac.ReportsExport)).Get("/reports/monthly", repH.Monthly)
		ar.With(middleware.RequirePermission(rbac.ReportsExport)).Get("/reports/ltifr", repH.LTIFR)
		ar.With(middleware.RequirePermission(rbac.ReportsExport)).Get("/reports/corrective-actions", repH.CorrectiveActions)
		ar.With(middleware.RequirePermission(rbac.ReportsExport)).Get("/reports/investigation/{id}", repH.Investigation)
		ar.With(middleware.RequirePermission(rbac.ReportsExport)).Get("/reports/jobs/{id}", repH.Job)
		ar.With(middleware.RequirePermission(rbac.ReportsExport)).Get("/reports", repH.Archive)
		ar.With(middleware.RequirePermission(rbac.LocationsWrite)).Put("/work-hours", repH.PutWorkHours)
		ar.With(middleware.RequirePermission(rbac.LocationsWrite)).Get("/work-hours", repH.ListWorkHours)
		ar.With(middleware.RequirePermission(rbac.AuditLogsRead)).Get("/audit-logs", auditH.List)
	})
	return &api{h: r, tokens: map[string]string{}, rep: repSvc}
}

func (a *api) login(t *testing.T, email string) string {
	t.Helper()
	status, data, _ := a.call(t, http.MethodPost, "/auth/login", "", map[string]string{"email": email, "password": "password12"}, nil)
	require.Equal(t, http.StatusOK, status)
	token, _ := data["accessToken"].(string)
	require.NotEmpty(t, token)
	a.tokens[email] = token
	return token
}

func (a *api) call(t *testing.T, method, path, email string, jsonBody any, raw *rawBody) (int, map[string]any, []byte) {
	t.Helper()
	var body io.Reader
	ct := "application/json"
	if raw != nil {
		body = bytes.NewReader(raw.b)
		ct = raw.ct
	} else if jsonBody != nil {
		b, err := json.Marshal(jsonBody)
		require.NoError(t, err)
		body = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", ct)
	}
	if email != "" {
		req.Header.Set("Authorization", "Bearer "+a.tokens[email])
	}
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	resp := rec.Body.Bytes()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(resp, &env)
	m := map[string]any{}
	if len(env.Data) > 0 && env.Data[0] == '{' {
		_ = json.Unmarshal(env.Data, &m)
	}
	return rec.Code, m, resp
}

type rawBody struct {
	b  []byte
	ct string
}

func idOf(t *testing.T, data map[string]any) string {
	t.Helper()
	id, _ := data["id"].(string)
	require.NotEmpty(t, id)
	return id
}

func TestAPIWalkCoversHandlers(t *testing.T) {
	db := testDB(t)
	seedUser(t, db, "admin@example.com", database.RoleAdmin)
	seedUser(t, db, "rep@example.com", database.RoleReporter)
	seedUser(t, db, "sup@example.com", database.RoleSupervisor)
	seedUser(t, db, "off@example.com", database.RoleHSEOfficer)
	seedUser(t, db, "mgr@example.com", database.RoleHSEManager)
	a := mount(t, db)

	status, _, body := a.call(t, http.MethodPost, "/auth/login", "", map[string]string{}, nil)
	require.Equal(t, http.StatusUnprocessableEntity, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/auth/me", "", nil, nil)
	require.Equal(t, http.StatusUnauthorized, status)

	admin := a.login(t, "admin@example.com")
	_ = admin
	a.login(t, "rep@example.com")
	a.login(t, "sup@example.com")
	a.login(t, "off@example.com")
	a.login(t, "mgr@example.com")

	status, me, _ := a.call(t, http.MethodGet, "/auth/me", "admin@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "admin@example.com", me["email"])

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	req = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec = httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	status, created, body := a.call(t, http.MethodPost, "/users", "admin@example.com", map[string]string{
		"email": "extra@example.com", "password": "password12", "name": "Extra", "role": "REPORTER", "status": "ACTIVE",
	}, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	extraID := idOf(t, created)
	status, _, body = a.call(t, http.MethodPatch, "/users/"+extraID, "admin@example.com", map[string]string{"name": "Extra Two", "role": "REPORTER"}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, body = a.call(t, http.MethodGet, "/users?page=1&pageSize=10", "admin@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodPost, "/users", "rep@example.com", map[string]string{"email": "x@y.z"}, nil)
	require.Equal(t, http.StatusForbidden, status)
	status, _, _ = a.call(t, http.MethodPost, "/users", "admin@example.com", nil, &rawBody{b: []byte(`{`), ct: "application/json"})
	require.Equal(t, http.StatusUnprocessableEntity, status)

	status, region, body := a.call(t, http.MethodPost, "/regions", "admin@example.com", map[string]string{"name": "Sulawesi", "code": "SUL"}, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	regionID := idOf(t, region)
	status, _, _ = a.call(t, http.MethodGet, "/regions", "admin@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)

	var sup, off database.User
	require.NoError(t, db.Where("email = ?", "sup@example.com").First(&sup).Error)
	require.NoError(t, db.Where("email = ?", "off@example.com").First(&off).Error)
	status, loc, body := a.call(t, http.MethodPost, "/locations", "admin@example.com", map[string]any{
		"name": "Tambang A", "code": "TMB-A", "type": "TAMBANG", "regionId": regionID,
		"supervisorId": sup.ID.String(), "hseOfficerId": off.ID.String(),
	}, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	locID := idOf(t, loc)
	status, _, body = a.call(t, http.MethodPatch, "/locations/"+locID, "admin@example.com", map[string]string{"name": "Tambang A Utara"}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, area, body := a.call(t, http.MethodPost, "/locations/"+locID+"/areas", "admin@example.com", map[string]string{"name": "Pit North", "code": "PIT-N"}, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	areaID := idOf(t, area)
	status, _, _ = a.call(t, http.MethodGet, "/locations/"+locID+"/areas", "admin@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodGet, "/locations", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)

	status, spare, body := a.call(t, http.MethodPost, "/locations", "admin@example.com", map[string]any{
		"name": "Gudang", "code": "GDG-1", "type": "GUDANG",
		"supervisorId": sup.ID.String(), "hseOfficerId": off.ID.String(),
	}, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	status, _, _ = a.call(t, http.MethodDelete, "/locations/"+idOf(t, spare), "admin@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)

	incBody := map[string]any{
		"title": "Slip", "description": longDesc, "category": "NEAR_MISS", "severity": "LOW",
		"incidentDatetime": "2026-09-01T08:00:00Z", "locationId": locID, "areaId": areaID, "hasVictim": false,
	}
	status, inc, body := a.call(t, http.MethodPost, "/incidents", "rep@example.com", incBody, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	incID := idOf(t, inc)
	status, _, body = a.call(t, http.MethodPatch, "/incidents/"+incID, "rep@example.com", map[string]string{"title": "Slip updated"}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/incidents/"+incID, "rep@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodGet, "/incidents?status=DRAFT&category=NEAR_MISS&locationId="+locID, "rep@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodPost, "/incidents", "rep@example.com", map[string]string{"title": "x"}, nil)
	require.Equal(t, http.StatusUnprocessableEntity, status)
	status, _, _ = a.call(t, http.MethodGet, "/incidents/not-a-uuid", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusUnprocessableEntity, status)

	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 13}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("files", "site.png")
	require.NoError(t, err)
	_, err = part.Write(png)
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	status, uploaded, body := a.call(t, http.MethodPost, "/incidents/"+incID+"/files", "rep@example.com", nil, &rawBody{b: buf.Bytes(), ct: mw.FormDataContentType()})
	require.Equal(t, http.StatusCreated, status, string(body))
	var files []map[string]any
	var env struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &env))
	files = env.Data
	require.NotEmpty(t, files)
	fileID, _ := files[0]["id"].(string)
	status, _, body = a.call(t, http.MethodGet, "/incidents/"+incID+"/files", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, body = a.call(t, http.MethodDelete, "/incidents/"+incID+"/files/"+fileID, "rep@example.com", nil, nil)
	require.Equal(t, http.StatusNoContent, status, string(body))
	_ = uploaded

	status, _, body = a.call(t, http.MethodPost, "/incidents/"+incID+"/submit", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodPost, "/incidents/"+incID+"/submit", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusConflict, status)

	status, notes, body := a.call(t, http.MethodGet, "/notifications", "sup@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	var noteEnv struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &noteEnv))
	if len(noteEnv.Data) > 0 {
		nid, _ := noteEnv.Data[0]["id"].(string)
		status, _, _ = a.call(t, http.MethodPatch, "/notifications/"+nid+"/read", "sup@example.com", nil, nil)
		require.Equal(t, http.StatusNoContent, status)
	}
	_ = notes
	status, _, body = a.call(t, http.MethodPut, "/notifications/preferences", "sup@example.com", map[string]any{
		"eventType": "INCIDENT_SUBMITTED", "emailEnabled": true,
	}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodPut, "/notifications/preferences", "sup@example.com", map[string]any{}, nil)
	require.Equal(t, http.StatusUnprocessableEntity, status)

	status, _, body = a.call(t, http.MethodPost, "/incidents/"+incID+"/verify", "sup@example.com", map[string]string{"comment": "checked"}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/incidents/"+incID+"/timeline", "sup@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodPost, "/incidents/"+incID+"/verify", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusForbidden, status)

	status, _, body = a.call(t, http.MethodPut, "/rca-templates/NEAR_MISS", "off@example.com", map[string]any{
		"timeline": "template", "fiveWhys": []map[string]string{{"why": "wet", "answer": "no drain"}},
	}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/rca-templates/NEAR_MISS", "off@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, body = a.call(t, http.MethodPut, "/incidents/"+incID+"/rca", "off@example.com", map[string]any{
		"timeline": "morning slip", "humanFactor": "rush", "environmentFactor": "wet", "equipmentFactor": "none",
		"fiveWhys": []map[string]string{{"why": "1", "answer": "floor"}}, "completed": true,
	}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/incidents/"+incID+"/rca", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodPut, "/incidents/"+incID+"/rca", "sup@example.com", map[string]any{"timeline": "x"}, nil)
	require.Equal(t, http.StatusForbidden, status)

	status, _, body = a.call(t, http.MethodPost, "/incidents/"+incID+"/start-corrective-action", "off@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))

	var rep database.User
	require.NoError(t, db.Where("email = ?", "rep@example.com").First(&rep).Error)
	status, ca, body := a.call(t, http.MethodPost, "/incidents/"+incID+"/corrective-actions", "off@example.com", map[string]string{
		"description": "Install drain", "actionType": "IMMEDIATE", "priority": "HIGH",
		"assigneeId": rep.ID.String(), "dueDate": "2026-10-01",
	}, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	caID := idOf(t, ca)
	status, _, _ = a.call(t, http.MethodGet, "/incidents/"+incID+"/corrective-actions", "off@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodGet, "/corrective-actions?status=OPEN&priority=HIGH&locationId="+locID, "off@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, body = a.call(t, http.MethodPatch, "/corrective-actions/"+caID, "rep@example.com", map[string]string{"status": "IN_PROGRESS"}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, body = a.call(t, http.MethodPatch, "/corrective-actions/"+caID, "rep@example.com", map[string]string{
		"status": "DONE", "completionNotes": "drain installed",
	}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodPost, "/corrective-actions/"+caID+"/verify", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusForbidden, status)
	status, _, body = a.call(t, http.MethodPost, "/corrective-actions/"+caID+"/verify", "off@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, body = a.call(t, http.MethodPost, "/incidents/"+incID+"/close", "mgr@example.com", map[string]string{"comment": "closed"}, nil)
	require.Equal(t, http.StatusOK, status, string(body))

	incBody["title"] = "Second slip"
	status, inc2, body := a.call(t, http.MethodPost, "/incidents", "rep@example.com", incBody, nil)
	require.Equal(t, http.StatusCreated, status, string(body))
	inc2ID := idOf(t, inc2)
	status, _, body = a.call(t, http.MethodPost, "/incidents/"+inc2ID+"/submit", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodPost, "/incidents/"+inc2ID+"/reject", "sup@example.com", nil, nil)
	require.Equal(t, http.StatusUnprocessableEntity, status)
	status, _, body = a.call(t, http.MethodPost, "/incidents/"+inc2ID+"/reject", "sup@example.com", map[string]string{"comment": "need more detail"}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/incidents/"+inc2ID, "off@example.com", nil, nil)
	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, status)

	q := "?from=2026-01-01T00:00:00Z&to=2026-12-31T00:00:00Z&locationId=" + locID + "&category=NEAR_MISS"
	status, _, body = a.call(t, http.MethodGet, "/dashboard/summary"+q, "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/dashboard/trends"+q, "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodGet, "/dashboard/heatmap"+q, "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodGet, "/dashboard/summary?locationId=bad", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusUnprocessableEntity, status)
	status, _, _ = a.call(t, http.MethodGet, "/dashboard/summary", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusForbidden, status)

	status, _, body = a.call(t, http.MethodPut, "/work-hours", "admin@example.com", map[string]any{
		"locationId": locID, "periodStart": "2026-09-01T00:00:00Z", "periodEnd": "2026-09-30T00:00:00Z", "hours": 1000000,
	}, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/work-hours?locationId="+locID, "admin@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodPut, "/work-hours", "admin@example.com", map[string]any{"hours": 1}, nil)
	require.Equal(t, http.StatusUnprocessableEntity, status)

	status, _, body = a.call(t, http.MethodGet, "/reports/monthly?format=csv", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/reports/monthly", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, job, body := a.call(t, http.MethodGet, "/reports/monthly?format=pdf", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusAccepted, status, string(body))
	jobID, _ := job["jobId"].(string)
	require.NotEmpty(t, jobID)
	require.NoError(t, a.rep.ProcessPending(context.Background(), 5))
	status, _, body = a.call(t, http.MethodGet, "/reports/jobs/"+jobID, "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	ltifrQ := "/reports/ltifr?format=csv&locationId=" + locID + "&from=2026-09-01T00:00:00Z&to=2026-09-30T00:00:00Z"
	status, _, body = a.call(t, http.MethodGet, ltifrQ, "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/reports/ltifr?format=pdf&locationId="+locID+"&from=2026-09-01T00:00:00Z&to=2026-09-30T00:00:00Z", "off@example.com", nil, nil)
	require.Equal(t, http.StatusAccepted, status)
	status, _, _ = a.call(t, http.MethodGet, "/reports/corrective-actions?format=csv", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodGet, "/reports/corrective-actions", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, body = a.call(t, http.MethodGet, "/reports/investigation/"+incID+"?format=pdf", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusAccepted, status, string(body))
	status, _, _ = a.call(t, http.MethodGet, "/reports", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status)
	status, _, _ = a.call(t, http.MethodGet, "/reports/monthly?from=nope", "mgr@example.com", nil, nil)
	require.Equal(t, http.StatusUnprocessableEntity, status)

	status, _, body = a.call(t, http.MethodGet, "/audit-logs?action=STATUS_CHANGED&entityType=Incident", "admin@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	status, _, body = a.call(t, http.MethodGet, "/audit-logs?format=csv", "admin@example.com", nil, nil)
	require.Equal(t, http.StatusOK, status, string(body))
	require.Contains(t, string(body), "createdAt")
	status, _, _ = a.call(t, http.MethodGet, "/audit-logs", "rep@example.com", nil, nil)
	require.Equal(t, http.StatusForbidden, status)
}
