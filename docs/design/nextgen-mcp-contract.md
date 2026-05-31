# next-gen MCP contract

Raven will use a dedicated next-gen MCP surface so AI agents can query next-gen through safe tools, then use Raven MCP to resolve identity, read history, and persist selected facts.

## Quick path

1. Start with read-only tools only.
2. Authenticate to next-gen with environment-provided API settings.
3. Return next-gen identifiers as upstream references, not Raven canonical IDs.
4. Let Raven tools resolve aliases and record events.
5. Add mutating tools only after read-only behavior is validated.

## Runtime shape

Recommended command:

```bash
raven nextgen-mcp
```

Alternative if we want one MCP entrypoint later:

```bash
raven mcp nextgen --read-only
```

Keep it explicit so operators can distinguish:

| Server | Responsibility |
| --- | --- |
| `raven mcp` | Local Raven CMDB/timeline tools. |
| `raven nextgen-mcp` | Read-only proxy to next-gen REST API. |

## Configuration

| Env var | Required | Meaning |
| --- | --- | --- |
| `NEXTGEN_BASE_URL` | yes | next-gen backend base URL, e.g. `http://localhost:8000` for local dev or `https://ops.example/proxy`; tools append `/api/...` while preserving path prefixes. |
| `NEXTGEN_ACCESS_TOKEN` | yes for first version | Bearer token used for next-gen requests. Prefer an `AI_DIAGNOSTIC` token. |
| `NEXTGEN_REFRESH_TOKEN` | later | Optional refresh token for one retry on 401. |
| `NEXTGEN_TIMEOUT` | optional | Positive HTTP timeout, default `10s`. Non-positive values are invalid. |
| `NEXTGEN_USER_AGENT` | optional | Default `raven-nextgen-mcp/<version>`. |

First implementation should require `NEXTGEN_BASE_URL` and `NEXTGEN_ACCESS_TOKEN`. Add refresh support only after the simple path is stable. To protect bearer tokens, `http://` is allowed only for `localhost` or loopback hosts; remote deployments must use `https://`.

## Common response envelope

All tools should return a predictable envelope:

```json
{
  "ok": true,
  "source": "next-gen",
  "endpoint": "GET /api/events/{event_id}",
  "fetched_at": "2026-05-30T00:00:00Z",
  "data": {}
}
```

Response bodies are bounded before decoding so a misconfigured remote endpoint cannot exhaust MCP process memory. Error details are truncated before being returned to the agent.

Errors should be readable and structured when possible:

```json
{
  "ok": false,
  "source": "next-gen",
  "endpoint": "GET /api/events/{event_id}",
  "status_code": 404,
  "detail": "Event not found"
}
```

## Read-only tools: milestone 1

### `nextgen_list_events`

Lists next-gen events.

Maps to:

```http
GET /api/events?status=<status>
```

Input:

```json
{
  "status": "ACTIVE"
}
```

Allowed `status` values:

```text
OPEN, ACK, CLOSED, RECOVERED, ACTIVE, CONSOLE
```

Output `data`:

```json
{
  "events": [
    {
      "id": "evt-001",
      "ci_id": "42",
      "status": "OPEN",
      "severity": "WARNING",
      "message": "High packet loss detected",
      "created_at": "2026-05-30T00:00:00Z",
      "last_seen": "2026-05-30T00:05:00Z",
      "ci_name": "fw-main",
      "ci_hostname": "10.53.1.22",
      "metric_name": "packet_loss",
      "metric_protocol": "ICMP"
    }
  ]
}
```

### `nextgen_get_event`

Fetches one event detail.

Maps to:

```http
GET /api/events/{event_id}
```

Input:

```json
{
  "event_id": "evt-001"
}
```

Output `data` should preserve next-gen's detail shape:

```json
{
  "event": {
    "id": "evt-001",
    "ci_id": "42",
    "severity": "WARNING",
    "status": "OPEN",
    "message": "High packet loss detected",
    "created_at": "2026-05-30T00:00:00Z",
    "last_seen": "2026-05-30T00:05:00Z",
    "ci_ref": {
      "id": "42",
      "label": "fw-main",
      "hostname": "10.53.1.22",
      "location_name": "HQ"
    }
  },
  "business_context": {},
  "itsm_context": {}
}
```

### `nextgen_search_cis`

Searches next-gen CIs by text, IP, hostname, or label.

Maps to:

```http
GET /api/nodes/search?q=<query>
```

Input:

```json
{
  "query": "10.53.1.22",
  "limit": 10
}
```

Output `data`:

```json
{
  "cis": [
    {
      "id": "42",
      "label": "fw-main",
      "ip": "10.53.1.22",
      "status": "ACTIVE",
      "brand": "Fortinet",
      "model": "FortiGate"
    }
  ],
  "count": 1
}
```

### `nextgen_get_ci_events`

Lists active/open events for one next-gen CI.

Maps to:

```http
GET /api/events/related/{ci_id}
```

Input:

```json
{
  "ci_id": "42"
}
```

Output `data`:

```json
{
  "ci_id": "42",
  "events": []
}
```

### `nextgen_get_ci_metrics`

Reads applicable metrics for one next-gen CI.

Maps to:

```http
GET /api/nodes/{node_id}/metrics
```

Input:

```json
{
  "node_id": "42"
}
```

Output `data`:

```json
{
  "node_id": "42",
  "metrics": []
}
```

## Raven bridge contract

next-gen tools must not invent Raven `ci_id`. They return upstream references that Raven can resolve.

Recommended reference object:

```json
{
  "source": "next-gen",
  "type": "ci_id",
  "value": "42"
}
```

Skill flow:

```text
nextgen_get_event(event_id)
→ raven_resolve_ci_ref(source="next-gen", type="ci_id", value=event.ci_ref.id)
→ raven_get_timeline(ci_id=<canonical Raven CI>)
→ raven_record_event(...)
```

## Normalization helper: milestone 2

After read-only tools work, add a helper that returns a Raven candidate payload but does not write it:

```text
nextgen_build_raven_event_candidate
```

Input:

```json
{
  "event_id": "evt-001",
  "canonical_ci_id": "RAVEN-FW-MAIN-001"
}
```

Output:

```json
{
  "candidate": {
    "ci_id": "RAVEN-FW-MAIN-001",
    "type": "network_alert",
    "severity": "warning",
    "summary": "High packet loss detected",
    "source": "next-gen",
    "external_id": "evt-001",
    "observed_at": "2026-05-30T00:00:00Z",
    "raw": "{...original next-gen detail...}"
  }
}
```

If no canonical Raven ID is known, emit `ci_ref` instead:

```json
{
  "ci_ref": {
    "source": "next-gen",
    "type": "ci_id",
    "value": "42"
  }
}
```

## Raw/redaction policy before Raven persistence

Milestone 1 candidate generation is read-only. Before an agent calls `raven_record_event` or `raven event ingest`, it must reduce the candidate to operator-safe evidence.

Persist by default:

- next-gen event ID, CI reference, severity, status, timestamps, message/summary, metric name/protocol, and concise business/ITSM context.
- Evidence snippets needed to understand the incident or resolution.

Redact or omit before persistence:

- access tokens, cookies, API keys, authorization headers, passwords, SNMP communities, private keys, session IDs, refresh tokens.
- User PII not needed for operations, chat transcripts, browser/client metadata, full request headers, and unrelated nested payloads.
- Large raw payloads. Keep `raw` small enough for review; prefer a short JSON object with source IDs and selected evidence over the complete next-gen response.

If the agent is unsure whether a field is sensitive, omit it from Raven and mention that raw next-gen detail remains available in next-gen.

## Mutating tools: later milestone

Do not implement these in milestone 1:

```text
nextgen_run_diagnostic
nextgen_ack_event
nextgen_comment_event
nextgen_close_event
```

Reasons:

- next-gen applies cooldowns and behavioral guards.
- critical events require human escalation.
- AI cannot force-close events.
- normal close notes require `Causa raíz:` and `Nota:`.

When implemented, each mutating tool must be explicit, non-read-only, and return next-gen's audit result.

## MCP annotations

Read-only next-gen tools should use:

```text
readOnlyHint = true
destructiveHint = false
idempotentHint = true
openWorldHint = true
additionalProperties = false
```

`openWorldHint=true` because the tools query an external system, unlike local Raven storage.

## Validation plan

Unit tests should use a fake HTTP server, not a real next-gen instance.

Required coverage:

- Missing `NEXTGEN_BASE_URL` or `NEXTGEN_ACCESS_TOKEN` returns a clear setup error.
- Successful GET maps into the common envelope.
- 401/403/404/5xx return readable MCP tool errors.
- Query parameters are encoded correctly.
- Base URL path prefixes are preserved and path parameters are escaped exactly once.
- `http://` URLs are rejected except localhost/loopback development URLs.
- Response bodies are capped and oversized error details are truncated.
- Non-positive `NEXTGEN_TIMEOUT` values are rejected.
- Tool schemas reject unknown fields where transport validation is available.
- Handlers also validate status enum, non-empty IDs, query length, and limit range.
- Tool annotations mark milestone-1 tools read-only.
- Existing `raven mcp` tools remain unchanged.

Suggested commands:

```bash
go test ./internal/nextgen ./internal/nextgenmcp ./internal/cli
go test ./...
```

## Open decisions

- Auth refresh remains deferred.
- Exact max persisted `raw` byte size and whether Raven should enforce it in storage/CLI.
- Whether transport-level schema rejection should be tested with an MCP client/integration test instead of direct handler tests.
