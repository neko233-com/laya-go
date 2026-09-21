# laya-server

HTTP System 1 decision server for the open Laya model family.

## What it does

Serves `/health`, `/v1/models`, `/v1/decide`, `/v1/predict`, `/v1/explain`, and the JEV-compatible `/v1/jev/decide`. Scoring is non-autoregressive; responses are ranked pattern probabilities.

## Run

```sh
go build -o bin/laya-server ./server
./bin/laya-server -addr 127.0.0.1:7710
```

Config: `LAYA_ADDR` or `-addr`. Default `127.0.0.1:7710`.

## Config

No credentials. Bind to loopback unless you intentionally expose a trusted network; this process has no auth layer.

## Known limitations

- No built-in authentication or TLS termination; put a reverse proxy in front for non-local exposure.
- Pattern weights are open rules in `internal/engine/models.go`.

Docs: [../docs/api.md](../docs/api.md), [../docs/architecture.md](../docs/architecture.md). Skill: [../.mimocode/skills/laya-server/SKILL.md](../.mimocode/skills/laya-server/SKILL.md).
