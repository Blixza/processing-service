package config

import (
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"
)

type ServerConfig struct {
	Port                 int
	MetricsPort          int
	GrpcPort             int
	ShutdownTimeoutSec   int
	ReadHeaderTimeoutSec int
	ReadTimeoutSec       int
	WriteTimeoutSec      int
	IdleTimeoutSec       int
}

func NewServerConfig(log *zap.Logger) ServerConfig {
	cfg := ServerConfig{
		Port:                 8081,  //nolint:mnd // default value
		MetricsPort:          2112,  //nolint:mnd // default value
		GrpcPort:             50051, //nolint:mnd // default value
		ShutdownTimeoutSec:   5,     //nolint:mnd // default value
		ReadHeaderTimeoutSec: 20,    //nolint:mnd // default value
		ReadTimeoutSec:       5,     //nolint:mnd // default value
		WriteTimeoutSec:      10,    //nolint:mnd // default value
		IdleTimeoutSec:       120,   //nolint:mnd // default value
	}

	if v := parse(log, "SERVER_PORT"); v != 0 {
		cfg.Port = v
	}

	if v := parse(log, "SHUTDOWN_TIMEOUT_SEC"); v != 0 {
		cfg.Port = v
	}

	if v := parse(log, "READ_HEADER_TIMEOUT_SEC"); v != 0 {
		cfg.Port = v
	}

	if v := parse(log, "READ_TIMEOUT_SEC"); v != 0 {
		cfg.Port = v
	}

	if v := parse(log, "WRITE_TIMEOUT_SEC"); v != 0 {
		cfg.Port = v
	}

	if v := parse(log, "IDLE_TIMEOUT_SEC"); v != 0 {
		cfg.Port = v
	}

	return cfg
}

func parse(log *zap.Logger, key string) int {
	value := os.Getenv(key)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Error(fmt.Sprintf("Invalid %s env value", key), zap.Error(err))
	}

	return parsed
}
