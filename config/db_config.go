package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type DBConfig struct {
	Name     string `mapstructure:"DB_NAME"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	SSLMode  string `mapstructure:"DB_SSLMODE"`
}

func NewDBConfig(path string) (DBConfig, error) {
	var cfg DBConfig

	viper.SetConfigFile(path)

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return cfg, err
	}

	err = viper.Unmarshal(&cfg)
	return cfg, err
}

func (c *DBConfig) Dsn() string {
	return fmt.Sprintf( //nolint:nosprintfhostport // not web url
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}
