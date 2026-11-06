# Information Exchange Service

Тестовое задание: REST-сервис для хранения информации об угрозах. Сервис позволяет создавать, получать и обновлять записи по MD5. Для запуска используются Go, PostgreSQL, OpenAPI и Docker Compose.

## Возможности

- создание, получение и полное обновление сведений об угрозе;
- валидация MD5, типа, критичности, описания и категорий;
- запрет повторного создания записи с тем же MD5;
- PostgreSQL и версионируемые SQL-миграции;
- healthcheck, Swagger UI и отдача OpenAPI-спецификации;
- ограничение частоты запросов по IP;
- `X-Request-ID` и структурированные журналы;
- unit-, HTTP- и интеграционные тесты;
- воспроизводимый запуск через Docker Compose.

## Стек

| Область | Решение |
|---|---|
| Язык и HTTP | Go 1.26, стандартный `net/http` |
| Контракт API | OpenAPI 3.0.3, `oapi-codegen` |
| Хранилище | PostgreSQL 17, `pgx` |
| Миграции | `golang-migrate` |
| Документация API | Swagger UI |
| Контейнеризация | Docker и Docker Compose |

## Архитектура

| Каталог | Ответственность |
|---|---|
| `cmd/server` | Сборка зависимостей и запуск HTTP-сервера |
| `cmd/migrate` | Применение миграций с журналированием результата |
| `internal/domain` | Сущности угроз и бизнес-валидация |
| `internal/usecase` | Сценарии создания, получения и обновления угроз |
| `internal/repository` | Контракт хранилища и PostgreSQL-адаптер |
| `internal/transport` | HTTP-обработчики, middleware, rate limit и Swagger |
| `migrations` | Последовательные SQL-миграции |
| `openapi` | Исходная OpenAPI-спецификация |

Проект разделён на слои Clean Architecture: доменная модель и сценарии находятся отдельно от HTTP и PostgreSQL; адаптеры подключаются в `cmd/server` при запуске.

## Быстрый старт

Требуются Docker Desktop с Docker Compose и `make`.

```bash
cp .env.example .env
make hooks
make up
```

`make up` собирает сервис, запускает PostgreSQL, применяет миграции и поднимает API. Контейнеры стартуют в следующем порядке:

```text
db ──► healthcheck PostgreSQL ──► migrate ──► api ──► healthcheck API
```

API запускается только после успешного завершения миграций. По умолчанию внешний адрес сервиса — `http://localhost:8080`.

Если порты `8080` или `5432` уже заняты, измените в `.env` три значения согласованно:

```dotenv
HTTP_PORT=8083
POSTGRES_PORT=5434
PUBLIC_BASE_URL=http://localhost:8083
```

После запуска задайте адрес API и проверьте результат:

```bash
export API_URL=http://localhost:8080
curl "$API_URL/health"
make verify
```

Затем откройте в браузере `http://localhost:8080/swagger/`. При изменённом порте используйте значение `PUBLIC_BASE_URL` из `.env` вместо `http://localhost:8080`.

Для просмотра журналов используйте `make logs`.

Остановить сервисы, сохранив БД:

```bash
make down
```

Полностью удалить контейнеры и локальные данные PostgreSQL:

```bash
make reset
```

## Документация API

После запуска доступны:

| Адрес | Назначение |
|---|---|
| `/swagger/` | Интерактивная Swagger UI-документация |
| `/openapi.yaml` | Исходная OpenAPI 3.0-спецификация |
| `/health` | Readiness-проверка API и PostgreSQL |

Swagger UI встроен в исполняемый файл и не требует доступа к внешним CDN.

## HTTP API

| Метод | Путь | Назначение | Успешный ответ |
|---|---|---|---|
| `POST` | `/info` | Создать угрозу | `201 Created` |
| `GET` | `/info/{md5}` | Получить угрозу по MD5 | `200 OK` |
| `PUT` | `/info/{md5}` | Полностью обновить угрозу | `200 OK` |
| `GET` | `/health` | Проверить готовность сервиса | `200 OK` |

`md5` должен содержать ровно 32 шестнадцатеричных символа. При создании он уникален.

### Модель угрозы

| Поле | Тип | Обязательное | Описание |
|---|---|---|---|
| `md5` | string | при создании | MD5-хеш файла |
| `name` | string | да | Название угрозы |
| `type` | enum | да | `trojan`, `worm`, `ransomware`, `adware`, `spyware`, `other` |
| `severity` | enum | да | `low`, `medium`, `high`, `critical` |
| `description` | string | нет | Описание, до 4 000 символов |
| `categories` | string[] | нет | Не более 20 уникальных категорий |
| `firstSeenAt` | RFC 3339 date-time | нет | Время первого обнаружения |
| `createdAt` | RFC 3339 date-time | в ответе | Время создания записи |
| `updatedAt` | RFC 3339 date-time | в ответе | Время последнего обновления |

### Пример создания угрозы

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

Ошибки всегда отдаются в JSON-формате:

```json
{
  "message": "Угроза не найдена",
  "details": "Дополнительное описание, если оно нужно"
}
```

Возможные статусы: `400` — неверные данные, `404` — запись не найдена, `409` — MD5 уже существует, `429` — превышен лимит, `500` — внутренняя ошибка.

## OpenAPI и генерация кода

[`openapi/openapi.yaml`](openapi/openapi.yaml) — единственный источник правды внешнего контракта. Из него генерируются модели JSON, интерфейс обработчиков и маршрутизация HTTP-слоя.

```bash
go generate ./internal/transport/generated
```

Файл `internal/transport/generated/api.gen.go` вручную не редактируется. После изменения спецификации нужно выполнить команду выше и добавить обновлённый файл в тот же коммит. `make check` и pre-commit hook проверяют актуальность генерации автоматически.

## Миграции

При `make up` отдельный контейнер `migrate` применяет SQL-файлы из `migrations` до запуска API.

## Журналирование

Все журналы выводятся в `stdout`, поэтому доступны через Docker Compose:

```bash
make logs
```

Сервис журналирует запуск, подключение к PostgreSQL, URL API, Swagger и OpenAPI. Каждая HTTP-строка содержит `request_id`, метод, путь, IP клиента, статус и длительность. Тела запросов, заголовки и пароль БД в журналы не попадают.

## Проверки и pre-commit

Проверка исходного кода:

```bash
make check
```

Она проверяет форматирование `gofmt`, актуальность сгенерированного OpenAPI-кода, unit- и HTTP-тесты, а также выполняет `go vet`.

Проверка уже поднятого Docker-окружения без изменения данных:

```bash
make verify
```

Команда проверяет `/health` внутри контейнера API — тем самым подтверждает доступность API и PostgreSQL — а также проверяет доступность Swagger UI.

Pre-commit hook не требует сторонних менеджеров пакетов. Он подключается один раз на рабочем компьютере:

```bash
make hooks
```

После этого перед каждым `git commit` автоматически запускается `make check`. Коммит не будет создан, пока не пройдут форматирование, генерация, тесты и `go vet`.

## Конфигурация

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `POSTGRES_DB` | `threat_service` | Имя базы данных |
| `POSTGRES_USER` | `threat_service` | Пользователь PostgreSQL |
| `POSTGRES_PASSWORD` | `threat_service` | Пароль PostgreSQL |
| `POSTGRES_PORT` | `5432` | Порт PostgreSQL на хосте |
| `HTTP_PORT` | `8080` | Порт API на хосте |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | Внешний URL в журналах запуска |
| `RATE_LIMIT_REQUESTS` | `60` | Число запросов с одного IP за окно |
| `RATE_LIMIT_WINDOW` | `1m` | Длительность окна rate limit |

`DATABASE_URL` и `HTTP_ADDRESS` формируются Docker Compose автоматически и не должны добавляться в `.env` для обычного локального запуска.

## Команды Make

| Команда | Назначение |
|---|---|
| `make hooks` | Подключить pre-commit hook |
| `make up` / `make down` | Запустить / остановить окружение |
| `make reset` | Удалить окружение и данные БД |
| `make verify` | Проверить health API, PostgreSQL и Swagger UI |
| `make logs` | Следить за журналами всех сервисов |
| `make check` | Запустить обязательные проверки |

## Что можно добавить далее

Авторизация в рамках задания не требовалась, поэтому в текущую версию она не входит. При развитии сервиса можно добавить:

1. Аутентификацию сервисов через API-ключи или OAuth 2.0 Client Credentials.
2. Авторизацию пользователей через OIDC/JWT и роли, например `reader` для получения данных и `editor` для создания или изменения угроз.
