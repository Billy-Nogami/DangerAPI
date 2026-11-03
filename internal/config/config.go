package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config хранит настройки, необходимые для запуска сервиса.
type Config struct {
	HTTPAddress       string
	DatabaseURL       string
	RateLimitRequests int
	RateLimitWindow   time.Duration
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
	rateLimitRequests, err := readPositiveInt("RATE_LIMIT_REQUESTS", 60)
	if err != nil {
		return Config{}, err
	}
	rateLimitWindow, err := readDuration("RATE_LIMIT_WINDOW", time.Minute)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddress:       httpAddress,
		DatabaseURL:       databaseURL,
		RateLimitRequests: rateLimitRequests,
		RateLimitWindow:   rateLimitWindow,
	}, nil
}

func readPositiveInt(name string, defaultValue int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("переменная окружения %s должна быть положительным целым числом", name)
	}

	return parsed, nil
}

func readDuration(name string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("переменная окружения %s должна быть положительной длительностью", name)
	}

	return parsed, nil
}
