# CLEVER Dashboard

Дашборд для руководителя развития интернет-магазина **CleverWear.ru**: недельный
обзор результатов на основе данных из Битрикса (далее — Яндекс Метрика и
Аппметрика для воронки трафика и конверсии).

MVP-1 закрывает **источник «Битрикс» через загрузку файла выгрузки заказов**.

## Возможности (MVP-1)

- Загрузка выгрузки заказов Битрикса (XLS/HTML или CSV) через UI.
- Идемпотентный импорт с дедупом по номеру заказа.
- Товарная детализация из столбца «Позиции» (бренд/категория/пол/размер).
- Дашборд с выбором периода и **сравнением с предыдущим периодом той же длины**:
  - KPI: выручка, заказы, средний чек (AOV), **средняя цена позиции (ASP)**, оплачиваемость, отмены, продано товаров, покупатели;
  - воронка по статусам;
  - срезы: канал (приложение/сайт), оплата, доставка, регионы;
  - товары: по категориям, полу, брендам, топ по выручке.
- Вкладка **«Воронки»**: путь заказа (гросс → оплата → сборка → отправка → доставка → выкуп)
  с точками отвала и разрезами по оплате/доставке/каналу/региону, топ проблем сборки и причин отмены
  (см. [docs/funnel-analysis.md](docs/funnel-analysis.md)).
- Вкладка **«Логистика»**: KPI доставки, пилот vs контроль по городам, службы, недельная динамика
  (`LOGISTICS_PILOT_CITIES`, `LOGISTICS_PILOT_START` — см. [docs/api.md](docs/api.md)).

## Стек

- **Backend:** Go + Fiber, слои handler → service → repository, DI-контейнер.
- **БД:** SQLite по умолчанию (без зависимостей), Postgres через env.
- **Frontend:** React + TypeScript + Tailwind (Vite), порт 3000, проксирование на 8080.

## Запуск

```bash
# Backend (:8080) — SQLite-файл создаётся в data/clever.db
make run

# Frontend (:3000)
make frontend-install   # один раз
make frontend-start
```

Открыть http://localhost:3000, загрузить файл выгрузки заказов Битрикса.

### Postgres (опционально)

```bash
export DB_DRIVER=postgres
export DB_DSN="postgres://user:pass@localhost:5432/clever_dashboard?sslmode=disable"
make run
```

## Структура

```
cmd/server            — точка входа
internal/config       — конфигурация (env)
internal/db           — подключение, миграции, диалект (SQLite/Postgres)
internal/model        — нейтральные доменные типы
internal/ingestion    — приём файла Битрикса (парсинг HTML/CSV + позиции)
internal/normalize    — нормализация (деньги, гео, статусы, товар)
internal/services/orders   — импорт и витрина заказов
internal/services/metrics  — KPI, срезы, сравнение периодов
internal/services/funnel   — воронка пути заказа и разрезы
internal/handlers     — HTTP-слой (Fiber)
internal/container    — DI
frontend/             — React + TS + Tailwind
docs/                 — архитектура, API, ADR
```

## Документация

- [docs/architecture.md](docs/architecture.md) — архитектура и слои.
- [docs/bitrix-fields.md](docs/bitrix-fields.md) — словарь всех полей выгрузки Битрикса.
- [docs/funnel-analysis.md](docs/funnel-analysis.md) — модель воронки и аналитические выводы.
- [docs/import-bitrix.md](docs/import-bitrix.md) — контракт импорта и разбор позиций.
- [docs/api.md](docs/api.md) — HTTP API.
- [docs/adr/README.md](docs/adr/README.md) — архитектурные решения.
- [docs/ROADMAP.md](docs/ROADMAP.md) — план развития.

## Тесты

```bash
go test ./... -count=1
cd frontend && npx tsc --noEmit
```
