.DEFAULT_GOAL := help

.PHONY: help up down reset logs ps migrate test test-race vet format check

help:
	@printf "Доступные команды:\n"
	@printf "  make up        Запустить PostgreSQL, миграции и API\n"
	@printf "  make down      Остановить контейнеры\n"
	@printf "  make reset     Удалить контейнеры и данные PostgreSQL\n"
	@printf "  make logs      Показать логи всех сервисов\n"
	@printf "  make ps        Показать состояние сервисов\n"
	@printf "  make migrate   Повторно применить миграции\n"
	@printf "  make test      Запустить unit-тесты\n"
	@printf "  make test-race Запустить тесты с проверкой гонок\n"
	@printf "  make vet       Запустить статический анализ Go\n"
	@printf "  make format    Отформатировать Go-код\n"
	@printf "  make check     Проверить форматирование, тесты и vet\n"

up:
	docker compose up --build -d

down:
	docker compose down

reset:
	docker compose down --volumes --remove-orphans

logs:
	docker compose logs --follow

ps:
	docker compose ps

migrate:
	docker compose run --rm migrate

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

format:
	gofmt -w $$(rg --files -g '*.go')

check:
	@test -z "$$(gofmt -l $$(rg --files -g '*.go'))"
	@go test ./...
	@go vet ./...
