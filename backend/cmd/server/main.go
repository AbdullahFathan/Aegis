// Package main is the Aegis HTTP API server.
//
//	@title			Aegis API
//	@version		1.0
//	@description	K3 incident reporting and compliance API (MVP).
//	@BasePath		/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"aegis/config"
	_ "aegis/docs"
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
	"aegis/pkg/redisx"
	"aegis/pkg/response"
	"aegis/pkg/storage"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	var db *gorm.DB
	if cfg.DatabaseDSN != "" {
		db, err = database.OpenPostgres(cfg.DatabaseDSN)
		if err != nil {
			log.Fatal("database_open_failed", zap.Error(err))
		}
		if err := database.AutoMigrate(db); err != nil {
			log.Fatal("database_migrate_failed", zap.Error(err))
		}
		created, err := user.SeedSuperAdmin(&user.Repository{DB: db}, cfg.SuperadminUsername, cfg.SuperadminPassword)
		if err != nil {
			log.Fatal("superadmin_seed_failed", zap.Error(err))
		}
		if created {
			log.Info("superadmin_seeded")
		}
		log.Info("database_ready")
	} else {
		log.Warn("database_dsn_empty")
	}

	rdb, err := redisx.Open(cfg.RedisAddr)
	if err != nil {
		log.Fatal("redis_open_failed", zap.Error(err))
	}
	defer rdb.Close()

	r := newRouter(cfg, log, db, rdb)
	addr := ":" + cfg.Port
	log.Info("server_listen", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Error("server_exit", zap.Error(err))
		os.Exit(1)
	}
}

func newRouter(cfg config.Config, log *zap.Logger, db *gorm.DB, rdb *redis.Client) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer(log))
	r.Use(middleware.RequestLog(log))

	r.Get("/swagger/*", httpSwagger.WrapHandler)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]string{"status": "ok", "db": "skipped"}
		if db != nil {
			if err := database.Ping(db); err != nil {
				payload["status"] = "degraded"
				payload["db"] = "down"
				_ = response.Success(w, http.StatusServiceUnavailable, payload)
				return
			}
			payload["db"] = "up"
		}
		_ = response.Success(w, http.StatusOK, payload)
	})

	if db == nil {
		return r
	}

	tokens := auth.Tokens{Secret: []byte(cfg.JWTAccessSecret), AccessTTL: cfg.JWTAccessTTL}
	authSvc := &auth.Service{
		Users:  &auth.Repository{DB: db},
		Store:  auth.RedisRefreshStore{Client: rdb},
		Tokens: tokens,
		TTL:    cfg.JWTRefreshTTL,
	}
	authH := &auth.Handler{Service: authSvc, CookieSecure: cfg.CookieSecure, CookieMaxAge: cfg.JWTRefreshTTL}

	limiter := middleware.RedisLimiter{
		Client: rdb,
		Limit:  5,
		Window: time.Minute,
		Prefix: "rl:login:",
	}

	audit := &auditlog.Repository{DB: db}
	userH := &user.Handler{Service: &user.Service{Users: &user.Repository{DB: db}, Audit: audit}}
	locH := &location.Handler{Service: &location.Service{Repo: &location.Repository{DB: db}, Audit: audit}}

	incRepo := &incident.Repository{DB: db}
	notifSvc := &notification.Service{
		DB: db, Mailer: notifier.NoopMailer{}, Log: log, Once: notification.RedisOnce{Client: rdb},
	}
	emerg := notifier.LogEmergency{Log: log}
	incH := &incident.Handler{Service: &incident.Service{Repo: incRepo, Audit: audit, Notify: notifSvc, Emergency: emerg}}
	wfH := &workflow.Handler{Service: &workflow.Service{DB: db, Audit: audit, Notify: notifSvc, Emergency: emerg}}
	rcaH := &rca.Handler{Service: &rca.Service{DB: db, Audit: audit}}
	caH := &correctiveaction.Handler{Service: &correctiveaction.Service{DB: db, Audit: audit, Notify: notifSvc}}
	notifH := &notification.Handler{Service: notifSvc}
	storeCtx, storeCancel := context.WithTimeout(context.Background(), 15*time.Second)
	store, err := storage.Open(storeCtx, cfg.RustFSEndpoint, cfg.RustFSAccessKey, cfg.RustFSSecretKey, cfg.RustFSBucket)
	storeCancel()
	if err != nil {
		log.Fatal("object_store_failed", zap.Error(err))
	}
	fileH := &file.Handler{Service: &file.Service{Repo: incRepo, Store: store, Audit: audit}}
	tz, err := time.LoadLocation(cfg.ReportTZ)
	if err != nil {
		tz = time.UTC
	}
	dashH := &dashboard.Handler{Service: &dashboard.Service{DB: db}}
	repSvc := &report.Service{
		DB: db, Queue: report.RedisQueue{Client: rdb}, Store: store, PDF: aegispdf.Fpdf{},
		Notify: notifSvc, Once: notification.RedisOnce{Client: rdb}, TZ: tz, CompanyName: cfg.CompanyName, Log: log,
	}
	repH := &report.Handler{Service: repSvc}
	auditH := &auditlog.Handler{Repo: audit}

	r.With(middleware.LoginRateLimit(limiter)).Post("/auth/login", authH.Login)
	r.Post("/auth/refresh", authH.Refresh)
	r.Post("/auth/logout", authH.Logout)

	r.Group(func(ar chi.Router) {
		ar.Use(middleware.RequireAuth(tokens))
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

	return r
}
