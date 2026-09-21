# AGENTS.md — Laya Go

Laya is the open horizontal System 1 decision model family. This repository is the Go server + agent-side integration that replaces the legacy JEV decision service.

Laya does **not** generate text. Clients send structured features; the engine scores every pattern in one non-autoregressive pass and returns ranked probabilities.

## Repository layout

| Path | Role |
| --- | --- |
| `server/` | HTTP decision server (`laya-server`) |
| `cli/` | Agent-side CLI (`laya`) |
| `mcp/` | MCP server for AI agents (`laya-mcp`) |
| `deploy/` | Go deploy server (`laya-deploy`) + ps1/sh one-click scripts |
| `internal/engine` | System 1 scoring + built-in model family |
| `internal/httpserver` | HTTP routes and JSON contract |
| `internal/client` | Shared HTTP client used by CLI/MCP |
| `internal/apitypes` | Wire types |
| `docs/` | Long-term documentation (see [docs/AGENTS.md](docs/AGENTS.md)) |
| `.mimocode/skills/` | Component skills for agents working in this repo |

## Commands

```sh
go test ./...
go build -o bin/laya-server ./server
go build -o bin/laya ./cli
go build -o bin/laya-mcp ./mcp
go build -o bin/laya-deploy ./deploy
```

One-click local deploy (default port **7710**):

```powershell
.\deploy\deploy.ps1 -Port 7710
```

```sh
./deploy/deploy.sh --port 7710
```

Global MCP install for agents (`laya` tool on PATH + MiMo/Codex/Claude configs):

```powershell
.\deploy\install-mcp.ps1 -Url http://127.0.0.1:7710
```

```sh
./deploy/install-mcp.sh --url http://127.0.0.1:7710
```

Windows service + auto-update (Administrator):

```powershell
.\deploy\install-service.ps1 -Port 7710
# toggle auto-update
Invoke-RestMethod -Method Post http://127.0.0.1:7710/deploy/config -ContentType application/json -Body '{"auto_update":{"enabled":false,"auto_apply":false,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
```

Service details: [docs/deploy.md](docs/deploy.md). Version: `internal/version/version.go`.

Smoke:

```sh
./bin/laya-server -addr 0.0.0.0:7710
./bin/laya health
./bin/laya decide --feature intent_change=0.9 --feature target_known=0.8
```

## Hard rules

- Feature keys are lower-case structured signals; never invent free-text generation paths in the engine.
- JEV replacement lives at `POST /v1/jev/decide` and CLI `laya jev-decide` / MCP `laya_decide` with `jev_compat:true`. New agents use native `/v1/decide`.
- Default listen address is `0.0.0.0:7710` (LAN-facing). Override with `LAYA_ADDR` (server) or `LAYA_URL` (CLI/MCP). There is no auth — restrict with firewall/VPN on untrusted networks.
- Docs follow [docs/AGENTS.md](docs/AGENTS.md): one fact one home; archive holds one-off records.
- No credentials, production IPs, or private keys in this repository.

## Layered agent instructions

| File | When to load |
| --- | --- |
| This file (`AGENTS.md`) | Every session: layout, commands, hard rules |
| [docs/AGENTS.md](docs/AGENTS.md) | Before writing docs |
| [.mimocode/skills/laya-server/SKILL.md](.mimocode/skills/laya-server/SKILL.md) | Changing the HTTP server |
| [.mimocode/skills/laya-cli/SKILL.md](.mimocode/skills/laya-cli/SKILL.md) | Changing the CLI |
| [.mimocode/skills/laya-mcp/SKILL.md](.mimocode/skills/laya-mcp/SKILL.md) | Changing the MCP server / agent integration |
| [.mimocode/skills/laya-deploy/SKILL.md](.mimocode/skills/laya-deploy/SKILL.md) | Changing deploy server or ps1/sh scripts |
| [docs/deploy.md](docs/deploy.md) | One-click deploy, Port 7710, stop/verify |
| [docs/agent-integration.md](docs/agent-integration.md) | Codex / Claude / MiMo / HTTP / CLI agent 接入 |
| [docs/jev-migration.md](docs/jev-migration.md) | Moving callers off JEV |

## Editing instructions

Rules live once in their owning layer. Other layers link here; do not duplicate long procedures.
