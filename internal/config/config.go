package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config хранит настройки, необходимые для запуска сервиса.
type Config struct {
	HTTPAddress       string
	PublicBaseURL     string
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
	publicBaseURL, err := readPublicBaseURL(httpAddress)
	if err != nil {
		return Config{}, err
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
		PublicBaseURL:     publicBaseURL,
		DatabaseURL:       databaseURL,
		RateLimitRequests: rateLimitRequests,
		RateLimitWindow:   rateLimitWindow,
	}, nil
}

// DatabaseTarget возвращает адрес БД без учётных данных для журналов.
func DatabaseTarget(databaseURL string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Host == "" {
		return "неизвестно"
	}

	return parsed.Host + parsed.EscapedPath()
}

func readPublicBaseURL(httpAddress string) (string, error) {
	value := os.Getenv("PUBLIC_BASE_URL")
	if value == "" {
		return defaultPublicBaseURL(httpAddress), nil
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("переменная окружения PUBLIC_BASE_URL должна содержать HTTP(S)-адрес")
	}

	return strings.TrimRight(value, "/"), nil
}

func defaultPublicBaseURL(httpAddress string) string {
	host, port, err := net.SplitHostPort(httpAddress)
	if err != nil {
		return "http://localhost:8080"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	return "http://" + net.JoinHostPort(host, port)
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
