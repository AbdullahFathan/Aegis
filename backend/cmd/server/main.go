// Package main is the Aegis HTTP API server.
//
//	@title			Aegis API
//	@version		1.0
//	@description	K3 incident reporting backend (Phase 1 skeleton).
//	@BasePath		/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
package main

import (
	"net/http"
	"os"
	"time"

	"aegis/config"
	_ "aegis/docs"
	"aegis/internal/auditlog"
	"aegis/internal/auth"
	"aegis/internal/location"
	"aegis/internal/user"
	"aegis/pkg/database"
	"aegis/pkg/logger"
	"aegis/pkg/middleware"
	"aegis/pkg/rbac"
	"aegis/pkg/redisx"
	"aegis/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
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
	})

	return r
}
