package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(appEnv string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	if appEnv == "development" {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	return cfg.Build()
}
