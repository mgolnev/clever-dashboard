# Развёртывание (app.onreza.ru)

Приложение собирается в **единый артефакт**: Go-бэкенд отдаёт API (`/api/*`) и
собранный фронтенд (SPA) с одного порта. Рекомендуемый способ — Docker за
обратным прокси (nginx/Caddy) с TLS.

## Состав

- `Dockerfile` — multi-stage: сборка фронта (Vite), сборка статического Go-бинаря
  (без CGO, SQLite на чистом Go), компактный alpine-рантайм.
- `docker-compose.yml` — сервис `app` (+ опциональный `postgres` по профилю).
- `.env.example` — шаблон переменных окружения.
- CI: `.github/workflows/ci.yml` (тесты/сборка), `docker-publish.yml`
  (публикация образа в GHCR на push в `main` и теги `v*`).

## Переменные окружения

| Переменная | Назначение | Дефолт (в образе) |
|------------|-----------|-------------------|
| `PORT` | Порт HTTP-сервера | `8080` |
| `DB_DRIVER` | `sqlite` или `postgres` | `sqlite` |
| `DB_DSN` | Путь к файлу SQLite или DSN Postgres | `/app/data/clever.db` |
| `STATIC_DIR` | Каталог собранного фронта | `/app/web` |
| `LOGISTICS_PILOT_CITIES` | Города пилота (через запятую) | — |
| `LOGISTICS_PILOT_START` | Дата старта пилота `YYYY-MM-DD` | — |

SQLite-файл лежит в volume `clever-data` (`/app/data`) — данные переживают
пересоздание контейнера.

## Деплой на сервер

```bash
# 1. На сервере: клонировать репозиторий
git clone https://github.com/mgolnev/clever-dashboard.git
cd clever-dashboard

# 2. Подготовить окружение
cp .env.example .env
# при необходимости отредактировать .env

# 3. Собрать и запустить
docker compose up -d --build

# 4. Проверить здоровье
curl -s http://127.0.0.1:8080/api/health   # {"status":"ok"}
```

Приложение слушает `:8080` (по умолчанию). На хосте порт настраивается через
`APP_PORT` в `.env`.

### Готовый образ из GHCR (без сборки на сервере)

После первого пуша в `main` workflow опубликует образ
`ghcr.io/mgolnev/clever-dashboard:latest`. Тогда на сервере достаточно:

```bash
docker run -d --name clever-dashboard \
  -p 8080:8080 \
  -v clever-data:/app/data \
  --restart unless-stopped \
  ghcr.io/mgolnev/clever-dashboard:latest
```

## Обратный прокси и TLS (app.onreza.ru)

Терминируем TLS на прокси, проксируем на `127.0.0.1:8080`.

### Caddy (проще всего — авто-TLS Let's Encrypt)

```
app.onreza.ru {
    reverse_proxy 127.0.0.1:8080
}
```

### nginx

```nginx
server {
    listen 443 ssl http2;
    server_name app.onreza.ru;

    ssl_certificate     /etc/letsencrypt/live/app.onreza.ru/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/app.onreza.ru/privkey.pem;

    client_max_body_size 64m;   # выгрузки Битрикса бывают крупными

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

> DNS: A-запись `app.onreza.ru` → IP сервера. Файрвол: открыть 80/443.

## Postgres (опционально)

```bash
# в .env:
#   DB_DRIVER=postgres
#   DB_DSN=postgres://clever:clever@db:5432/clever?sslmode=disable
docker compose --profile postgres up -d --build
```

Схема создаётся миграциями при старте (`Migrate()`), отдельных шагов не нужно.

## Первая загрузка данных

Дашборд пустой до импорта. Откройте `https://app.onreza.ru`, загрузите выгрузку
заказов Битрикса (XLS/HTML/CSV) через UI — импорт идемпотентный (дедуп по номеру
заказа), повторная загрузка обновляет данные и заполняет новые поля (например
промокод).

## Обновление

```bash
git pull
docker compose up -d --build
# или, если используете образ из GHCR:
docker compose pull && docker compose up -d
```
