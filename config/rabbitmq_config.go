package config

import (
	"fmt"
	"os"
)

type RabbitMQConfig struct {
	User     string
	Password string
	Host     string
	Port     string
}

func NewRabbitMQConfig() RabbitMQConfig {
	return RabbitMQConfig{
		User:     os.Getenv("RABBITMQ_USER"),
		Password: os.Getenv("RABBITMQ_PASSWORD"),
		Host:     os.Getenv("RABBITMQ_HOST"),
		Port:     os.Getenv("RABBITMQ_PORT"),
	}
}

func (c *RabbitMQConfig) Dsn() string {
	return fmt.Sprintf( //nolint:nosprintfhostport
		"amqp://%s:%s@%s:%s/",
		c.User, c.Password, c.Host, c.Port,
	)
}
