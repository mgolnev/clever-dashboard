#!/usr/bin/env bash
# Перезапуск dev-сервисов. Использование: ./scripts/restart-dev.sh [backend|frontend|all]
set -euo pipefail
cd "$(dirname "$0")/.."

restart_backend() {
  echo "[restart] backend :8080"
  lsof -ti:8080 | xargs kill -9 2>/dev/null || true
  nohup go run ./cmd/server >/tmp/clever-backend.log 2>&1 &
  echo "[restart] backend started, log: /tmp/clever-backend.log"
}

restart_frontend() {
  echo "[restart] frontend :3000"
  lsof -ti:3000 | xargs kill -9 2>/dev/null || true
  ( cd frontend && nohup npm start >/tmp/clever-frontend.log 2>&1 & )
  echo "[restart] frontend started, log: /tmp/clever-frontend.log"
}

case "${1:-all}" in
  backend) restart_backend ;;
  frontend) restart_frontend ;;
  all) restart_backend; restart_frontend ;;
  *) echo "usage: $0 [backend|frontend|all]"; exit 1 ;;
esac
