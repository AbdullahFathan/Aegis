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
	"aegis/pkg/database"
	"aegis/pkg/logger"
	"aegis/pkg/notifier"
	"aegis/pkg/redisx"

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
