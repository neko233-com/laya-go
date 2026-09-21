---
name: laya-deploy
description: Use when changing laya-go deployment (deploy/*.go/ps1/sh), Windows service LayaDeploy, port 7400, config, or GitHub auto-update. Covers build, service lifecycle, update check/apply, and ops status.
whenToUse: Any request that modifies deploy server code, install/uninstall scripts, service registration, auto-update, or local deployment procedure for Laya.
user-invocable: true
---

# laya-deploy

This skill is guidance, not a replacement for repository rules. Read [AGENTS.md](../../../AGENTS.md) and [docs/deploy.md](../../../docs/deploy.md) first.

## Scope

1. `deploy/main.go` embeds `internal/httpserver` handlers; model math stays in `internal/engine`.
2. Version is owned by [internal/version/version.go](../../../internal/version/version.go). Auto-update compares GitHub tags to this value.
3. Windows service name is `LayaDeploy`. Install layout is `%ProgramData%\Laya` with `config.json`.
4. Auto-update is toggled in config (`auto_update.enabled` / `auto_apply`) and via `POST /deploy/config`.

## Implement

- Keep `/v1/*` behavior identical to `laya-server`. Ops-only routes live under `/deploy/*`.
- Service mode: `-service` or SCM detection; handle stop/shutdown via `golang.org/x/sys/windows/svc`.
- Update apply: download `laya-deploy_<goos>_<goarch>.exe` (plus laya / laya-mcp siblings), stage as `.new`, rename running image to `.old`, restart service.
- Windows install scripts must require elevation, stop stray `laya-deploy` on the port, and write config before `sc create`.
- No credentials in scripts, config samples, or status payloads.

## Verify

```powershell
go test ./...
go build -o bin\laya-deploy.exe .\deploy
.\deploy\install-service.ps1 -Port 7400
Get-Service LayaDeploy
curl.exe -s http://127.0.0.1:7400/deploy/status
curl.exe -s http://127.0.0.1:7400/deploy/update/check
Invoke-RestMethod -Method Post http://127.0.0.1:7400/deploy/config -ContentType application/json -Body '{"auto_update":{"enabled":false,"auto_apply":false,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
```

```sh
./deploy/deploy.sh --port 7400
```

Update [docs/deploy.md](../../../docs/deploy.md) when service names, ports, config fields, or update asset names change.

## Related skills

- [laya-server](../laya-server/SKILL.md) — model HTTP contract
- [laya-cli](../laya-cli/SKILL.md) — client against deploy port
- [laya-mcp](../laya-mcp/SKILL.md) — set `LAYA_URL` to the deploy port
