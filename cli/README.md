# laya CLI

Agent-side command line for the Laya System 1 decision server. Replaces legacy JEV CLIs.

## What it does

Calls the Laya HTTP API: `health`, `models`, `decide`, `jev-decide`, `predict`.

## Run

```sh
go build -o bin/laya ./cli
./bin/laya health --url http://127.0.0.1:7710
./bin/laya decide --feature intent_change=0.9 --feature target_known=0.8 --top-k 3 --json
```

Config: `LAYA_URL` or `--url`. Default `http://127.0.0.1:7710`.

## Config

Features are `key=value` flags. Example keys: `intent_change`, `blocked`, `goal_done`.

## Known limitations

- Assumes a reachable `laya-server`.
- Does not parse natural-language tasks into features.

Docs: [../docs/api.md](../docs/api.md), [../docs/jev-migration.md](../docs/jev-migration.md). Skill: [../.mimocode/skills/laya-cli/SKILL.md](../.mimocode/skills/laya-cli/SKILL.md).
