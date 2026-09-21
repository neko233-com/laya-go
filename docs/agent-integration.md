# Agent integration

How external agents (Codex CLI, Claude Code, MiMo Desktop, custom workers) call the Laya System 1 decision service.

Server side: `laya-deploy` listens on **`0.0.0.0:7710`** by default and hosts both the model API and MCP client binaries under `%ProgramData%\Laya` on the deploy host.

```text
  Codex / Claude / MiMo / custom agent
           |  HTTP  /v1/decide
           |  or MCP stdio laya-deploy tools
           v
  laya-deploy @ <laya-host>:7710
```

Use one of three integration styles:

| Style | When | Needs |
| --- | --- | --- |
| **MCP** | Agent runtime supports MCP tools | `laya-mcp` binary + `LAYA_URL` |
| **HTTP** | Script / SDK / simple agent loop | network access to `:7710` |
| **CLI** | Shell tool-calling agents | `laya` binary + `LAYA_URL` |

Replace `<laya-host>` with the deploy machine LAN address reachable by the agent. Local agents on the deploy host can use `127.0.0.1`.

## One-click global install (`laya-mcp`)

Repo scripts install **laya-mcp globally** (binary on user PATH + MCP entries for known agents).

### Windows

```powershell
# local server default http://127.0.0.1:7710
.\deploy\install-mcp.ps1

# teammates pointing at LAN deploy host
.\deploy\install-mcp.ps1 -Url http://<laya-host>:7710
```

What it does:

1. `go build ./mcp` (or reuse existing binary)
2. Copy to `%LOCALAPPDATA%\Laya\bin\laya-mcp.exe` (+ `laya-mcp.cmd` shim)
3. Append install dir to **user PATH**
4. Merge MCP name `laya` into:
   - MiMo Desktop `%USERPROFILE%\.config\mimocode\mimocode.jsonc`
   - Codex `%USERPROFILE%\.codex\config.toml`
   - Claude Desktop / Claude Code config when present
5. Stdio smoke test against `LAYA_URL`

Uninstall binary/PATH:

```powershell
.\deploy\uninstall-mcp.ps1
.\deploy\uninstall-mcp.ps1 -RemoveAgentEntries
```

### Linux / macOS

```sh
./deploy/install-mcp.sh
./deploy/install-mcp.sh --url http://<laya-host>:7710
```

Installs to `~/.local/bin/laya-mcp` and merges the same agent configs when files exist.

**Restart** Codex / MiMo / Claude after install so the MCP server loads.

Examples under [../deploy/examples/](../deploy/examples/) if you prefer manual edits.

## Tool / API mapping

| Capability | MCP tool | HTTP | CLI |
| --- | --- | --- | --- |
| Health | `laya_health` | `GET /health` | `laya health` |
| List models | `laya_models` | `GET /v1/models` | `laya models` |
| Decide | `laya_decide` | `POST /v1/decide` | `laya decide` |
| JEV-compat decide | `laya_decide` + `jev_compat:true` | `POST /v1/jev/decide` | `laya jev-decide` |
| Full scores | — | `POST /v1/predict` | `laya predict` |

Decide request body (HTTP / MCP arguments):

```json
{
  "model": "laya-base",
  "features": {
    "intent_change": 0.9,
    "target_known": 0.8,
    "has_plan": 0.6
  },
  "top_k": 3
}
```

Response top field is the recommended pattern id (`tool.edit`, `tool.ask`, `agent.finish`, ...). Laya does not generate text.

## Codex CLI (OpenAI)

**Recommended:** run [`../deploy/install-mcp.ps1`](../deploy/install-mcp.ps1) (Windows) or [`../deploy/install-mcp.sh`](../deploy/install-mcp.sh) — it writes the Codex MCP block automatically.

Codex loads MCP servers from `%USERPROFILE%\.codex\config.toml` (Windows) or `~/.codex/config.toml`.

Native Windows agent on a developer laptop:

```toml
[mcp_servers.laya]
command = "C:\\ProgramData\\Laya\\laya-mcp.exe"
env = { LAYA_URL = "http://127.0.0.1:7710" }
```

Agent machine on the same LAN as the deploy host (binary copied locally, server remote):

```toml
[mcp_servers.laya]
command = "C:\\tools\\laya-mcp.exe"
env = { LAYA_URL = "http://<laya-host>:7710" }
```

WSL2 Codex install (Linux Codex, Windows-hosted service):

```toml
[mcp_servers.laya]
command = "/mnt/c/ProgramData/Laya/laya-mcp.exe"
env = { LAYA_URL = "http://<laya-host>:7710" }
```

If WSL cannot exec the Windows `.exe`, copy a Linux `laya-mcp` build onto the WSL filesystem and point `command` at that path.

Headless Codex can also call Laya without MCP by instructing it to use shell/HTTP. Example autonomous prompt fragment:

```text
When you need a structured next-step decision, call Laya:
curl -s http://<laya-host>:7710/v1/decide \
  -H 'content-type: application/json' \
  -d '{"model":"laya-base","features":{"intent_change":0.9,"target_known":0.8},"top_k":3}'
Treat response top.id as the preferred pattern. Do not invent tool names outside the returned candidates.
```

Restart Codex after editing `config.toml`. Verify with `codex mcp list` or the Codex MCP status UI if available in your version.

## Claude Code

Claude Code MCP settings (user or project):

```json
{
  "mcpServers": {
    "laya": {
      "command": "C:\\ProgramData\\Laya\\laya-mcp.exe",
      "env": {
        "LAYA_URL": "http://127.0.0.1:7710"
      }
    }
  }
}
```

Project-level example file: [../deploy/examples/mcp.claude-code.json.sample](../deploy/examples/mcp.claude-code.json.sample).

## MiMo Desktop

Config file: `C:\Users\<you>\.config\mimocode\mimocode.jsonc`, top-level `mcp` section:

```jsonc
"laya": {
  "type": "local",
  "command": ["C:\\ProgramData\\Laya\\laya-mcp.exe"],
  "environment": { "LAYA_URL": "http://127.0.0.1:7710" },
  "enabled": true
}
```

Sample: [../deploy/examples/mcp.mimocode.jsonc.sample](../deploy/examples/mcp.mimocode.jsonc.sample). Restart the engine or open a new conversation so tools load.

## Any agent: HTTP

No extra binary required on the agent host.

```sh
curl -s http://<laya-host>:7710/health
curl -s http://<laya-host>:7710/v1/models
curl -s http://<laya-host>:7710/v1/decide \
  -H 'content-type: application/json' \
  -d '{"features":{"blocked":1,"ambiguity":0.8},"top_k":2}'
```

Python sketch:

```python
import json, urllib.request

url = "http://<laya-host>:7710/v1/decide"
body = json.dumps({
    "model": "laya-base",
    "features": {"intent_change": 0.9, "target_known": 0.8},
    "top_k": 3,
}).encode()
req = urllib.request.Request(url, data=body, headers={"content-type": "application/json"})
with urllib.request.urlopen(req, timeout=5) as resp:
    data = json.load(resp)
print(data["top"]["id"], data["top"]["probability"])
```

## Any agent: CLI

Copy `laya` / `laya.exe` to the agent host (or call the copy under `ProgramData\Laya` on the deploy host).

```sh
export LAYA_URL="http://<laya-host>:7710"
laya health
laya models
laya decide --model laya-base \
  --feature intent_change=0.9 \
  --feature target_known=0.8 \
  --top-k 3 --json
# legacy JEV-shaped call during migration
laya jev-decide --feature blocked=1 --json
```

## Agent-side prompt contract

Tell the calling agent:

1. Convert task state into **structured features** (see [api.md](api.md) feature keys). Do not send free-form novels to the server.
2. Call decide **before** picking a tool when unsure.
3. Use returned `top.id` / `candidates[].id` as the action vocabulary.
4. Respect `guard.block` if it appears — stop and escalate; Laya is advisory only.
5. `LAYA_URL` / host:port is an ops value; do not hardcode secrets (there are none today).

Feature keys to start with: `needs_context`, `uncertainty`, `scope_unknown`, `intent_change`, `target_known`, `has_plan`, `needs_verify`, `blocked`, `ambiguity`, `risk_high`, `unsafe`, `complexity`, `parallelizable`, `repetitive`, `external_info`, `has_error`, `goal_done`.

## Ops endpoints for agents / admins

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/admin` | Web dashboard |
| GET | `/admin/api/overview` | Models + metrics + update snapshot |
| POST | `/admin/api/reload` | Hot-reload config + `models.json` overlay |
| GET | `/deploy/update/check` | GitHub release check |
| POST | `/deploy/config` | Toggle auto-update / write config |

## Security

- Default bind is `0.0.0.0:7710` with **no authentication**.
- Only expose to trusted LAN / VPN. Firewall the port otherwise.
- Agents on the LAN can decide **and** call `/deploy/config` / reload. Treat the host as admin-trusted.
- Migration from JEV: [jev-migration.md](jev-migration.md).

## Troubleshooting

| Symptom | Check |
| --- | --- |
| MCP tools missing in agent | Restart agent / new session; binary path exists; `LAYA_URL` set |
| HTTP connection refused | Service running (`Get-Service LayaDeploy`); firewall TCP 7710; correct host IP |
| decide returns error | Feature keys lowercase; model id in `laya-mini/base/pro` |
| jev path only in old clients | Use `/v1/jev/decide` or `laya jev-decide` until callers migrate |
| Update check 403 | GitHub anonymous rate limit; wait or run release check from a logged-in `gh` host |
