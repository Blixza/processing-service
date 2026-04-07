package config

import "os"

type LoggerConfig struct {
	Level             string
	DisableStacktrace string
	FormatTime        string
}

func NewLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:             os.Getenv("LOG_LEVEL"),
		DisableStacktrace: os.Getenv("LOG_DISABLE_STACKTRACE"),
		FormatTime:        os.Getenv("LOG_FORMAT_TIME"),
	}
}
