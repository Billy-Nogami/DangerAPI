.DEFAULT_GOAL := help

.PHONY: help hooks up down reset logs verify check

help:
	@printf "Доступные команды:\n"
	@printf "  make hooks     Подключить локальный pre-commit hook\n"
	@printf "  make up        Запустить PostgreSQL, миграции и API\n"
	@printf "  make down      Остановить контейнеры\n"
	@printf "  make reset     Удалить контейнеры и данные PostgreSQL\n"
	@printf "  make logs      Показать журналы сервисов\n"
	@printf "  make verify    Проверить API, PostgreSQL и Swagger\n"
	@printf "  make check     Проверить код, генерацию, тесты и vet\n"

hooks:
	git config core.hooksPath .githooks
	@printf "Pre-commit hook подключён. Перед коммитом будет выполняться make check.\n"

up:
	docker compose up --build -d

down:
	docker compose down

reset:
	docker compose down --volumes --remove-orphans

logs:
	docker compose logs --follow

verify:
	@docker compose exec -T api wget -q -O - http://localhost:8080/health
	@docker compose exec -T api wget -q -O /dev/null http://localhost:8080/swagger/
	@printf "API, PostgreSQL и Swagger UI доступны.\n"

check:
	@test -z "$$(gofmt -l $$(rg --files -g '*.go'))"
	@go generate ./internal/transport/generated
	@git diff --exit-code -- internal/transport/generated
	@go test ./...
	@go vet ./...
