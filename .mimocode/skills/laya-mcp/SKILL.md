---
name: laya-mcp
description: Use when changing the laya MCP server or agent integration that replaces JEV tools (mcp/main.go, host config, tool schemas).
whenToUse: Any request that modifies MCP tools, stdio JSON-RPC handling, JEV tool renames, or agent-side Laya integration docs.
user-invocable: true
---

# laya-mcp

This skill is guidance, not a replacement for repository rules. Read [AGENTS.md](../../../AGENTS.md) and [docs/jev-migration.md](../../../docs/jev-migration.md) first.

## Scope

1. `mcp/` is the agent-facing surface. Decision math stays on `laya-server`; MCP only adapts tools.
2. Tool names are a public agent contract: `laya_health`, `laya_models`, `laya_decide`. Renames require a migration doc update.
3. Legacy JEV parity is `laya_decide` + `jev_compat: true`, not a separate `jev_decide` tool.

## Implement

- Transport in v0.1 is stdio JSON-RPC. Handle `initialize`, `tools/list`, `tools/call`, `ping`.
- Use `internal/client` for HTTP calls; set base URL from `LAYA_URL`.
- Tool results are MCP content blocks (`type: text`) wrapping JSON. Set `isError: true` on failures.
- Do not log secrets; this repo never needs credentials.

## Verify

```sh
go build -o bin/laya-mcp ./mcp
./bin/laya-server -addr 127.0.0.1:7710

# tools/list + decide via stdio
$payload = @'
{"jsonrpc":"2.0","id":1,"method":"tools/list"}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"laya_decide","arguments":{"features":{"intent_change":0.9,"target_known":0.8},"top_k":2}}}
'@
$payload | ./bin/laya-mcp
```

Host config examples live in [mcp/README.md](../../../mcp/README.md). Keep the JEV rename map current in [docs/jev-migration.md](../../../docs/jev-migration.md).

## Related skills

- [laya-server](../laya-server/SKILL.md)
- [laya-cli](../laya-cli/SKILL.md)
