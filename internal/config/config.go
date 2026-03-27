// internal/config/config.go
package config

import "os"

type Config struct {
	Port string
	Env  string
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
