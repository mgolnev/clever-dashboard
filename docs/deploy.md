# Развёртывание

Приложение собирается в **единый артефакт**: Go-бэкенд отдаёт API (`/api/*`) и
собранный фронтенд (SPA) с одного порта (single-binary в одном контейнере).

Способы (по возрастанию ручной работы):

1. **Amvera Cloud** — деплой по `git push`, платформа сама собирает наш
   `Dockerfile`, выдаёт HTTPS-домен и постоянное хранилище. См.
   [раздел Amvera](#деплой-на-amvera-cloud) ниже. **Рекомендуется**, если нет
   своего сервера.
2. **Свой VPS** — Docker за обратным прокси (Caddy/nginx) с TLS.
3. **Готовый образ из GHCR** — `docker run` где угодно.

> Важно: **Onreza** (Vercel-подобная платформа) умеет только статику и
> Node.js/Bun — Go-рантайма/Dockerfile там нет, поэтому весь бэкенд на ней не
> запускается. На Onreza можно держать максимум статику фронта и проксировать
> `/api/*` на внешний бэкенд через Routing Rules → Rewrites.

## Состав

- `Dockerfile` — multi-stage: сборка фронта (Vite), сборка статического Go-бинаря
  (без CGO, SQLite на чистом Go), компактный alpine-рантайм.
- `docker-entrypoint.sh` — стартует от root, чинит права на смонтированный `/data`
  и сбрасывает привилегии до пользователя `app`.
- `amvera.yml` — конфиг для Amvera Cloud (Docker, порт 8080, хранилище `/data`).
- `docker-compose.yml` — сервис `app` (+ опциональный `postgres` по профилю).
- `.env.example` — шаблон переменных окружения.
- CI: `.github/workflows/ci.yml` (тесты/сборка), `docker-publish.yml`
  (публикация образа в GHCR на push в `main` и теги `v*`).

## Переменные окружения

| Переменная | Назначение | Дефолт (в образе) |
|------------|-----------|-------------------|
| `PORT` | Порт HTTP-сервера | `8080` |
| `DB_DRIVER` | `sqlite` или `postgres` | `sqlite` |
| `DB_DSN` | Путь к файлу SQLite или DSN Postgres | `/data/clever.db` |
| `STATIC_DIR` | Каталог собранного фронта | `/app/web` |
| `LOGISTICS_PILOT_CITIES` | Города пилота (через запятую) | — |
| `LOGISTICS_PILOT_START` | Дата старта пилота `YYYY-MM-DD` | — |

SQLite-файл лежит в постоянном хранилище `/data` (volume `clever-data` в Docker,
`persistenceMount` в Amvera) — данные переживают пересоздание/пересборку.

## Деплой на Amvera Cloud

Amvera собирает наш `Dockerfile` сама; конфиг — `amvera.yml` (уже в репозитории):
порт `8080`, постоянное хранилище `/data` (туда пишется SQLite).

1. Зарегистрируйтесь на [amvera.ru](https://amvera.ru), создайте проект
   (тип — **из Git / загрузка**).
2. Залейте код в git-репозиторий Amvera (или подключите внешний). По первому
   пушу Amvera найдёт `amvera.yml`/`Dockerfile`, соберёт и запустит контейнер:

   ```bash
   git remote add amvera https://git.amvera.ru/<username>/<project>.git
   git push amvera main
   ```

3. В разделе **Настройки → Домены** привяжите домен:
   - бесплатный поддомен Amvera (HTTPS «из коробки»), либо
   - свой домен (например `app.onreza.ru`) — добавьте указанную DNS-запись
     (CNAME/A) у регистратора зоны.
4. Переменные окружения (опционально) — в **Настройки → Переменные**:
   `LOGISTICS_PILOT_CITIES`, `LOGISTICS_PILOT_START`. Менять `DB_DSN` не нужно —
   дефолт `/data/clever.db` уже указывает в постоянное хранилище.
5. После старта откройте домен и загрузите выгрузку Битрикса через UI.

> SQLite пишется **только** в `/data` — это требование Amvera (папка `data/` в
> коде и хранилище `/data` — разные вещи; данные вне `/data` теряются при
> обновлении проекта).

## Деплой на свой VPS

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
  -v clever-data:/data \
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
