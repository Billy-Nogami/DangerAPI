package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"threat-service/internal/config"
	"threat-service/internal/logging"
	"threat-service/internal/repository/postgres"
	"threat-service/internal/transport"
	"threat-service/internal/usecase"
)

func main() {
	logger := logging.New(os.Stdout)
	configuration, err := config.Load()
	if err != nil {
		logger.Error("не удалось загрузить конфигурацию", "error", err)
		os.Exit(1)
	}
	logger.Info(
		"загружена конфигурация сервиса",
		"http_address", configuration.HTTPAddress,
		"public_base_url", configuration.PublicBaseURL,
		"rate_limit_requests", configuration.RateLimitRequests,
		"rate_limit_window", configuration.RateLimitWindow,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("подключение к PostgreSQL", "database_target", config.DatabaseTarget(configuration.DatabaseURL))
	pool, err := pgxpool.New(ctx, configuration.DatabaseURL)
	if err != nil {
		logger.Error("не удалось создать пул соединений PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer func() {
		pool.Close()
		logger.Info("пул соединений PostgreSQL закрыт")
	}()

	pingContext, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	defer cancelPing()
	if err := pool.Ping(pingContext); err != nil {
		logger.Error("не удалось подключиться к PostgreSQL", "error", err)
		os.Exit(1)
	}
	logger.Info("подключение к PostgreSQL установлено", "database_target", config.DatabaseTarget(configuration.DatabaseURL))

	threatService := usecase.NewThreatService(
		postgres.NewThreatRepository(pool),
		usecase.SystemClock{},
	)
	server := &http.Server{
		Addr: configuration.HTTPAddress,
		Handler: transport.NewHTTPHandler(
			logger,
			threatService,
			pool,
			transport.NewRateLimiter(configuration.RateLimitRequests, configuration.RateLimitWindow),
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	listener, err := net.Listen("tcp", configuration.HTTPAddress)
	if err != nil {
		logger.Error("не удалось запустить HTTP-сервер", "http_address", configuration.HTTPAddress, "error", err)
		os.Exit(1)
	}

	errorsChannel := make(chan error, 1)
	go func() {
		errorsChannel <- server.Serve(listener)
	}()
	logger.Info(
		"HTTP-сервис запущен",
		"listen_address", configuration.HTTPAddress,
		"api_url", configuration.PublicBaseURL,
		"health_url", configuration.PublicBaseURL+"/health",
		"openapi_url", configuration.PublicBaseURL+"/openapi.yaml",
		"swagger_url", configuration.PublicBaseURL+"/swagger/",
	)

	select {
	case err := <-errorsChannel:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP-сервер остановлен с ошибкой", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("получен сигнал остановки HTTP-сервера")
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			logger.Error("не удалось корректно остановить HTTP-сервер", "error", err)
			return
		}
		logger.Info("HTTP-сервер корректно остановлен")
	}
}
