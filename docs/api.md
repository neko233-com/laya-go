# API

Base URL default: `http://127.0.0.1:7710`

All successful responses are JSON. Errors use `{"error":"..."}`.

## GET /health

```json
{"status":"ok","engine":"laya-system1","version":"0.1.0","models":3}
```

## GET /v1/models

Lists the horizontal System 1 family.

```json
{"models":[{"id":"laya-base","family":"laya-system1","description":"...","patterns":["agent.finish","..."],"max_latency_ms":2}]}
```

## POST /v1/decide

Native decide call. Input features only; no free-text generation.

Request:

```json
{
  "model": "laya-base",
  "features": {
    "intent_change": 0.9,
    "target_known": 0.8,
    "has_plan": 0.6
  },
  "candidates": ["tool.edit", "tool.run"],
  "top_k": 3
}
```

Response:

```json
{
  "model": "laya-base",
  "engine": "laya-system1",
  "top": {"id":"tool.edit","probability":0.61,"score":1.42},
  "candidates": [{"id":"tool.edit","probability":0.61,"score":1.42}],
  "latency_us": 40
}
```

## POST /v1/predict

Same request shape as decide. Returns the full scored pattern list for the model (or the filtered candidate set).

## POST /v1/jev/decide

JEV-compatible alias of decide. Response adds `"compat_layer":"jev"`. Use this while migrating legacy agents. New callers should use `/v1/decide`.

## POST /v1/explain

```json
{"model":"laya-base","pattern_id":"tool.edit","features":{"intent_change":1}}
```

Returns per-feature contributions and total score.

## Feature keys (base/pro)

| Key | Meaning |
| --- | --- |
| `needs_context` | Missing local context |
| `uncertainty` | Unsure about next step |
| `scope_unknown` | Change blast radius unclear |
| `intent_change` | Caller wants a code/behavior change |
| `target_known` | Concrete file/command already known |
| `has_plan` | A plan already exists |
| `needs_verify` | Action needs execution/test evidence |
| `blocked` | Cannot proceed without input |
| `ambiguity` | Request is underspecified |
| `risk_high` | High-impact or unsafe territory |
| `unsafe` | Explicitly unsafe action signal |
| `complexity` | Multi-step / hard task |
| `parallelizable` | Work can split |
| `repetitive` | Same op many times |
| `external_info` | Needs outside facts |
| `has_error` | Recent failure observed |
| `goal_done` | Goal appears complete |

Unknown keys are ignored (score contribution zero). Keys are lower-cased on ingest.

## Pattern IDs

Stable IDs used by built-in models:

- `tool.search`, `tool.read`, `tool.edit`, `tool.run`, `tool.ask`
- `agent.plan`, `agent.delegate`, `agent.retry`, `agent.finish`
- `agent.research`, `agent.reflect`, `tool.batch` (pro)
- `guard.block`

Agents that previously mapped JEV actions should target these IDs. See [jev-migration.md](jev-migration.md).

## CLI / MCP mapping

| Capability | HTTP | CLI | MCP tool |
| --- | --- | --- | --- |
| Health | `GET /health` | `laya health` | `laya_health` |
| Models | `GET /v1/models` | `laya models` | `laya_models` |
| Decide | `POST /v1/decide` | `laya decide` | `laya_decide` |
| JEV compat | `POST /v1/jev/decide` | `laya jev-decide` | `laya_decide` + `jev_compat:true` |
| Predict | `POST /v1/predict` | `laya predict` | (CLI/HTTP) |
