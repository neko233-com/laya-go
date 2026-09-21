# JEV migration

Laya replaces the legacy JEV structured decision service. Same job in the agent loop: fast pattern probability, no text generation. Different implementation: open model family, open HTTP/MCP contract, Go server in this repository.

## What stays the same

- Callers still submit structured state (features / signals).
- Callers still receive ranked pattern IDs with probabilities.
- Decision latency remains on the System 1 budget (microseconds to low milliseconds on localhost).

## What changes

| Before (JEV) | After (Laya) |
| --- | --- |
| JEV server binary / endpoint | `laya-server` (default `127.0.0.1:7710`) |
| `jev decide` style CLI | `laya decide` / `laya jev-decide` |
| `jev_*` MCP tools | `laya_health` / `laya_models` / `laya_decide` |
| Closed pattern service | Open family: `laya-mini` / `laya-base` / `laya-pro` |
| Ad-hoc action names | Stable Laya pattern IDs (see [api.md](api.md)) |

## Migration paths

### HTTP agents

1. Point base URL to the Laya server.
2. Keep sending the same feature map on `POST /v1/jev/decide` during the cutover window.
3. Switch to `POST /v1/decide` and map old action names to Laya pattern IDs.
4. Remove the JEV client package after one release cycle.

### CLI agents

```sh
# old
jev decide --feature intent_change=0.9

# transition
laya jev-decide --feature intent_change=0.9 --json

# target
laya decide --model laya-base --feature intent_change=0.9 --feature target_known=0.8
```

### MCP agents

Replace the JEV MCP server entry with `laya-mcp`. Example stdio config shape (no secrets):

```json
{
  "mcpServers": {
    "laya": {
      "command": "path/to/laya-mcp",
      "env": {
        "LAYA_URL": "http://127.0.0.1:7710"
      }
    }
  }
}
```

Tool rename map:

| JEV tool (legacy) | Laya tool |
| --- | --- |
| jev health / jev_ping | `laya_health` |
| jev models | `laya_models` |
| jev decide | `laya_decide` |

Set `"jev_compat": true` in `laya_decide` arguments only while legacy semantics are still required.

Full agent integration (Codex CLI, Claude Code, MiMo, HTTP, CLI): [agent-integration.md](agent-integration.md).

## Action name mapping starters

| Legacy intent | Laya pattern ID |
| --- | --- |
| search / grep / locate | `tool.search` |
| open / read | `tool.read` |
| patch / write / modify | `tool.edit` |
| exec / test / build | `tool.run` |
| clarify / ask | `tool.ask` |
| plan / outline | `agent.plan` |
| spawn worker / parallel task | `agent.delegate` |
| retry / recover | `agent.retry` |
| done / complete | `agent.finish` |
| deny / unsafe stop | `guard.block` |

Keep a project-local mapping table if your JEV vocabulary is richer; do not fork engine code for naming.

## Rollback

Keep the JEV binary or previous client available for one release. Laya `laya-pro` is a superset of `laya-base` patterns; rolling back model id is config-only and does not require redeploying callers.
