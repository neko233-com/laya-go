# Architecture

Laya is a horizontal System 1 decision model family for agents. It replaces the legacy JEV decision service with an open Go implementation.

## Why System 1

Agents need a fast, structured decision signal on every step: which pattern fits the current state (read, edit, run, ask, finish, guard, ...). Autoregressive text generation is the wrong tool for that loop:

- latency budget is sub-millisecond to a few milliseconds
- output space is a finite pattern set, not free text
- callers need calibrated probabilities, not prose

Laya therefore scores structured features against registered patterns in one pass. There is no token loop.

## Components

```text
                    +------------------+
  agent runtime --->|  cli/  (laya)    |---
                    +------------------+   \
                    +------------------+    \    +----------------------+
  Claude/MiMo  --->|  mcp/  (laya-mcp) |----+-->| server/ (laya-server)|
                    +------------------+    /    |  internal/engine     |
                    +------------------+   /     +----------------------+
  any HTTP client ->| /v1/decide etc.  |---
                    +------------------+
```

- `server/` owns the HTTP contract and process lifecycle.
- `cli/` and `mcp/` are agent-side entry points; both use `internal/client`.
- `internal/engine` is the only place that computes probabilities.

## Scoring model

For each pattern `p` and feature vector `x`:

```text
score(p) = bias(p) + Σ weight(p,f) * x[f] + ε * priority(p)
prob     = softmax(scores, temperature)
```

All patterns are scored from the same feature snapshot. Ranking is by probability. Optional candidate filters restrict the pattern allow-list before scoring.

## Built-in family

| Model | Role | Latency budget |
| --- | --- | --- |
| `laya-mini` | High-frequency agent steps | 1 ms |
| `laya-base` | Default tool routing | 2 ms |
| `laya-pro` | Wider pattern set for complex plans | 5 ms |

Pattern IDs are stable agent contracts. See [api.md](api.md) and [jev-migration.md](jev-migration.md).

## Trust boundary

Laya is an open local/remote decision oracle. It never executes tools. Callers remain responsible for authorization, safety gates, and side effects. The `guard.block` pattern is advisory signal only; agents must still enforce policy.

## Related docs

- [api.md](api.md) — wire contract
- [jev-migration.md](jev-migration.md) — replace JEV callers
- [development.md](development.md) — build and test
