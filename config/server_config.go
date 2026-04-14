package config

import (
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port                     int `mapstructure:"SERVER_PORT"`
	MetricsPort              int `mapstructure:"SERVER_METRICS_PORT"`
	GrpcPort                 int `mapstructure:"SERVER_GRPC_PORT"`
	ShutdownTimeoutSec       int `mapstructure:"SERVER_SHUTDOWN_SEC"`
	ReadHeaderTimeoutSec     int `mapstructure:"SERVER_READ_HEADER_TIMEOUT_SEC"`
	ReadTimeoutSec           int `mapstructure:"SERVER_READ_TIMEOUT_SEC"`
	WriteTimeoutSec          int `mapstructure:"SERVER_WRITE_TIMEOUT_SEC"`
	IdleTimeoutSec           int `mapstructure:"SERVER_IDLE_TIMEOUT_SEC"`
	WorkerRestartIntervalSec int `mapstructure:"SERVER_WORKER_RESTART_INTERNVAL_SEC"`
}

func NewServerConfig(path string) (ServerConfig, error) {
	cfg := ServerConfig{
		Port:                     8081,  //nolint:mnd // default value
		MetricsPort:              2112,  //nolint:mnd // default value
		GrpcPort:                 50051, //nolint:mnd // default value
		ShutdownTimeoutSec:       5,     //nolint:mnd // default value
		ReadHeaderTimeoutSec:     20,    //nolint:mnd // default value
		ReadTimeoutSec:           5,     //nolint:mnd // default value
		WriteTimeoutSec:          10,    //nolint:mnd // default value
		IdleTimeoutSec:           120,   //nolint:mnd // default value
		WorkerRestartIntervalSec: 5,     //nolint:mnd // default value
	}

	viper.SetConfigFile(path)

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return cfg, err
	}

	err = viper.Unmarshal(&cfg)
	return cfg, err
}
