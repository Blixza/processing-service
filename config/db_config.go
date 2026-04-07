package config

import (
	"fmt"
	"os"
)

type DBConfig struct {
	Name     string
	User     string
	Password string
	Host     string
	Port     string
	SSLMode  string
}

func NewDBConfig() DBConfig {
	return DBConfig{
		Name:     os.Getenv("DB_NAME"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}
}

func (c *DBConfig) Dsn() string {
	return fmt.Sprintf( //nolint:nosprintfhostport // not web url
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}
