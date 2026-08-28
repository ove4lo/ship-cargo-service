package config

import (
	"fmt"
	"os"
	"time"
)

// Config represents a general configuration.
type Config struct {
	App      App
	Postgres Postgres
	JWT      JWT
	Redis    Redis
	RabbitMQ RabbitMQ
}

// App represents application's configuration.
type App struct {
	Port string
	Env  string
}

// Postgres represents postgres's configuration.
type Postgres struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

// DSN returns the data source name string for PostgreSQL connection.
func (p Postgres) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DB,
	)
}

// JWT represents jwt's configuration.
type JWT struct {
	Secret     string
	Expiration time.Duration
}

// Redis represents redis's configuration.
type Redis struct {
	Host string
	Port string
}

// Addr returns the host and port joined by a colon for Redis connection.
func (r Redis) Addr() string {
	return r.Host + ":" + r.Port
}
// RabbitMQ represents rabbitmq's configuration.
type RabbitMQ struct {
	Host     string
	Port     string
	User     string
	Password string
}

// URL returns the RabbitMQ connection URL.
func (r RabbitMQ) URL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/",
		r.User, r.Password, r.Host, r.Port,
	)
}

// Load is responsible for loading the config with variables from env.
func Load() (*Config, error) {
	jwtExp, err := time.ParseDuration(env("JWT_EXPIRATION", "24h"))
	if err != nil {
		return nil, fmt.Errorf("parse JWT_EXPIRATION: %w", err)
	}

	return &Config{
		App: App{
			Port: env("APP_PORT", "8081"),
			Env:  env("APP_ENV", "development"),
		},

		Postgres: Postgres{
			Host:     env("POSTGRES_HOST", "localhost"),
			Port:     env("POSTGRES_PORT", "5432"),
			User:     env("POSTGRES_USER", "cargo"),
			Password: env("POSTGRES_PASSWORD", "cargo_secret"),
			DB:       env("POSTGRES_DB", "ship_cargo"),
		},

		JWT: JWT{
			Secret:     env("JWT_SECRET", "your-secret-key-change-in-production"),
			Expiration: jwtExp,
		},

		Redis: Redis{
			Host: env("REDIS_HOST", "localhost"),
			Port: env("REDIS_PORT", "6379"),
		},

		RabbitMQ: RabbitMQ{
			Host:     env("RABBITMQ_HOST", "localhost"),
			Port:     env("RABBITMQ_PORT", "5672"),
			User:     env("RABBITMQ_USER", "guest"),
			Password: env("RABBITMQ_PASSWORD", "guest"),
		},
	}, nil
}

// env is a helper: if the variable exists in the environment, we use it; otherwise, we use the default value.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}