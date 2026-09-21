# laya-mcp

MCP stdio server exposing Laya System 1 decisions to AI agents. Drop-in replacement for legacy JEV MCP tools.

## What it does

Tools:

- `laya_health`
- `laya_models`
- `laya_decide` (optional `jev_compat: true`)

## Run

```sh
go build -o bin/laya-mcp ./mcp
# stdio: launched by the agent host
LAYA_URL=http://127.0.0.1:7710 ./bin/laya-mcp
```

Example host config shape (no secrets):

```json
{
  "mcpServers": {
    "laya": {
      "command": "bin/laya-mcp",
      "env": { "LAYA_URL": "http://127.0.0.1:7710" }
    }
  }
}
```

## Config

| Variable | Default |
| --- | --- |
| `LAYA_URL` | `http://127.0.0.1:7710` |

## Known limitations

- stdio only in v0.1 (no SSE/HTTP MCP transport yet).
- Requires `laya-server` on `LAYA_URL`.

Docs: [../docs/jev-migration.md](../docs/jev-migration.md), [../docs/api.md](../docs/api.md). Skill: [../.mimocode/skills/laya-mcp/SKILL.md](../.mimocode/skills/laya-mcp/SKILL.md).
