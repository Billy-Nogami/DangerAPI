package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"threat-service/internal/config"
	"threat-service/internal/repository/postgres"
	"threat-service/internal/transport"
	"threat-service/internal/usecase"
)

func main() {
	configuration, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, configuration.DatabaseURL)
	if err != nil {
		log.Fatalf("не удалось создать пул соединений PostgreSQL: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("не удалось подключиться к PostgreSQL: %v", err)
	}

	threatService := usecase.NewThreatService(
		postgres.NewThreatRepository(pool),
		usecase.SystemClock{},
	)
	server := &http.Server{
		Addr:              configuration.HTTPAddress,
		Handler:           transport.NewHTTPHandler(threatService),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errorsChannel := make(chan error, 1)
	go func() {
		log.Printf("HTTP-сервер слушает %s", configuration.HTTPAddress)
		errorsChannel <- server.ListenAndServe()
	}()

	select {
	case err := <-errorsChannel:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP-сервер остановлен с ошибкой: %v", err)
		}
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("не удалось корректно остановить HTTP-сервер: %v", err)
		}
	}
}
