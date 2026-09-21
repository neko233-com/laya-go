---
name: laya-deploy
description: Use when changing laya-go deployment (deploy/main.go, deploy.ps1, deploy.sh, local port 7400, one-click install). Covers build, process lifecycle, health checks, and ops status.
whenToUse: Any request that modifies deploy scripts, the Go deploy server, default port, or local deployment procedure for Laya.
user-invocable: true
---

# laya-deploy

This skill is guidance, not a replacement for repository rules. Read [AGENTS.md](../../../AGENTS.md) and [docs/deploy.md](../../../docs/deploy.md) first.

## Scope

1. `deploy/main.go` embeds `internal/httpserver` handlers; model math stays in `internal/engine`.
2. Deploy scripts only build, start, stop, and verify. They must not invent a second scoring path.
3. Default local port is **7400**. Scripts accept override; docs and smoke commands should stay consistent.

## Implement

- Keep `/v1/*` behavior identical to `laya-server`. Deploy-only extras live under `/deploy/status`.
- Windows script must set `CREATE_NO_WINDOW` for child processes and not leave console hosts.
- Unix script writes `bin/laya-deploy.pid` and `bin/laya-deploy.log`; stop uses the pid file.
- No credentials in scripts or status payloads.

## Verify

```powershell
.\deploy\deploy.ps1 -Port 7400
curl.exe -s http://127.0.0.1:7400/health
curl.exe -s http://127.0.0.1:7400/deploy/status
.\bin\laya.exe health --url http://127.0.0.1:7400
.\bin\laya.exe decide --url http://127.0.0.1:7400 --feature intent_change=0.9 --feature target_known=0.8
```

```sh
./deploy/deploy.sh --port 7400
curl -s http://127.0.0.1:7400/deploy/status
```

Update [docs/deploy.md](../../../docs/deploy.md) when flags, ports, or stop procedure change.

## Related skills

- [laya-server](../laya-server/SKILL.md) — model HTTP contract
- [laya-cli](../laya-cli/SKILL.md) — client against deploy port
- [laya-mcp](../laya-mcp/SKILL.md) — set `LAYA_URL` to the deploy port
