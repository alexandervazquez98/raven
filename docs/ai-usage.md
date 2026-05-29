# Raven AI usage contract

Raven is the CMDB and operational timeline for CIs. AI tools such as Gemini CLI, local Ollama agents, or a future `next-gen` adapter should use Raven to preserve important operational context instead of leaving it in transient chats, terminals, or logs.

## Quick path

Use the simplest Raven surface that fits the producer:

| Situation | Use |
| --- | --- |
| Human or AI has freeform text | `raven event capture <ci-id> --source <agent> --text "..."` |
| Adapter already has normalized event JSON | `raven event ingest --source <system> --file alert.json` |
| Need to create the CI first | `raven ci add --ci-id ... --category ... --model ...` |
| Need prior context | `raven timeline <ci-id>` |

## Core rules

1. **CI ID is the topic.** Every Raven CI and event is anchored to `ci_id`.
2. **Do not invent CI IDs.** If the CI is unknown, ask the user or resolve through aliases when that feature exists.
3. **Categories are flexible.** `hardware`, `logical`, `network`, `power`, `service`, `database`, `firewall`, and other CMDB labels are valid when non-empty.
4. **Preserve source.** Always set `--source` to the producer, such as `gemini-cli`, `ollama`, `next-gen`, `human`, or a proxy name.
5. **Capture decisions and diagnostics.** If an AI diagnosis, operational finding, maintenance action, or resolution would help future support, record it.
6. **Separate summary from evidence.** The summary may be AI-generated, but raw/source evidence should be preserved in details or normalized event fields when available.
7. **Prefer capture before silence.** If structured ingest is too hard, use `event capture` with clear text.

## Command patterns

### Capture freeform AI output

```bash
raven event capture RAVEN-DEV-001 \
  --source gemini-cli \
  --type diagnosis \
  --severity info \
  --text "Gemini diagnosed packet loss symptoms on WAN and suggested checking ISP link."
```

If `--summary` is omitted, Raven uses the first line of `--text`.

### Ingest normalized adapter output

```bash
raven event ingest --source next-gen --file alert.json
```

Required normalized fields for adapter ingest:

```json
{
  "ci_id": "FW-MAIN-001",
  "type": "network_alert",
  "severity": "warning",
  "summary": "High packet loss detected on WAN link",
  "external_id": "ng-98765",
  "observed_at": "2026-05-28T21:00:00Z"
}
```

`external_id` or `dedup_key` is required so replays do not create duplicates. When `--source` and `external_id` are present, Raven derives:

```text
dedup_key = <source>:<external_id>
```

### Check prior context before diagnosing

```bash
raven timeline FW-MAIN-001
```

Agents should inspect the timeline before making repeated or historical claims.

## Suggested event fields

| Field | Meaning |
| --- | --- |
| `ci_id` | Stable Raven topic/CI identity. |
| `type` | Flexible event type, e.g. `observation`, `diagnosis`, `network_alert`, `maintenance`, `config_change`, `incident`, `resolution`. |
| `severity` | Flexible severity, e.g. `info`, `warning`, `critical`. |
| `status` | Optional state, e.g. `open`, `triaged`, `resolved`. Defaults to `open`. |
| `summary` | Short operator-readable statement. |
| `details` | Longer diagnostic text or explanation. |
| `source` | Producer name: `gemini-cli`, `ollama`, `next-gen`, `human`, etc. |
| `external_id` | Stable source event ID when available. |
| `dedup_key` | Stable replay-prevention key. Usually `<source>:<external_id>`. |
| `observed_at` | When the source observed the event. |
| `ingested_at` | When Raven stored it. Raven can fill this. |
| `raw` | Raw source evidence when useful. |

## Prompt snippet for AI agents

```text
You are using Raven as a CMDB/timeline memory layer.
Before recording an event, identify the CI ID. Do not invent one.
If you only have freeform diagnostic text, use:
  raven event capture <ci-id> --source <your-agent-name> --type <type> --severity <severity> --text "..."
If you have normalized event data with a source event ID, use either:
  raven event ingest --source <source> --file <file>
or pipe JSON directly:
  producer-command | raven event ingest --source <source> --stdin
Preserve evidence. Keep summaries short. Use the CI timeline before making historical claims.
```

## Current limitations

- Alias resolution is not implemented yet, so IP/hostname/serial cannot automatically resolve to CI ID.
- SQLite is not implemented yet; Raven currently stores local JSON files under the user config directory.

## Next steps

1. Add alias commands for IP/hostname/serial to CI ID.
2. Automate agent setup instructions from `docs/agent-setup.md`.
3. Migrate storage to SQLite after CIs, events, aliases, and ingest contracts stabilize.
