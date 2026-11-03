package config

import (
	"fmt"
	"os"
)

// Config хранит настройки, необходимые для запуска сервиса.
type Config struct {
	HTTPAddress string
	DatabaseURL string
}

// Load читает настройки сервиса из переменных окружения.
func Load() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("переменная окружения DATABASE_URL не задана")
	}

	httpAddress := os.Getenv("HTTP_ADDRESS")
	if httpAddress == "" {
		httpAddress = ":8080"
	}

	return Config{
		HTTPAddress: httpAddress,
		DatabaseURL: databaseURL,
	}, nil
}
