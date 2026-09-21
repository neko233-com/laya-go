# Laya Go

English | [中文](README.zh.md)

Open horizontal **System 1** decision models for agents. Laya does **not** generate text. It scores structured features once and returns ranked pattern probabilities in microseconds to milliseconds. This repo is the Go **server + agent integration** that replaces legacy **JEV**.

**How to use:** run `laya-deploy` (default `0.0.0.0:7710`). Agents call `decide` via MCP, HTTP, or CLI and take `top.id` as the action.

---

## 30-second start

**Deploy host (Administrator PowerShell):**

```powershell
git clone https://github.com/neko233-com/laya-go.git
cd laya-go
.\deploy\install-service.ps1 -Port 7710
```

Service `LayaDeploy` listens on `0.0.0.0:7710` (model API + admin UI in one process).

**Local or teammate machines (agent access):**

```powershell
.\deploy\install-mcp.ps1 -Url http://127.0.0.1:7710
# teammates: use the deploy host IP, e.g. http://192.168.x.x:7710
```

**Restart** Codex / MiMo / Claude. MCP tool name: `laya`.

**Verify:**

```sh
curl -s http://127.0.0.1:7710/health
curl -s http://127.0.0.1:7710/v1/decide -H 'content-type: application/json' \
  -d '{"features":{"intent_change":0.9,"target_known":0.8},"top_k":3}'
```

Admin dashboard: `http://127.0.0.1:7710/admin` (use the deploy host IP on LAN).

---

## Pick your path

| Role | Goal | Start here |
| --- | --- | --- |
| Ops / deploy | Install service, expose LAN, monitor, hot-reload | Install + Admin below |
| Teammate | Install MCP or use HTTP from your agent | Agent usage below |
| Agent builder | Encode state as features, consume `top.id` | Make it pleasant below |
| Coming from JEV | Compat API + action map | [`docs/jev-migration.md`](docs/jev-migration.md) |

---

## Install

### Server (deploy host)

| Mode | Command | Notes |
| --- | --- | --- |
| Windows service (recommended) | `.\deploy\install-service.ps1 -Port 7710` | Admin; autostart; `%ProgramData%\Laya` |
| One-click foreground | `.\deploy\deploy.ps1 -Port 7710` | No service; current session |
| Linux / macOS | `./deploy/deploy.sh --port 7710` | Writes `bin/laya-deploy.pid` + log |

From source:

```sh
go test ./...
go build -o bin/laya-deploy ./deploy
go build -o bin/laya ./cli
go build -o bin/laya-mcp ./mcp
./bin/laya-deploy -addr 0.0.0.0:7710
```

### Client / agents (global MCP)

```powershell
.\deploy\install-mcp.ps1 -Url http://<laya-host>:7710
```

```sh
./deploy/install-mcp.sh --url http://<laya-host>:7710
```

Installs `laya-mcp` on user PATH and merges MCP config for MiMo Desktop / Codex / Claude.  
Uninstall: `.\deploy\uninstall-mcp.ps1` (`-RemoveAgentEntries` also removes agent entries).

### Dev without service

```powershell
go build -o bin\laya-server.exe .\server
.\bin\laya-server.exe -addr 127.0.0.1:7710
$env:LAYA_URL = "http://127.0.0.1:7710"
.\bin\laya.exe decide --feature goal_done=1 --json
```

---

## Agent usage

Three styles. Config samples: [`docs/agent-integration.md`](docs/agent-integration.md), [`deploy/examples/`](deploy/examples/).

### 1. MCP (recommended for Codex / Claude / MiMo)

| Tool | Purpose |
| --- | --- |
| `laya_health` | Liveness |
| `laya_models` | Models and pattern ids |
| `laya_decide` | Structured decision; `jev_compat: true` for legacy semantics |

Example arguments:

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

### 2. HTTP (any language / any agent)

```sh
curl -s http://<laya-host>:7710/v1/decide \
  -H 'content-type: application/json' \
  -d '{"model":"laya-base","features":{"blocked":1,"ambiguity":0.8},"top_k":2}'
```

Read `top.id` and `candidates[]`. If `top` is far ahead, use it. If candidates are close, pick by policy or re-ask with sharper features.

### 3. CLI (scripts / shell agents)

```powershell
$env:LAYA_URL = "http://<laya-host>:7710"
laya health
laya models
laya decide --model laya-base --feature intent_change=0.9 --feature target_known=0.8 --top-k 3 --json
laya jev-decide --feature blocked=1 --json
```

---

## Make it pleasant

### Feed features, not essays

The server does **not** parse natural language. Map state to 0~1 lowercase keys:

| Feature | Meaning |
| --- | --- |
| `needs_context` | Missing context |
| `uncertainty` | Unsure next step |
| `scope_unknown` | Blast radius unclear |
| `intent_change` | Wants code/behavior change |
| `target_known` | File/command already known |
| `has_plan` | Plan exists |
| `needs_verify` | Needs run/test evidence |
| `blocked` | Stuck on input |
| `ambiguity` | Underspecified |
| `risk_high` / `unsafe` | Dangerous territory |
| `complexity` | Multi-step / hard |
| `parallelizable` | Work can split |
| `repetitive` | Same op many times |
| `external_info` | Needs outside facts |
| `has_error` | Recent failure |
| `goal_done` | Goal looks complete |

Unknown keys are ignored. Full table: [`docs/api.md`](docs/api.md).

### Ready-made recipes

| Scenario | Features | Typical top |
| --- | --- | --- |
| Know target + plan | `intent_change=0.9, target_known=0.8, has_plan=0.6` | `tool.edit` |
| Don't know where code is | `uncertainty=0.8, needs_context=0.7, scope_unknown=0.6` | `tool.search` / `tool.read` |
| Need human input | `blocked=0.9, ambiguity=0.8` | `tool.ask` |
| Change done, verify | `needs_verify=0.9, has_plan=0.5` | `tool.run` |
| Work finished | `goal_done=0.9` | `agent.finish` |
| Clearly unsafe | `unsafe=0.9, risk_high=0.8` | `guard.block` |
| Big messy task | `complexity=0.8, ambiguity=0.5, scope_unknown=0.5` | `agent.plan` |
| Recover after failure | `has_error=0.9, needs_verify=0.4` | `agent.retry` |

Debug scoring:

```sh
curl -s http://127.0.0.1:7710/v1/explain -H 'content-type: application/json' \
  -d '{"pattern_id":"tool.edit","features":{"intent_change":1,"target_known":1}}'
```

### Pick the model

| Model | Use when |
| --- | --- |
| `laya-mini` | High-frequency coarse routing |
| `laya-base` (default) | Everyday agent tool routing |
| `laya-pro` | Complex multi-step plans |

### Pin an agent policy snippet

Add to Codex / Claude / MiMo project instructions:

```text
When you need a next-step decision, call Laya (MCP laya_decide or HTTP /v1/decide):
- Encode current state as features (0~1 lowercase keys); do not send essays
- Use top.id / candidates[].id as the action vocabulary
- If guard.block appears: stop and escalate; do not force the action
- Laya only advises; tool execution remains your decision
```

### Admin dashboard

`http://<laya-host>:7710/admin` — health, versions, models, decide metrics, recent decisions, hot reload, auto-update toggle.  
API: `GET /admin/api/overview`, `POST /admin/api/reload`.

### Hot-reload models without restart

Drop overlay next to service config:

```text
%ProgramData%\Laya\models.json
```

Sample: [`deploy/examples/models.overlay.sample.json`](deploy/examples/models.overlay.sample.json)  
Then click reload in `/admin` or `POST /admin/api/reload`.

### Auto-update (toggleable)

On by default: checks GitHub Releases, downloads, restarts service.

```powershell
# off
Invoke-RestMethod -Method Post http://127.0.0.1:7710/deploy/config -ContentType application/json `
  -Body '{"auto_update":{"enabled":false,"auto_apply":false,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
# on
Invoke-RestMethod -Method Post http://127.0.0.1:7710/deploy/config -ContentType application/json `
  -Body '{"auto_update":{"enabled":true,"auto_apply":true,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
```

Publish for the fleet:

```powershell
.\deploy\publish-release.ps1 -Tag v0.3.0
```

### Replace JEV

```sh
laya jev-decide --feature blocked=1 --json
laya decide --model laya-base --feature blocked=1 --feature ambiguity=0.5
```

MCP rename: `jev decide` → `laya_decide` (optional `jev_compat: true`).  
Details: [`docs/jev-migration.md`](docs/jev-migration.md).

### Smoke checklist

1. `GET /health` returns `ok`
2. `/admin` opens and metrics move
3. Agent lists MCP tool `laya_decide`
4. Real features produce a sensible `top.id`
5. Teammate can `curl http://<deploy-host-ip>:7710/health`

---

## Config

| Variable / file | Component | Default |
| --- | --- | --- |
| `LAYA_ADDR` / `-addr` | laya-deploy / laya-server | `0.0.0.0:7710` |
| `LAYA_URL` | cli / mcp | `http://127.0.0.1:7710` |
| `LAYA_CONFIG` | laya-deploy | `%ProgramData%\Laya\config.json` |
| `%ProgramData%\Laya\models.json` | model overlay | built-ins if absent |

No API key. **No auth**: `0.0.0.0:7710` is for trusted LAN only; firewall or gateway otherwise.

---

## Components

| Path | Binary | Role |
| --- | --- | --- |
| [`deploy/`](deploy/) | `laya-deploy` | Deploy server + model API + `/admin` + scripts |
| [`server/`](server/) | `laya-server` | Slim HTTP decision process |
| [`cli/`](cli/) | `laya` | CLI client |
| [`mcp/`](mcp/) | `laya-mcp` | stdio MCP for AI agents |

Models: `laya-mini` / `laya-base` / `laya-pro`.  
Stable patterns: `tool.search|read|edit|run|ask`, `agent.plan|delegate|retry|finish`, `agent.research|reflect`, `tool.batch`, `guard.block`.

---

## Docs and skills

| Doc | Topic |
| --- | --- |
| [`docs/agent-integration.md`](docs/agent-integration.md) | Codex / Claude / MiMo / HTTP / CLI |
| [`docs/deploy.md`](docs/deploy.md) | Deploy, service, port, stop/verify |
| [`docs/api.md`](docs/api.md) | HTTP contract + feature keys |
| [`docs/architecture.md`](docs/architecture.md) | System 1 architecture |
| [`docs/jev-migration.md`](docs/jev-migration.md) | JEV migration |
| [`docs/development.md`](docs/development.md) | Build and test |

Repo agent skills: `.mimocode/skills/laya-*`. Rules: [`AGENTS.md`](AGENTS.md).

---

## Known limitations

- Built-in models are rule/weight patterns, not neural checkpoints; trained weights can plug in later.
- Advisory only; `guard.block` is not a policy engine.
- Callers must turn language into features.
- No auth; LAN exposure is full trust.
- Anonymous GitHub update checks can hit rate limits (403).

## License

MIT — see [LICENSE](LICENSE).

## Contributors

- [neko233-com](https://github.com/neko233-com)
