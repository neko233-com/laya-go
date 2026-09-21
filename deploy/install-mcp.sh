#!/usr/bin/env bash
# One-click GLOBAL install for laya-mcp on Linux/macOS.
set -euo pipefail

URL="${LAYA_URL:-}"
INSTALL_DIR="${LAYA_MCP_INSTALL_DIR:-$HOME/.local/bin}"
SKIP_BUILD=0
AGENTS="mimocode,codex,claude"

usage() {
  cat <<'EOF'
Usage: deploy/install-mcp.sh [--url http://127.0.0.1:7710] [--install-dir DIR]
                             [--skip-build] [--agents mimocode,codex,claude]

Installs laya-mcp to a user PATH dir and merges MCP config into known agents.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --url) URL="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    --skip-build) SKIP_BUILD=1; shift ;;
    --agents) AGENTS="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown arg: $1" >&2; usage; exit 2 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "$REPO_ROOT"

step() { echo "[laya-mcp-install] $*"; }
ok() { echo "[laya-mcp-install] $*"; }
warn() { echo "[laya-mcp-install] $*" >&2; }
die() { echo "[laya-mcp-install] $*" >&2; exit 1; }

if [[ -z "$URL" ]]; then
  URL="http://127.0.0.1:7710"
fi
URL="${URL%/}"
step "target LAYA_URL=$URL"

SRC=""
if [[ "$SKIP_BUILD" -ne 1 ]] && command -v go >/dev/null 2>&1; then
  step "go build ./mcp"
  mkdir -p "$REPO_ROOT/bin"
  go build -o "$REPO_ROOT/bin/laya-mcp" ./mcp
  SRC="$REPO_ROOT/bin/laya-mcp"
elif [[ -x "$REPO_ROOT/bin/laya-mcp" ]]; then
  SRC="$REPO_ROOT/bin/laya-mcp"
elif command -v laya-mcp >/dev/null 2>&1; then
  SRC="$(command -v laya-mcp)"
else
  die "laya-mcp binary not found; install Go and re-run"
fi

mkdir -p "$INSTALL_DIR"
DEST="$INSTALL_DIR/laya-mcp"
cp -f "$SRC" "$DEST"
chmod +x "$DEST"
ok "installed $DEST"

# PATH hint for current shell + common rc files
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ok "PATH already contains $INSTALL_DIR" ;;
  *)
    warn "add to PATH: export PATH=\"$INSTALL_DIR:\$PATH\""
    for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
      if [[ -f "$rc" ]] && ! grep -q "$INSTALL_DIR" "$rc" 2>/dev/null; then
        echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >>"$rc"
        ok "appended PATH export to $rc"
      fi
    done
    export PATH="$INSTALL_DIR:$PATH"
    ;;
esac

merge_mcp_json() {
  local path="$1"
  local url="$2"
  local cmd="$DEST"
  if [[ ! -f "$path" ]]; then
    warn "skip missing $path"
    return 0
  fi
  python3 - "$path" "$cmd" "$url" <<'PY'
import json, sys
path, cmd, url = sys.argv[1], sys.argv[2], sys.argv[3]
try:
    raw = open(path, "r", encoding="utf-8").read()
    # drop // comments (jsonc)
    lines = []
    for line in raw.splitlines():
        if line.lstrip().startswith("//"):
            continue
        lines.append(line)
    data = json.loads("\n".join(lines) or "{}")
except Exception as e:
    print("parse failed", path, e, file=sys.stderr)
    sys.exit(0)
if not isinstance(data, dict):
    print("skip non-object", path, file=sys.stderr)
    sys.exit(0)
key = "mcpServers" if "mcpServers" in data or "claude" in path.lower() else "mcp"
if key == "mcp" and "mcp" not in data and "mimocode" in path:
    key = "mcp"
if "mcpServers" in data:
    key = "mcpServers"
elif "mcp" in data:
    key = "mcp"
else:
    key = "mcp"
data.setdefault(key, {})
if key == "mcpServers":
    data[key]["laya"] = {"command": cmd, "env": {"LAYA_URL": url}}
else:
    data[key]["laya"] = {
        "type": "local",
        "command": [cmd],
        "environment": {"LAYA_URL": url},
        "enabled": True,
    }
open(path, "w", encoding="utf-8").write(json.dumps(data, indent=2, ensure_ascii=False) + "\n")
print("updated", path)
PY
}

merge_codex_toml() {
  local path="$HOME/.codex/config.toml"
  local url="$1"
  local cmd="$DEST"
  if [[ ! -d "$HOME/.codex" ]]; then
    warn "skip Codex ($HOME/.codex missing)"
    return 0
  fi
  local block
  block=$(cat <<EOF

[mcp_servers.laya]
command = '$cmd'
env = { LAYA_URL = "$url" }
EOF
)
  if [[ ! -f "$path" ]]; then
    printf '%s\n' "$block" >"$path"
    ok "created $path"
    return 0
  fi
  if grep -q '^\[mcp_servers\.laya\]' "$path"; then
    python3 - "$path" "$block" <<'PY'
import re, sys
path, block = sys.argv[1], sys.argv[2].lstrip() + "\n"
raw = open(path, "r", encoding="utf-8").read()
new = re.sub(r"(?ms)^\[mcp_servers\.laya\].*?(?=^\[|\Z)", block, raw, count=1)
open(path, "w", encoding="utf-8").write(new)
print("updated", path)
PY
  else
    printf '\n%s\n' "$block" >>"$path"
    ok "appended $path"
  fi
}

IFS=',' read -ra AGENT_ARR <<<"$AGENTS"
for a in "${AGENT_ARR[@]}"; do
  a="$(echo "$a" | tr '[:upper:]' '[:lower:]' | tr -d ' ')"
  case "$a" in
    mimocode|mimo)
      for p in "$HOME/.config/mimocode/mimocode.jsonc" "$HOME/.config/mimocode/mimocode.json"; do
        merge_mcp_json "$p" "$URL"
      done
      ;;
    codex) merge_codex_toml "$URL" ;;
    claude)
      for p in "$HOME/.claude.json" "$HOME/.config/Claude/claude_desktop_config.json"; do
        merge_mcp_json "$p" "$URL"
      done
      ;;
    *) warn "unknown agent $a" ;;
  esac
done

ok "GLOBAL install done"
echo
echo "  Binary : $DEST"
echo "  LAYA_URL default: $URL"
echo "  MCP name: laya"
echo "  Tools: laya_health / laya_models / laya_decide"
echo
echo "Restart agent hosts to load MCP. Docs: docs/agent-integration.md"
