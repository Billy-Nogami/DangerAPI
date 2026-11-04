# Information Exchange Service

REST-сервис для хранения и получения сведений об угрозах. Контракт API описан в [openapi/openapi.yaml](openapi/openapi.yaml).

## Что реализовано

- создание, получение и полное обновление угрозы;
- валидация MD5, типа, критичности, текста и категорий;
- PostgreSQL и SQL-миграции;
- HTTP-ошибки `400`, `404`, `409`, `500` в JSON-формате;
- readiness-проверка `GET /health`;
- Swagger UI для интерактивной работы с API;
- `X-Request-ID` и журналы HTTP-запросов;
- структурированные журналы запуска, подключения к PostgreSQL и миграций;
- Docker Compose для локального запуска;
- unit-тесты домена, сценариев, HTTP- и PostgreSQL-слоя.

## OpenAPI и генерация кода

[openapi/openapi.yaml](openapi/openapi.yaml) — единственный источник правды для внешнего HTTP-контракта. Из него генерируются JSON-модели, интерфейс обработчиков и маршруты стандартного `net/http`.

```bash
make generate
```

Сгенерированный файл `internal/transport/generated/api.gen.go` вручную не редактируется. Проверка актуальности генерации:

```bash
make check-generate
```

После запуска сервиса спецификация доступна по адресу `http://localhost:8080/openapi.yaml`, а интерактивная документация Swagger UI — по адресу `http://localhost:8080/swagger/`. Swagger UI встроен в исполняемый файл и не требует доступа к внешним CDN.

## Быстрый старт

Нужны Docker с Docker Compose и `make`.

```bash
cp .env.example .env
make up
```

Команда поднимет PostgreSQL, применит миграции и запустит API на `http://localhost:8080`.

Открыть Swagger UI в браузере:

```text
http://localhost:8080/swagger/
```

Проверить состояние сервисов:

```bash
make ps
```

Посмотреть логи:

```bash
make logs
```

Отдельно посмотреть обращения к API или применение миграций:

```bash
make logs-api
make logs-migrate
```

Остановить окружение:

```bash
make down
```

Полностью удалить локальную БД и контейнеры:

```bash
make reset
```

`make reset` удаляет данные PostgreSQL.

## Переменные окружения

| Переменная | Значение по умолчанию | Назначение |
|---|---|---|
| `POSTGRES_DB` | `threat_service` | Имя базы данных |
| `POSTGRES_USER` | `threat_service` | Пользователь PostgreSQL |
| `POSTGRES_PASSWORD` | `threat_service` | Пароль PostgreSQL |
| `POSTGRES_PORT` | `5432` | Порт PostgreSQL на хосте |
| `HTTP_PORT` | `8080` | Порт API на хосте |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | Внешний адрес API, отображаемый в журнале запуска |
| `RATE_LIMIT_REQUESTS` | `60` | Число API-запросов с одного IP за окно |
| `RATE_LIMIT_WINDOW` | `1m` | Длительность окна ограничения |

Приложение получает `DATABASE_URL` и `HTTP_ADDRESS` из Docker Compose. При ручном запуске их нужно задать самостоятельно.

## Журналирование

Все журналы выводятся в `stdout`, поэтому при локальном запуске через Docker доступны командой `make logs`. Пароль из строки подключения к PostgreSQL в журнал не попадает.

- API при старте фиксирует параметры запуска, адрес прослушивания, подключение к PostgreSQL и URL API, healthcheck, OpenAPI и Swagger UI;
- обработанный HTTP-запрос содержит `request_id`, метод, путь, IP клиента, статус и длительность; тела запросов и заголовки не записываются;
- контейнер `migrate` фиксирует начало миграций, версию до применения, результат и итоговую версию.

Если меняется `HTTP_PORT`, укажите соответствующий внешний адрес в `PUBLIC_BASE_URL`, чтобы ссылки в сообщении о запуске оставались корректными.

## Примеры запросов

Создать угрозу:

```bash
curl --request POST http://localhost:8080/info \
  --header 'Content-Type: application/json' \
  --data '{
    "md5": "d41d8cd98f00b204e9800998ecf8427e",
    "name": "Trojan.Win32.Generic",
    "type": "trojan",
    "severity": "high",
    "description": "Вредоносная программа.",
    "categories": ["banker", "stealer"],
    "firstSeenAt": "2026-10-15T10:00:00Z"
  }'
```

Получить угрозу:

```bash
curl http://localhost:8080/info/d41d8cd98f00b204e9800998ecf8427e
```

Проверить готовность сервиса и доступность PostgreSQL:

```bash
curl http://localhost:8080/health
```

Обновить угрозу:

```bash
curl --request PUT http://localhost:8080/info/d41d8cd98f00b204e9800998ecf8427e \
  --header 'Content-Type: application/json' \
  --data '{
    "name": "Trojan.Win32.Banker",
    "type": "trojan",
    "severity": "critical",
    "description": "Обновлённая классификация.",
    "categories": ["banker", "credential-theft"],
    "firstSeenAt": "2026-10-15T10:00:00Z"
  }'
```

## Проверки

```bash
make check
```

Дополнительно можно проверить гонки данных:

```bash
make test-race
```

Интеграционные тесты поднимают временный контейнер PostgreSQL и проверяют API через HTTP:

```bash
make test-integration
```

## Структура

- `cmd/server` — точка входа;
- `internal/domain` — сущности и правила;
- `internal/usecase` — прикладные сценарии;
- `internal/repository` — контракт хранилища и PostgreSQL-адаптер;
- `internal/transport` — HTTP-слой;
- `migrations` — SQL-миграции;
- `openapi` — контракт API.
