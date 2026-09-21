# Deploy

One-click local deployment for Laya: Go deploy server (`laya-deploy`) hosts the model API.

Default port: **7400**. Version lives in [`internal/version/version.go`](../internal/version/version.go).

## One-click scripts

| Script | Platform |
| --- | --- |
| [`deploy.ps1`](../deploy/deploy.ps1) | Windows PowerShell (foreground/user process) |
| [`deploy.sh`](../deploy/deploy.sh) | Linux / macOS |

```powershell
.\deploy\deploy.ps1 -Port 7400
```

```sh
./deploy/deploy.sh --port 7400
```

## Windows service

Install as a Windows service (Administrator required):

```powershell
.\deploy\install-service.ps1 -Port 7400
```

Defaults:

| Item | Value |
| --- | --- |
| Service name | `LayaDeploy` |
| Display name | Laya Deploy Server |
| Install dir | `%ProgramData%\Laya` |
| Config | `%ProgramData%\Laya\config.json` |
| Listen | `127.0.0.1:7400` |
| Startup | Automatic |

Uninstall:

```powershell
.\deploy\uninstall-service.ps1
.\deploy\uninstall-service.ps1 -RemoveFiles
```

## Config file

`%ProgramData%\Laya\config.json` (or `LAYA_CONFIG`, or `exeDir/laya-config.json`):

```json
{
  "addr": "127.0.0.1:7400",
  "auto_update": {
    "enabled": true,
    "auto_apply": true,
    "interval_minutes": 360,
    "repo": "neko233-com/laya-go",
    "prerelease": false
  }
}
```

| Field | Meaning |
| --- | --- |
| `auto_update.enabled` | Master switch for GitHub release checks |
| `auto_update.auto_apply` | Download + stage binaries + restart service when newer release exists |
| `auto_update.interval_minutes` | Check cadence (also runs once at startup when enabled) |
| `auto_update.repo` | GitHub `owner/name` |
| `auto_update.prerelease` | Include prerelease tags |

## Auto-update API

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/deploy/update/check` | Query GitHub for newer release |
| GET | `/deploy/update/status` | Last check snapshot |
| POST | `/deploy/update/apply` | Download/stage assets; `?force=1` ignores version compare |
| GET | `/deploy/config` | Read current config |
| POST | `/deploy/config` | Patch config; writes file; auto_update toggles apply immediately |

Toggle auto-update at runtime:

```powershell
Invoke-RestMethod -Method Post http://127.0.0.1:7400/deploy/config `
  -ContentType application/json `
  -Body '{"auto_update":{"enabled":false,"auto_apply":false,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
```

### Update pipeline

1. `enabled=true` → periodic GitHub releases query.
2. Newer tag than `internal/version` → `update_available=true`.
3. `auto_apply=true` → download matching assets (`laya-deploy_windows_amd64.exe`, …) next to the service exe as `.new`.
4. Swap running binary name to `.old`, place `.new` as current, restart `LayaDeploy` service.
5. Next process start runs the new image.

If the GitHub release has no platform assets, check reports the tag but apply will not replace binaries. Publish assets with:

```powershell
.\deploy\publish-release.ps1 -Tag v0.2.0
```

## Verify service

```powershell
Get-Service LayaDeploy
curl.exe -s http://127.0.0.1:7400/health
curl.exe -s http://127.0.0.1:7400/deploy/status
curl.exe -s http://127.0.0.1:7400/deploy/update/check
```

CLI:

```powershell
$env:LAYA_URL = 'http://127.0.0.1:7400'
& "$env:ProgramData\Laya\laya.exe" health
& "$env:ProgramData\Laya\laya.exe" decide --feature intent_change=0.9 --feature target_known=0.8
```

## Environment

```text
LAYA_URL=http://127.0.0.1:7400
LAYA_CONFIG=C:\ProgramData\Laya\config.json
```

## Notes

- Service install requires elevation.
- `addr` changes in config need service restart; `auto_update` toggles apply without restart.
- Deploy API has no auth; bind to loopback unless a gateway is in front.
- Ad-hoc `deploy.ps1` process and the Windows service must not fight on port 7400; install script stops stray processes first.

Related: [api.md](api.md), [development.md](development.md), [architecture.md](architecture.md).
