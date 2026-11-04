package main

import (
	"errors"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"threat-service/internal/config"
	"threat-service/internal/logging"
)

const migrationsPath = "file:///migrations"

func main() {
	logger := logging.New(os.Stdout)
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("не задана строка подключения для миграций", "environment_variable", "DATABASE_URL")
		os.Exit(1)
	}

	logger.Info(
		"начато применение миграций",
		"database_target", config.DatabaseTarget(databaseURL),
		"migrations_path", migrationsPath,
	)
	migrator, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		logger.Error("не удалось подключиться к PostgreSQL для миграций", "error", err)
		os.Exit(1)
	}
	defer func() {
		if sourceError, databaseError := migrator.Close(); sourceError != nil || databaseError != nil {
			logger.Warn("не удалось корректно закрыть ресурсы мигратора", "source_error", sourceError, "database_error", databaseError)
		}
	}()

	logger.Info("подключение к PostgreSQL для миграций установлено")
	version, dirty, err := migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		logger.Error("не удалось прочитать состояние миграций", "error", err)
		os.Exit(1)
	}
	if dirty {
		logger.Error("обнаружена незавершённая миграция", "version", version)
		os.Exit(1)
	}

	logger.Info("проверено состояние миграций", "current_version", version)
	if err := migrator.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("миграции уже актуальны", "current_version", version)
			return
		}
		logger.Error("не удалось применить миграции", "error", err)
		os.Exit(1)
	}

	version, _, err = migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		logger.Error("миграции применены, но не удалось прочитать итоговую версию", "error", err)
		os.Exit(1)
	}
	logger.Info("миграции успешно применены", "current_version", version)
}
