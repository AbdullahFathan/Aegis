package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aegis/config"
	correctiveaction "aegis/internal/corrective_action"
	"aegis/internal/notification"
	"aegis/internal/report"
	"aegis/pkg/database"
	"aegis/pkg/logger"
	"aegis/pkg/notifier"
	aegispdf "aegis/pkg/pdf"
	"aegis/pkg/redisx"
	"aegis/pkg/storage"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	if cfg.DatabaseDSN == "" {
		log.Fatal("database_dsn_empty")
	}
	db, err := database.OpenPostgres(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal("database_open_failed", zap.Error(err))
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal("database_migrate_failed", zap.Error(err))
	}

	rdb, err := redisx.Open(cfg.RedisAddr)
	if err != nil {
		log.Fatal("redis_open_failed", zap.Error(err))
	}
	defer rdb.Close()

	once := notification.RedisOnce{Client: rdb}
	notif := &notification.Service{
		DB: db, Mailer: notifier.NoopMailer{}, Log: log, Once: once,
	}
	caJobs := &correctiveaction.Service{DB: db, Jobs: notif, Once: once}

	var store storage.ObjectStore = storage.NewMemory()
	if cfg.RustFSAccessKey != "" && cfg.RustFSSecretKey != "" {
		store = storage.NewS3(cfg.RustFSEndpoint, cfg.RustFSAccessKey, cfg.RustFSSecretKey, cfg.RustFSBucket)
	}
	tz, err := time.LoadLocation(cfg.ReportTZ)
	if err != nil {
		tz = time.UTC
	}
	reports := &report.Service{
		DB: db, Queue: report.RedisQueue{Client: rdb}, Store: store, PDF: aegispdf.Fpdf{},
		Notify: notif, Once: once, TZ: tz, CompanyName: cfg.CompanyName,
	}

	log.Info("worker_started", zap.String("mode", "jobs"))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	run := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
		defer cancel()
		ok, err := rdb.SetNX(ctx, "worker:tick", "1", 50*time.Second).Result()
		if err != nil || !ok {
			return
		}
		now := time.Now().UTC()
		if err := caJobs.RunOverdueJob(ctx, now); err != nil {
			log.Error("worker_overdue_failed", zap.Error(err))
		}
		if err := notif.RunSLAReminders(ctx, now); err != nil {
			log.Error("worker_sla_failed", zap.Error(err))
		}
		if err := reports.ProcessPending(ctx, 5); err != nil {
			log.Error("worker_reports_failed", zap.Error(err))
		}
		if err := reports.RunMonthlySchedule(ctx, now); err != nil {
			log.Error("worker_monthly_failed", zap.Error(err))
		}
	}
	run()

	for {
		select {
		case <-ticker.C:
			run()
		case sig := <-stop:
			log.Info("worker_stop", zap.String("signal", sig.String()))
			return
		}
	}
}
