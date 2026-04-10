package config

import "github.com/spf13/viper"

type LoggerConfig struct {
	Level             string `mapstructure:"LOG_LEVEL"`
	DisableStacktrace bool   `mapstructure:"LOG_DISABLE_STACKTRACE"`
	FormatTime        bool   `mapstructure:"LOG_FORMAT_TIME"`
}

func NewLoggerConfig(path string) (LoggerConfig, error) {
	var cfg LoggerConfig

	viper.SetConfigFile(path)

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return cfg, err
	}

	err = viper.Unmarshal(&cfg)
	return cfg, err
}
