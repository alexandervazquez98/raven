# Raven event ingest flow

Raven is the CMDB and operational timeline layer. It does not monitor the network directly. The `next-gen` ecosystem observes CI activity, an AI layer normalizes the signal, and Raven stores auditable events against stable CI IDs.

## Quick path

1. `next-gen` detects or receives a signal about a CI.
2. The AI layer extracts the CI ID, event type, severity, summary, and evidence.
3. Raven ingests the normalized event and deduplicates by source metadata.
4. Operators inspect the CI timeline in Raven.

## Current decision

| Topic | Decision |
| --- | --- |
| Raven role | CMDB + timeline memory, not a network monitor. |
| Primary identity | Raven records store canonical `ci_id`; ingest may resolve an explicit `ci_ref` alias when `ci_id` is absent. |
| Categories | Required but flexible CMDB labels. |
| Event source | Preserve source metadata from `next-gen`. |
| AI role | Interpret and normalize; do not invent missing facts. |
| Evidence | Store source/external ID, dedup key, timestamps, and raw/reference data. |

## Flow diagram

```mermaid
flowchart TD
    NG[next-gen ecosystem<br/>monitoring, discovery, alerts] -->|raw signal / payload| AI[AI normalization layer]

    AI -->|extracts| CIID[CI ID]
    AI -->|classifies| META[type, severity, status]
    AI -->|summarizes| SUMMARY[summary and details]
    AI -->|preserves| EVIDENCE[source, external_id, raw evidence]

    CIID --> CHECK{CI ID known?}
    CHECK -->|yes| INGEST[Raven event ingest]
    CHECK -->|no| RESOLVE[resolve by alias<br/>IP / hostname / serial]
    RESOLVE -->|resolved| INGEST
    RESOLVE -->|not resolved| ESCALATE[ask for CI ID<br/>or hold as unassociated candidate]

    INGEST --> DEDUP{dedup_key exists?}
    DEDUP -->|new| STORE[(Raven storage)]
    DEDUP -->|duplicate| UPDATE[ignore or update existing event]

    STORE --> TIMELINE[CI timeline]
    STORE --> CMDB[CI registry]

    TIMELINE --> HUMAN[operator / support workflow]
    CMDB --> HUMAN
```

## Event shape

```go
type Event struct {
    ID         string
    CIID       string
    Type       string
    Severity   string
    Status     string
    Summary    string
    Details    string
    Source     string
    ExternalID string
    DedupKey   string
    ObservedAt time.Time
    IngestedAt time.Time
    Raw        string
}
```

## CLI shape under consideration

Manual event entry:

```bash
raven event add RAVEN-DEV-001 \
  --type observation \
  --severity info \
  --summary "Initial Raven CI validated" \
  --source human
```

Normalized ingest from `next-gen` or an AI adapter can read either a JSON file or piped JSON from stdin:

```bash
raven event ingest --source next-gen --file alert.json

next-gen-export-alert | raven event ingest --source next-gen --stdin
```

Timeline inspection:

```bash
raven timeline RAVEN-DEV-001
```

## Rules for AI adapters

- Do not create or update Raven records without a canonical Raven CI ID unless an explicit unresolved-event flow exists.
- Treat `next-gen` IDs as aliases/references, not Raven identity.
- If the normalized payload has no `ci_id`, include `ci_ref` with `source`, `type`, and `value` so Raven can resolve it through aliases during ingest.
- Preserve source evidence. The summary is AI-generated; the source metadata is the audit trail.
- Use a stable `dedup_key`, preferably `<source>:<external_id>` when available.
- Do not treat category as a closed enum; Raven supports flexible CMDB categories.

## Current implementation status

Implemented Raven surfaces:

1. `Event` domain model.
2. JSON event storage at `~/.config/raven/events.json`.
3. JSON alias/reference storage at `~/.config/raven/aliases.json`.
4. CLI commands: `event add`, `event capture`, `event ingest`, `timeline`, `alias add`, `alias list`, and `alias resolve`.
5. `event ingest` accepts exactly one input source: `--file <json>` or `--stdin`.
6. Deduplication by `dedup_key` for structured ingest.
7. `event ingest` resolves missing `ci_id` values from `ci_ref.source + ci_ref.type + ci_ref.value` aliases.

Alias setup examples:

```bash
raven alias add --ci-id RAVEN-FW-MAIN-001 --source next-gen --type ci_id --value 42
raven alias add --ci-id RAVEN-FW-MAIN-001 --source next-gen --type hostname --value fw-main
raven alias resolve --source next-gen --type ci_id --value 42
```

The alias key is `source + type + value`; the canonical target remains Raven's `ci_id`. Alias values are exact-match after trimming whitespace, so adapters should normalize hostname, MAC, IP, serial, and upstream ID formats before calling `alias add`, `alias resolve`, or `event ingest` with `ci_ref`.

Normalized next-gen ingest by alias reference:

```json
{
  "ci_ref": {
    "source": "next-gen",
    "type": "ci_id",
    "value": "42"
  },
  "type": "network_alert",
  "severity": "warning",
  "summary": "High packet loss detected on WAN link",
  "external_id": "evt-98765",
  "observed_at": "2026-05-28T21:00:00Z",
  "raw": "{...original next-gen payload...}"
}
```

```bash
raven event ingest --source next-gen --file alert.json
```

## Next step

Continue with one of:

1. dedicated next-gen adapter command or script that emits normalized Raven event payloads.
2. `raven setup <agent>` automation for Gemini/Ollama/Codex/OpenCode instruction insertion.
3. SQLite storage once CIs, events, aliases, and ingest contracts stabilize.
