# laya-deploy

Go deployment server that hosts the Laya System 1 model service and exposes deployment status.

## What it does

- Serves the full Laya model API on one process (`/health`, `/v1/*`)
- Adds `GET /deploy/status` for ops (pid, uptime, binary path, model count)
- Default listen: `127.0.0.1:7400`

## Run

```sh
go build -o bin/laya-deploy ./deploy
./bin/laya-deploy -port 7400
# or
./bin/laya-deploy -addr 127.0.0.1:7400
```

One-click from repo root:

```powershell
.\deploy\deploy.ps1 -Port 7400
```

```sh
./deploy/deploy.sh --port 7400
```

## Config

| Flag | Default | Meaning |
| --- | --- | --- |
| `-port` | `7400` | Listen port |
| `-addr` | empty | Full address; overrides `-port` |

Env for clients: `LAYA_URL=http://127.0.0.1:7400`

## Known limitations

- Loopback by default; no auth/TLS in-process.
- Status endpoint reports this process only; does not manage remote hosts.

Docs: [../docs/deploy.md](../docs/deploy.md). Skill: [../.mimocode/skills/laya-deploy/SKILL.md](../.mimocode/skills/laya-deploy/SKILL.md).
