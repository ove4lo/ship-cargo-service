package config

import (
	"fmt"
	"os"
)

// Config represents a general configuration
type Config struct {
	App App
	Postgres Postgres
}

// App represents application's configuration
type App struct {
	Port string
	Env string
}

// Postgres represents postgres's configuration
type Postgres struct {
	Host string
	Port string
	User string
	Password string
	DB string
}

// DSN connect to database
func (p Postgres) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DB,
	)
}

// Load is responsible for loading the config with variables from env
func Load() *Config {
	return &Config{
		App: App{
			Port: env("APP_PORT", "8081"),
			Env: env("APP_ENV", "development"),
		},

		Postgres: Postgres{
			Host: env("POSTGRES_HOST", "localhost"),
			Port: env("POSTGRES_PORT", "5432"),
			User: env("POSTGRES_USER", "cargo"),
			Password: env("POSTGRES_PASSWORD", "cargo_secret"),
			DB: env("POSTGRES_DB", "ship_cargo"),
		},
	}
}

// env is a helper: if the variable exists in the environment, we use it; otherwise, we use the default value
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}