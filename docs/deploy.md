# Deploy

One-click local deployment for Laya: Go deploy server (`laya-deploy`) hosts the model API.

## Scripts

| Script | Platform |
| --- | --- |
| [`deploy.ps1`](../deploy/deploy.ps1) | Windows PowerShell |
| [`deploy.sh`](../deploy/deploy.sh) | Linux / macOS |

Default port: **7400**.

## Windows

```powershell
# from repo root
.\deploy\deploy.ps1 -Port 7400
```

What it does:

1. `go test ./...`
2. Build `bin/laya-deploy.exe`, `bin/laya.exe`, `bin/laya-mcp.exe`
3. If `/health` + `/deploy/status` are already ok on the port, reuse that process
4. Otherwise stop any previous laya-deploy on the same port
5. Start `laya-deploy -port 7400` (Windows: no console window)
6. Wait until `GET /health` and `GET /deploy/status` return ok
7. Print client hints (`LAYA_URL`)

Use `-SkipTests` to skip step 1. Use `-Foreground` to run in the current console.

## Linux / macOS

```sh
./deploy/deploy.sh --port 7400
```

Same steps as PowerShell; writes `bin/laya-deploy.pid` and `bin/laya-deploy.log`.

## Verify

```sh
curl -s http://127.0.0.1:7400/health
curl -s http://127.0.0.1:7400/deploy/status
curl -s http://127.0.0.1:7400/v1/decide \
  -H 'content-type: application/json' \
  -d '{"model":"laya-base","features":{"intent_change":0.9,"target_known":0.8},"top_k":3}'
```

CLI against the deploy port:

```sh
./bin/laya health --url http://127.0.0.1:7400
./bin/laya decide --url http://127.0.0.1:7400 --feature blocked=1
```

## Stop

Windows script does not leave a console window; stop via:

```powershell
Get-Process laya-deploy -ErrorAction SilentlyContinue | Stop-Process -Force
```

Unix:

```sh
kill "$(cat bin/laya-deploy.pid)" 2>/dev/null || true
```

## Environment for agents

```text
LAYA_URL=http://127.0.0.1:7400
```

MCP host config should set the same value on `laya-mcp`.

## Layout

| Path | Role |
| --- | --- |
| `deploy/main.go` | Go deploy server + embedded Laya model API |
| `deploy/deploy.ps1` | Windows one-click |
| `deploy/deploy.sh` | Unix one-click |
| `bin/laya-deploy` | Built deploy binary |

Related: [api.md](api.md), [development.md](development.md).
