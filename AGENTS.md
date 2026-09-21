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

One-click local deploy (default port **7400**):

```powershell
.\deploy\deploy.ps1 -Port 7400
```

```sh
./deploy/deploy.sh --port 7400
```

Smoke:

```sh
./bin/laya-server -addr 127.0.0.1:7710
./bin/laya health
./bin/laya decide --feature intent_change=0.9 --feature target_known=0.8
```

## Hard rules

- Feature keys are lower-case structured signals; never invent free-text generation paths in the engine.
- JEV replacement lives at `POST /v1/jev/decide` and CLI `laya jev-decide` / MCP `laya_decide` with `jev_compat:true`. New agents use native `/v1/decide`.
- Default listen address is `127.0.0.1:7710`. Override with `LAYA_ADDR` (server) or `LAYA_URL` (CLI/MCP).
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
| [docs/deploy.md](docs/deploy.md) | One-click deploy, port 7400, stop/verify |
| [docs/jev-migration.md](docs/jev-migration.md) | Moving callers off JEV |

## Editing instructions

Rules live once in their owning layer. Other layers link here; do not duplicate long procedures.
