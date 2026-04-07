package logger

import (
	"log"
	"main/config"
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
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}

	return l
}

func isStrTrue(s string) bool {
	return strings.ToLower(s) == "true"
}
