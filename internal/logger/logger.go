package logger

import (
	"log/slog"
	"main/config"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(cfg *config.LoggerConfig) *zap.Logger {
	var config zap.Config

	switch cfg.Level {
	case "prod":
		config = zap.NewProductionConfig()
	case "dev":
		config = zap.NewDevelopmentConfig()
	default:
		config = zap.NewProductionConfig()
	}

	if isStrTrue(cfg.DisableStacktrace) {
		config.DisableStacktrace = true
	}

	if isStrTrue(cfg.FormatTime) {
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	l, err := config.Build()
	if err != nil {
		handler := slog.NewJSONHandler(os.Stderr, nil)
		logger := slog.New(handler)
		logger.Error("Failed to initialize zap logger", "error", err)
	}

	return l
}

func isStrTrue(s string) bool {
	return strings.ToLower(s) == "true"
}
