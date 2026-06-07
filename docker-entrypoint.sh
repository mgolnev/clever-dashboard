#!/bin/sh
# Стартуем от root: гарантируем, что каталог под SQLite (постоянное хранилище,
# смонтированное платформой как root) доступен на запись пользователю app, и
# только потом сбрасываем привилегии. Для Postgres шаг с /data безвреден.
set -e

DB_DSN="${DB_DSN:-/data/clever.db}"
case "$DB_DSN" in
  postgres://*|postgresql://*) DATA_DIR="" ;;
  *) DATA_DIR="$(dirname "$DB_DSN")" ;;
esac

if [ -n "$DATA_DIR" ]; then
  mkdir -p "$DATA_DIR" 2>/dev/null || true
  chown -R app:app "$DATA_DIR" 2>/dev/null || true
fi

# Amvera по умолчанию выставляет PORT=80; для не-root процесса нужен >=1024.
export PORT="${PORT:-8080}"
case "$PORT" in
  80|443) PORT=8080 ;;
esac

exec su-exec app:app /app/server
