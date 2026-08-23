package config

import "os"

// Config represents a general configuration
type Config struct {
	App App
}

// App represents application's configuration
type App struct {
	Port string
	Env string
}

// Load is responsible for loading the config with variables from env
func Load() *Config {
	return &Config{
		App: App{
			Port: env("APP_PORT", "8081"),
			Env: env("APP_ENV", "development")
		}
	}
}

// env is a helper: if the variable exists in the environment, we use it; otherwise, we use the default value
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}