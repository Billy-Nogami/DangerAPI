.DEFAULT_GOAL := help

.PHONY: help up down reset logs ps migrate test test-race test-integration generate check-generate vet format check

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
	@printf "  make test-integration Запустить интеграционные тесты с PostgreSQL в Docker\n"
	@printf "  make generate  Сгенерировать HTTP-код из OpenAPI\n"
	@printf "  make check-generate Проверить актуальность сгенерированного кода\n"
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

test-integration:
	go test -tags=integration ./tests/integration

generate:
	go generate ./internal/transport/generated

check-generate:
	@$(MAKE) generate
	@git diff --exit-code -- internal/transport/generated

vet:
	go vet ./...

format:
	gofmt -w $$(rg --files -g '*.go')

check:
	@test -z "$$(gofmt -l $$(rg --files -g '*.go'))"
	@$(MAKE) check-generate
	@go test ./...
	@go vet ./...
