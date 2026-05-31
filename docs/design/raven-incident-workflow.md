# Raven incident workflow skill

Raven needs an agent skill that turns an operational report into a short, traceable incident workflow: identify the CI, gather context, enrich from next-gen, persist useful facts, support repair, and record the outcome.

## Quick path

1. Trigger when the user reports an incident, alert, CI problem, IP, hostname, or next-gen reference.
2. Resolve the canonical Raven CI before recording anything.
3. Read a short Raven timeline for context.
4. Query or request next-gen event details.
5. Ingest selected facts into Raven.
6. Assist detection/repair.
7. Persist a concise resolution or next step.

## Skill authoring rules

| Rule | Decision |
| --- | --- |
| Location | Use `.agents/skills/raven-incident/SKILL.md` so Codex and Gemini can discover the same project skill. |
| Shape | Keep `SKILL.md` concise and imperative. Put examples and future schemas in `references/` if they grow. |
| Triggering | Put essential trigger words in `description`: incident, alert, alarm, CI, IP, hostname, next-gen, diagnose, repair, outage. |
| Scope | One job only: operational incident intake and resolution capture. Not general memory, not generic troubleshooting. |
| Tools | Prefer Raven MCP tools when available. Fall back to `raven` CLI. Treat next-gen integration as a required enrichment step when a next-gen reference/event exists. |
| Safety | Never invent `ci_id`. Ask or resolve aliases first. Do not persist speculative root cause as fact. |
| Testing | Use representative prompts and verify that a fresh agent loads the skill, asks for missing identity, reads timeline, and proposes/records the right event. |

## Activation contract

Use the skill when the user says or implies any of these:

- There is an incident, alert, alarm, failure, outage, degradation, packet loss, high latency, device down, service down, or no connectivity.
- The user gives a CI ID, IP, hostname, serial, MAC, next-gen ID, ticket/event ID, or other operational reference.
- The user asks to diagnose, detect, triage, repair, resolve, follow up, or record an operational problem.

Do not use the skill for normal code tasks, generic product questions, or Raven implementation work unless the task is to test the incident workflow itself.

## Tool surface

The skill should think in terms of small tools/actions, even when the first implementation is MCP or CLI. This keeps the workflow portable between Codex, Gemini, and future wrappers.

### Existing Raven tools

| Skill action | MCP tool | CLI fallback | Purpose |
| --- | --- | --- | --- |
| Resolve CI reference | `raven_resolve_ci_ref` | `raven alias resolve --source <source> --type <type> --value <value>` | Convert next-gen/IP/hostname/serial/MAC reference into canonical Raven `ci_id`. |
| Read CI context | `raven_get_timeline` | `raven timeline <ci-id>` | Load short prior context before diagnosis or historical claims. |
| Record event | `raven_record_event` | `raven event capture` or `raven event ingest` | Persist selected facts, diagnosis, maintenance action, resolution, or follow-up. |
| List CIs | `raven_list_cis` | `raven ci list` | Help the user choose when identity is ambiguous. |
| Get CI | `raven_get_ci` | `raven ci show <ci-id>` | Confirm category/model/details for a known CI. |

### Needed next-gen tools

`next-gen` already exposes an AI-facing REST workflow; it is not currently a Raven producer. The skill should be written around these actions so an adapter can call the API safely.

| Skill action | next-gen REST source | Proposed adapter tool | Input | Output |
| --- | --- | --- | --- | --- |
| List events | `GET /api/events?status=ACTIVE` | `nextgen_list_events` | optional status/window | Candidate events with `id`, `ci_id`, severity, status, message, timestamps, CI display fields. |
| Fetch next-gen event | `GET /api/events/{event_id}` | `nextgen_get_event` | `event_id` | Detail payload with `event`, `business_context`, `itsm_context`, `ci_ref`. |
| Search CI | `GET /api/nodes/search?q=...` or `GET /api/nodes` | `nextgen_search_cis` | IP, hostname, label, serial, text | Candidate CIs with id/label/ip/status/model metadata. |
| Related CI events | `GET /api/events/related/{ci_id}` | `nextgen_get_ci_events` | next-gen `ci_id` | Active/open related events. |
| Run diagnostic | `POST /api/events/{event_id}/diagnose` | `nextgen_run_diagnostic` | `event_id` | Ping/SNMP diagnostic result; subject to 5 min AI cooldown. |
| Acknowledge/comment | `POST /api/events/{event_id}/ack`, `/comment` | `nextgen_ack_event`, `nextgen_comment_event` | `event_id`, note | Audit/comment in next-gen; subject to AI guardrails. |
| Close event | `POST /api/events/{event_id}/close` | `nextgen_close_event` | `event_id`, root cause, note | Normal closure; no forced close; critical events require human escalation. |
| Normalize payload | adapter code | `raven_normalize_nextgen_event` | next-gen detail + optional canonical Raven `ci_id` | Raven ingest JSON with `ci_id` or `ci_ref`, `type`, `severity`, `summary`, `external_id`, `observed_at`, `raw`. |

If next-gen MCP is unavailable or unconfigured, the skill must ask the user for the next-gen payload/output and then use Raven's existing ingest/capture tools.

### next-gen AI constraints to preserve

- AI roles are `AI_DIAGNOSTIC` and `AI_OPERATOR`; JWT must include an AI role/type.
- Diagnostics, acknowledgement, closure, and metadata updates are guarded by cooldowns and behavioral limits.
- AI cannot force-close events; critical events require human escalation.
- Normal close notes require `Causa raíz:` and `Nota:` shape in next-gen.
- Raven should store next-gen actions as evidence, not bypass next-gen's own audit/guard layer.

## Workflow

### 1. Identify the target

Extract all candidate identifiers from the user message:

- Raven `ci_id`
- IP address
- hostname
- serial
- MAC address
- next-gen CI/event ID
- ticket/reference ID

If a canonical `ci_id` is not already known, resolve aliases:

```bash
raven alias resolve --source <source> --type <ci_id|ip|hostname|serial|mac> --value <value>
```

If identity cannot be resolved, ask one focused question. Do not invent IDs.

### 2. Load short context

Read recent Raven context:

```bash
raven timeline <ci-id>
```

Keep only the relevant context: recent matching alerts, prior maintenance, open incidents, and recent resolutions.

### 3. Enrich from next-gen

When the user gives a next-gen reference or when the event likely originated from next-gen, retrieve as much structured event detail as available:

- source event ID
- observed timestamp
- affected CI/reference
- symptom
- severity
- status
- raw evidence
- impacted service/path
- suggested or detected cause

Until a next-gen adapter exists, ask the user for the event payload or command output.

### 4. Persist selected facts

Record the initial event when there is enough identity and evidence. Prefer normalized ingest for next-gen payloads; otherwise capture a concise event:

```bash
raven event capture <ci-id> \
  --source <codex|gemini-cli|next-gen|human> \
  --type <incident|network_alert|diagnosis|maintenance|resolution|follow_up> \
  --severity <info|warning|critical> \
  --summary "<short summary>" \
  --text "<evidence and selected details>"
```

Persist facts, not speculation. If the cause is unconfirmed, mark it as suspected in details. Before persisting next-gen `raw`, redact secrets and irrelevant PII; prefer selected evidence over full payloads.

### 5. Support repair

During diagnosis/repair:

- Use the timeline before making historical claims.
- Separate observed evidence from hypotheses.
- Ask before destructive actions.
- Record important milestones only, not every thought.

### 6. Close or defer

If resolved, record a `resolution` event with:

- what changed
- why it fixed the issue
- evidence of recovery
- operator or agent source

If unresolved, record `follow_up` or `incident` status with:

- current state
- next recommended action
- evidence still missing

## Event type guide

| Type | Use when |
| --- | --- |
| `network_alert` | Telemetry/next-gen reports a network symptom. |
| `incident` | User/operator confirms an active service or CI problem. |
| `diagnosis` | A meaningful cause, hypothesis, or finding is established. |
| `maintenance` | A repair/configuration/physical action is performed. |
| `resolution` | The problem is confirmed resolved. |
| `follow_up` | Work remains or extra validation is needed. |

## Evaluation prompts

Use these prompts in fresh Codex/Gemini sessions:

1. `Tengo una alerta de next-gen para la IP 10.53.1.22 con packet loss alto.`
   - Expected: resolve IP or ask for alias, read timeline, request/query next-gen details, do not invent CI.
2. `FW-MAIN-001 está sin conectividad desde hace 10 minutos.`
   - Expected: read timeline, record or propose incident capture, start diagnosis.
3. `Ya se resolvió FW-MAIN-001: era un cable WAN flojo, se reconectó y volvió el tráfico.`
   - Expected: record concise `resolution` with evidence/summary.

## Open questions

- Exact next-gen query surface: CLI, HTTP API, MCP, or file ingest?
- Required Raven tag schema and command names.
- Whether agents should auto-record initial incidents or ask first.
- Whether `event capture` should support tags/status updates directly.
