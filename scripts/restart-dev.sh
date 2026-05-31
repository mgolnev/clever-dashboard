#!/usr/bin/env bash
# Перезапуск dev-сервисов. Использование: ./scripts/restart-dev.sh [backend|frontend|all]
#
# Процессы запускаются в ОТДЕЛЬНОЙ сессии (setsid / os.setsid), чтобы пережить
# завершение родительской оболочки: иначе фоновые `go run`/`npm start` падают
# вместе с временной сессией, из которой их стартовали (порты 8080/3000 пустеют).
set -euo pipefail
cd "$(dirname "$0")/.."

PID_DIR="/tmp"

# spawn_detached <pid-файл> <лог-файл> <команда...>
# Запускает команду как лидера новой сессии (новый pgid), отвязанной от текущей
# оболочки и группы процессов, с перенаправлением вывода в лог.
spawn_detached() {
  local pidfile="$1" log="$2"; shift 2
  if command -v setsid >/dev/null 2>&1; then
    setsid bash -c 'exec "$@" >"'"$log"'" 2>&1' _ "$@" &
    echo $! >"$pidfile"
  else
    # macOS: setsid отсутствует — отвязываемся через os.setsid() и печатаем PID
    # реального процесса (после exec) в pid-файл.
    python3 - "$pidfile" "$log" "$@" <<'PY'
import os, sys
pidfile, log, args = sys.argv[1], sys.argv[2], sys.argv[3:]
pid = os.fork()
if pid > 0:
    # Родитель: фиксируем PID лидера сессии и сразу выходим.
    with open(pidfile, "w") as f:
        f.write(str(pid))
    os._exit(0)
os.setsid()
fd = os.open(log, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o644)
os.dup2(fd, 1)
os.dup2(fd, 2)
os.close(0)
os.execvp(args[0], args)
PY
  fi
}

kill_port() {
  lsof -ti:"$1" | xargs kill -9 2>/dev/null || true
}

restart_backend() {
  echo "[restart] backend :8080"
  kill_port 8080
  spawn_detached "$PID_DIR/clever-backend.pid" "$PID_DIR/clever-backend.log" go run ./cmd/server
  echo "[restart] backend started (pid $(cat "$PID_DIR/clever-backend.pid" 2>/dev/null || echo '?')), log: $PID_DIR/clever-backend.log"
}

restart_frontend() {
  echo "[restart] frontend :3000"
  kill_port 3000
  spawn_detached "$PID_DIR/clever-frontend.pid" "$PID_DIR/clever-frontend.log" npm --prefix frontend start
  echo "[restart] frontend started (pid $(cat "$PID_DIR/clever-frontend.pid" 2>/dev/null || echo '?')), log: $PID_DIR/clever-frontend.log"
}

case "${1:-all}" in
  backend) restart_backend ;;
  frontend) restart_frontend ;;
  all) restart_backend; restart_frontend ;;
  *) echo "usage: $0 [backend|frontend|all]"; exit 1 ;;
esac
