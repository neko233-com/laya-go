---
name: laya-server
description: Use when changing the laya-go HTTP decision server (server/, internal/httpserver, engine model wiring exposed over HTTP). Establishes routes, contract, and verification for laya-server.
whenToUse: Any request that modifies server routes, request/response JSON, listen address behavior, or the HTTP surface of Laya.
user-invocable: true
---

# laya-server

This skill is guidance, not a replacement for repository rules. Read [AGENTS.md](../../../AGENTS.md) and [docs/AGENTS.md](../../../docs/AGENTS.md) first.

## Scope

1. Confirm worktree status and the owning package: route/handler changes live in `internal/httpserver`; scoring stays in `internal/engine`.
2. Wire types are owned by `internal/apitypes`. Change the contract there first, then server, then CLI/MCP adapters.
3. JEV compatibility is a thin alias on decide (`/v1/jev/decide` + `compat_layer`). Do not fork scoring for JEV.

## Implement

- Keep handlers free of policy side effects. Laya scores; it does not execute tools.
- Preserve non-autoregressive semantics: one feature snapshot, score-all, softmax, rank.
- Default bind is loopback. Do not add silent public binds.
- Feature keys are lower-cased on ingest. Unknown keys contribute zero weight.

## Verify

```sh
go test ./internal/engine ./internal/httpserver
go build -o bin/laya-server ./server
./bin/laya-server -addr 127.0.0.1:7710
./bin/laya health --url http://127.0.0.1:7710
./bin/laya decide --url http://127.0.0.1:7710 --feature intent_change=0.9 --feature target_known=0.8
```

If public contract changed, update [docs/api.md](../../../docs/api.md) in the same change.

## Related skills

- [laya-cli](../laya-cli/SKILL.md) for CLI flags/output
- [laya-mcp](../laya-mcp/SKILL.md) for agent tool surface
