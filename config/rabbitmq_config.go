package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type RabbitMQConfig struct {
	User     string `mapstructure:"RABBITMQ_USER"`
	Password string `mapstructure:"RABBITMQ_PASSWORD"`
	Host     string `mapstructure:"RABBITMQ_HOST"`
	Port     string `mapstructure:"RABBITMQ_PORT"`
}

func NewRabbitMQConfig(path string) (RabbitMQConfig, error) {
	var cfg RabbitMQConfig

	viper.SetConfigFile(path)

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return cfg, err
	}

	err = viper.Unmarshal(&cfg)
	return cfg, err
}

func (c *RabbitMQConfig) Dsn() string {
	return fmt.Sprintf( //nolint:nosprintfhostport // not web url
		"amqp://%s:%s@%s:%s/",
		c.User, c.Password, c.Host, c.Port,
	)
}
