# laya-deploy

Go deployment server that hosts the Laya System 1 model service and exposes deployment status.

## What it does

- Serves the full Laya model API on one process (`/health`, `/v1/*`)
- Adds `GET /deploy/status` for ops (pid, uptime, binary path, model count)
- Default listen: `0.0.0.0:7710` (LAN)

## Run

```sh
go build -o bin/laya-deploy ./deploy
./bin/laya-deploy -Port 7710
# or
./bin/laya-deploy -addr 0.0.0.0:7710
```

One-click from repo root:

```powershell
.\deploy\deploy.ps1 -Port 7710
```

```sh
./deploy/deploy.sh --Port 7710
```

## Config

| Flag | Default | Meaning |
| --- | --- | --- |
| `-port` | `7710` | Listen port |
| `-addr` | empty | Full address; overrides `-port` |

Env for clients: `LAYA_URL=http://127.0.0.1:7710`

## Known limitations

- Bind `0.0.0.0` is intentional for LAN agent clients; there is no in-process auth. Firewall the port on untrusted networks.
- Status endpoint reports this process only; does not manage remote hosts.

Docs: [../docs/deploy.md](../docs/deploy.md). Skill: [../.mimocode/skills/laya-deploy/SKILL.md](../.mimocode/skills/laya-deploy/SKILL.md).
