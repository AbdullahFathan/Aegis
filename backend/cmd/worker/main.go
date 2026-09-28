package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"aegis/config"
	"aegis/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	log.Info("worker_started", zap.String("mode", "noop"))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Info("worker_heartbeat")
		case sig := <-stop:
			log.Info("worker_stop", zap.String("signal", sig.String()))
			return
		}
	}
}
