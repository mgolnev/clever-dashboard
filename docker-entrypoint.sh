#!/bin/sh
# Запуск от root: процесс должен слушать привилегированный порт 80 (дефолт
# маршрутизации Amvera) и иметь право писать в смонтированное хранилище /data.
set -e

DB_DSN="${DB_DSN:-/data/clever.db}"
case "$DB_DSN" in
  postgres://*|postgresql://*) ;;
  *) mkdir -p "$(dirname "$DB_DSN")" 2>/dev/null || true ;;
esac

exec /app/server
