---
name: laya-cli
description: Use when changing the laya agent-side CLI (cli/main.go). Covers flags, JEV transition commands, JSON output, and verification against laya-server.
whenToUse: Any request that modifies `laya` CLI commands, flags, env defaults, or printed decision output.
user-invocable: true
---

# laya-cli

This skill is guidance, not a replacement for repository rules. Read [AGENTS.md](../../../AGENTS.md) first.

## Scope

1. CLI is an adapter. HTTP contract changes belong in `internal/apitypes` + `internal/httpserver`, not in CLI-local structs.
2. Shared HTTP calls live in `internal/client`. Reuse that package; do not duplicate request code in `cli/main.go`.
3. `laya jev-decide` is the transition command. Native path is `laya decide`.

## Implement

- Flags: `--url`, `--model`, `--top-k`, `--feature key=value`, `--candidate id`, `--jev`, `--json`.
- Env default `LAYA_URL` (fallback `http://127.0.0.1:7710`).
- Human output stays one-line summary + ranked candidates; `--json` prints full wire objects.
- Exit codes: `0` success, `1` runtime/HTTP failure, `2` usage error.

## Verify

```sh
go build -o bin/laya ./cli
./bin/laya-server -addr 127.0.0.1:7710
./bin/laya health
./bin/laya models
./bin/laya decide --feature goal_done=1 --top-k 2 --json
./bin/laya jev-decide --feature blocked=1 --json
```

Update [docs/api.md](../../../docs/api.md) CLI mapping table if commands change. Migration notes go in [docs/jev-migration.md](../../../docs/jev-migration.md).

## Related skills

- [laya-server](../laya-server/SKILL.md)
- [laya-mcp](../laya-mcp/SKILL.md)
