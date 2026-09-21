#!/usr/bin/env bash
# One-click Laya deploy (Linux / macOS): Go deploy server + model service.
set -euo pipefail

PORT=7710
BIND="0.0.0.0"
SKIP_TESTS=0
FOREGROUND=0

usage() {
  cat <<'EOF'
Usage: deploy/deploy.sh [--Port 7710] [--bind 127.0.0.1] [--skip-tests] [--foreground]

Builds bin/laya-deploy, bin/laya, bin/laya-mcp, starts laya-deploy on PORT,
waits for /health and /deploy/status.

Env for clients: LAYA_URL=http://127.0.0.1:7710
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --port) PORT="$2"; shift 2 ;;
    --bind) BIND="$2"; shift 2 ;;
    --skip-tests) SKIP_TESTS=1; shift ;;
    --foreground|-f) FOREGROUND=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown arg: $1" >&2; usage; exit 2 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "$REPO_ROOT"

step() { echo "[laya-deploy] $*"; }
ok() { echo "[laya-deploy] $*"; }
die() { echo "[laya-deploy] $*" >&2; exit 1; }

if ! command -v go >/dev/null 2>&1; then
  die "Go toolchain not found on PATH"
fi
step "using $(command -v go) ($(go version))"

BIN_DIR="${REPO_ROOT}/bin"
mkdir -p "$BIN_DIR"

if [[ "$SKIP_TESTS" -ne 1 ]]; then
  step "go test ./..."
  go test ./...
fi

step "building binaries"
go build -o "${BIN_DIR}/laya-deploy" ./deploy
go build -o "${BIN_DIR}/laya" ./cli
go build -o "${BIN_DIR}/laya-mcp" ./mcp
ok "binaries ready in ${BIN_DIR}"

ADDR="${BIND}:${PORT}"
URL="http://${ADDR}"
EXE="${BIN_DIR}/laya-deploy"
LOG="${BIN_DIR}/laya-deploy.log"
PID_FILE="${BIN_DIR}/laya-deploy.pid"
HEALTH_URL="${URL}/health"
STATUS_URL="${URL}/deploy/status"

fetch() {
  local u="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -fsS --max-time 2 "$u" 2>/dev/null || true
  else
    wget -qO- --timeout=2 "$u" 2>/dev/null || true
  fi
}

is_healthy() {
  local h s
  h="$(fetch "$HEALTH_URL")"
  s="$(fetch "$STATUS_URL")"
  echo "$h" | grep -q '"status":"ok"' && echo "$s" | grep -q '"status":"ok"'
}

stop_previous() {
  step "stopping previous laya-deploy on ${ADDR} (if any)"
  if [[ -f "$PID_FILE" ]]; then
    old_pid="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [[ -n "${old_pid}" ]] && kill -0 "$old_pid" 2>/dev/null; then
      kill "$old_pid" 2>/dev/null || true
      sleep 0.2
      kill -9 "$old_pid" 2>/dev/null || true
    fi
  fi
  pkill -f "laya-deploy -addr ${ADDR}" 2>/dev/null || true
  pkill -f "laya-deploy -port ${PORT}" 2>/dev/null || true
}

if [[ "$FOREGROUND" -ne 1 ]] && is_healthy; then
  ok "already healthy on ${URL}; reusing"
  cat <<EOF

  URL          : ${URL}
  Health       : ${HEALTH_URL}
  Deploy status: ${STATUS_URL}
  Agent env    : export LAYA_URL='${URL}'
EOF
  exit 0
fi

stop_previous

if [[ "$FOREGROUND" -eq 1 ]]; then
  ok "starting laya-deploy in foreground on ${URL}"
  echo "LAYA_URL=${URL}"
  exec "$EXE" -addr "$ADDR"
fi

step "starting laya-deploy on ${URL}"
: >"$LOG"
nohup "$EXE" -addr "$ADDR" >>"$LOG" 2>&1 &
echo $! >"$PID_FILE"
DEPLOY_PID="$(cat "$PID_FILE")"

ok_url_health="${URL}/health"
ok_url_status="${URL}/deploy/status"
deadline=$((SECONDS + 15))
ok_flag=0
while (( SECONDS < deadline )); do
  if ! kill -0 "$DEPLOY_PID" 2>/dev/null; then
    die "laya-deploy exited early. See ${LOG}
$(tail -n 40 "$LOG" 2>/dev/null || true)"
  fi
  if command -v curl >/dev/null 2>&1; then
    h="$(curl -fsS --max-time 2 "$ok_url_health" 2>/dev/null || true)"
    s="$(curl -fsS --max-time 2 "$ok_url_status" 2>/dev/null || true)"
  else
    h="$(wget -qO- --timeout=2 "$ok_url_health" 2>/dev/null || true)"
    s="$(wget -qO- --timeout=2 "$ok_url_status" 2>/dev/null || true)"
  fi
  if echo "$h" | grep -q '"status":"ok"' && echo "$s" | grep -q '"status":"ok"'; then
    ok_flag=1
    break
  fi
  sleep 0.25
done

if [[ "$ok_flag" -ne 1 ]]; then
  die "health check timeout for ${ok_url_health}. Log: ${LOG}
$(tail -n 40 "$LOG" 2>/dev/null || true)"
fi

ok "Laya deploy server is up"
cat <<EOF

  URL          : ${URL}
  PID          : ${DEPLOY_PID}
  Health       : ${ok_url_health}
  Deploy status: ${ok_url_status}
  Models API   : ${URL}/v1/models
  Decide API   : ${URL}/v1/decide
  Log          : ${LOG}
  PID file     : ${PID_FILE}

Agent env:
  export LAYA_URL='${URL}'

Try:
  ./bin/laya health --url ${URL}
  ./bin/laya decide --url ${URL} --feature intent_change=0.9 --feature target_known=0.8

Stop:
  kill \$(cat ${PID_FILE})
EOF
