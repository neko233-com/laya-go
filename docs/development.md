# Development

## Prerequisites

- Go 1.27+
- Windows, macOS, or Linux

## Build

```sh
go test ./...
go build -o bin/laya-server ./server
go build -o bin/laya ./cli
go build -o bin/laya-mcp ./mcp
```

On Windows PowerShell:

```powershell
go test ./...
go build -o bin\laya-server.exe .\server
go build -o bin\laya.exe .\cli
go build -o bin\laya-mcp.exe .\mcp
```

## Run

```sh
./bin/laya-server -addr 127.0.0.1:7710
./bin/laya health --url http://127.0.0.1:7710
./bin/laya decide --feature intent_change=0.9 --feature target_known=0.8 --top-k 3
```

Env:

| Variable | Used by | Default |
| --- | --- | --- |
| `LAYA_ADDR` | server listen address | `0.0.0.0:7710` |
| `LAYA_URL` | cli / mcp base URL | `http://127.0.0.1:7710` (local) or `http://<host-ip>:7710` (LAN) |

## Component skills

Before changing a component, load the matching skill under `.mimocode/skills/`:

- `laya-server` — HTTP server
- `laya-cli` — CLI
- `laya-mcp` — MCP / agent integration

Root rules stay in [../AGENTS.md](../AGENTS.md). Doc rules stay in [AGENTS.md](AGENTS.md).

## Test discipline

- Engine behavior lives in `internal/engine` tests.
- HTTP contract lives in `internal/httpserver` tests.
- CLI/MCP are thin; change contract in apitypes + server first, then adapters.
- Run `go test ./...` before claiming a change complete.

## Docs updates

Public contract changes (routes, pattern IDs, env vars, tool names) must update [api.md](api.md) and, when relevant, [jev-migration.md](jev-migration.md) in the same change.
