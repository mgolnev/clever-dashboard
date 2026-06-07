#!/usr/bin/env bash
# Деплой на Amvera Cloud: push в git-репозиторий платформы → автосборка Dockerfile.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

REMOTE="${AMVERA_REMOTE:-amvera}"
BRANCH="${AMVERA_BRANCH:-main}"
URL="${AMVERA_URL:-https://clever-dashboard-golnev.amvera.io}"

if ! git remote get-url "$REMOTE" &>/dev/null; then
  echo "Remote '$REMOTE' не настроен. Пример:"
  echo "  git remote add amvera https://git.msk0.amvera.ru/<user>/clever-dashboard.git"
  exit 1
fi

echo "→ git push $REMOTE $BRANCH"
git push "$REMOTE" "$BRANCH"

echo ""
echo "Сборка на Amvera запущена. После завершения проверьте:"
echo "  curl -s $URL/api/health"
