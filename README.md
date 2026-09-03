# notification-service

Отправка уведомлений пользователям в Telegram по HTTP API. Клиент шлёт
запрос с `telegram_id`, именем шаблона и параметрами — сервис ставит
уведомление в очередь и асинхронно доставляет его через Telegram-бота,
с ретраями и трекингом статуса.

## Как это работает

```
POST /api/v1/notifications
        │
        ▼
   Postgres (notifications: status=pending)
        │
        ▼
   Redis Stream (очередь)
        │
        ▼
   Worker (consumer group) ──► Telegram Bot API
        │
        ▼
   Postgres (status: processing → sent/failed, + notification_logs)
```

- `users` — телеграм-пользователи, привязка по `telegram_id` (создаются
  автоматически при первом уведомлении).
- `templates` — именованные шаблоны сообщений (`text/template` синтаксис,
  например `Привет, {{.name}}!`).
- `notifications` — заявки на отправку со статусом и параметрами (jsonb).
- `notification_logs` — история смены статусов.
- Redis Stream — очередь между API и воркером; воркер обрабатывает её
  через consumer group, так что рестарт воркера не теряет сообщения.

## Быстрый старт

```bash
cp .env.example .env
# заполнить POSTGRES_PASSWORD, REDIS_PASSWORD, TELEGRAM_BOT_TOKEN

docker compose up -d postgres redis
go run ./cmd/migrate           # применить миграции

go run ./cmd/server            # поднять API + воркер
```

Либо целиком в Docker: `docker compose up -d --build` (сервис `app`
сам применяет миграции при старте и слушает `:8080`).

`TELEGRAM_BOT_TOKEN` — токен, выданный [@BotFather](https://t.me/BotFather).
Получатель уведомления должен хотя бы раз написать боту (или быть
добавлен в чат с ним) — иначе Telegram не даст отправить ему сообщение.

## API

Swagger UI: **http://localhost:8080/docs** (сам спек — `GET /openapi.json`).
Страница загружает swagger-ui из CDN, спецификация лежит в
`docs/openapi.json` (правится руками при добавлении новых эндпоинтов).

Если задан `APP_API_KEY` (`app.api_key` / env `APP_API_KEY`), все запросы
к `/api/v1/*` должны нести заголовок `X-API-Key`. Пустое значение
отключает проверку (по умолчанию, для локальной разработки).

### Создать шаблон

```bash
curl -X POST localhost:8080/api/v1/templates \
  -H 'Content-Type: application/json' \
  -d '{"name": "welcome", "body": "Привет, {{.name}}! Добро пожаловать."}'
```

### Список шаблонов

```bash
curl localhost:8080/api/v1/templates
```

### Отправить уведомление

```bash
curl -X POST localhost:8080/api/v1/notifications \
  -H 'Content-Type: application/json' \
  -d '{"telegram_id": 123456789, "template": "welcome", "params": {"name": "Алиса"}}'
```

Ответ `202 Accepted` со статусом `pending`. Пользователь с таким
`telegram_id` будет создан автоматически, если его ещё нет.

### Статус уведомления

```bash
curl localhost:8080/api/v1/notifications/<id>
```

`status`: `pending` → `processing` → `sent` | `failed`. При ошибке
отправки `error_message` содержит причину (сохраняется после исчерпания
`worker.retry_attempts`).

## Конфигурация

`configs/config.yaml`, значения переопределяются переменными окружения
(см. `.env.example`): `POSTGRES_*`, `REDIS_PASSWORD`, `REDIS_HOST`,
`DATABASE_HOST`, `TELEGRAM_BOT_TOKEN`, `APP_ENV`, `APP_API_KEY`.

`worker.concurrency` — число параллельных consumer-горутин;
`worker.retry_attempts` / `retry_delay_seconds` — ретраи отправки в
Telegram перед пометкой `failed`.

## Структура

```
cmd/server        — точка входа: HTTP API + воркер в одном процессе
cmd/migrate        — применение/откат SQL-миграций (goose)
internal/config     — загрузка конфига
internal/model       — доменные типы
internal/store        — репозитории Postgres (users/templates/notifications)
internal/queue          — очередь на Redis Streams
internal/telegram         — клиент Telegram Bot API
internal/render             — рендер шаблона (text/template) с params
internal/notification          — оркестрация: создание уведомления → очередь
internal/worker                   — обработка очереди → отправка → статус
internal/api                        — HTTP-хендлеры
migrations                            — SQL-миграции (goose)
docs                                    — OpenAPI-спек + страница Swagger UI
```

## Известные ограничения / что дальше

- Статус `delivered` зарезервирован под будущую интеграцию с delivery
  receipts — сейчас успешная отправка помечается сразу как `sent`.
- Нет пагинации в `GET /api/v1/templates` и списка уведомлений по
  пользователю — добавить по мере необходимости.
- `telegram.chat_ids` в конфиге не используется кодом — было
  заготовкой под broadcast-рассылку, оставлено на будущее.
