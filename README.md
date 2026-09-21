# Laya Go

English | [中文](README.zh.md)

Open horizontal **System 1** decision model family for agents. Non-autoregressive: Laya does not generate text. It scores structured features against a finite pattern set and returns ranked probabilities in microseconds.

This repository is the Go **server + agent-side integration** that replaces the legacy **JEV** decision service.

## What it does

- Hosts the open Laya family: `laya-mini`, `laya-base`, `laya-pro`
- HTTP API for fast structured pattern prediction
- Agent CLI (`laya`) and MCP server (`laya-mcp`) as drop-in JEV replacements
- Stable pattern IDs for agent tool routing (`tool.edit`, `agent.finish`, `guard.block`, ...)

## Components

| Directory | Binary | Role |
| --- | --- | --- |
| [`server/`](server/) | `laya-server` | HTTP decision server |
| [`cli/`](cli/) | `laya` | Agent-side CLI |
| [`mcp/`](mcp/) | `laya-mcp` | MCP tools for AI agents |

Shared engine and client live under `internal/`. Long-term docs: [`docs/`](docs/).

## Install / Run

```sh
go test ./...
go build -o bin/laya-server ./server
go build -o bin/laya ./cli
go build -o bin/laya-mcp ./mcp

./bin/laya-server -addr 127.0.0.1:7710
./bin/laya health
./bin/laya decide --feature intent_change=0.9 --feature target_known=0.8 --top-k 3
```

## Config

| Variable | Component | Default |
| --- | --- | --- |
| `LAYA_ADDR` | server | `127.0.0.1:7710` |
| `LAYA_URL` | cli / mcp | `http://127.0.0.1:7710` |

No API keys are required for local open-source usage.

## Replace JEV

```sh
# transition path
laya jev-decide --feature blocked=1 --json

# target path
laya decide --model laya-base --feature blocked=1 --feature ambiguity=0.5
```

MCP tool rename: `jev decide` → `laya_decide` (optional `jev_compat: true` during cutover).

Full mapping: [`docs/jev-migration.md`](docs/jev-migration.md).

## API quick look

```sh
curl -s http://127.0.0.1:7710/v1/decide \
  -H 'content-type: application/json' \
  -d '{"model":"laya-base","features":{"intent_change":0.9,"target_known":0.8},"top_k":3}'
```

Contract: [`docs/api.md`](docs/api.md). Architecture: [`docs/architecture.md`](docs/architecture.md).

## Agent skills

Component workflows for coding agents:

- [`.mimocode/skills/laya-server`](.mimocode/skills/laya-server/SKILL.md)
- [`.mimocode/skills/laya-cli`](.mimocode/skills/laya-cli/SKILL.md)
- [`.mimocode/skills/laya-mcp`](.mimocode/skills/laya-mcp/SKILL.md)

Repo rules: [`AGENTS.md`](AGENTS.md). Docs rules: [`docs/AGENTS.md`](docs/AGENTS.md).

## Known limitations

- Built-in models are rule/weight patterns, not neural checkpoints. They demonstrate the System 1 serving contract; plug-in trained weights can replace scoring later without changing the API.
- Laya is advisory: it does not execute tools or enforce policy. `guard.block` is a signal only.
- Free-text intents must be converted to features by the caller; the server will not parse natural language.

## License

MIT — see [LICENSE](LICENSE).
