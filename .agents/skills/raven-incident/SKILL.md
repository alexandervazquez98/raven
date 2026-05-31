---
name: raven-incident
description: "Trigger: incident, alert, alarm, CI, IP, hostname, next-gen, diagnose, repair, outage. Handles Raven incident intake, enrichment, capture, and resolution."
license: Apache-2.0
metadata:
  author: raven
  version: "0.1"
---

## Activation Contract

Use this skill when the user reports an operational incident, alert, alarm, outage, degraded service, packet loss, CI problem, IP, hostname, serial, MAC, next-gen event, or asks to diagnose/repair/resolve an infrastructure issue.

Do not use it for ordinary code changes unless testing this workflow.

## Hard Rules

- Never invent a `ci_id`.
- Resolve aliases before recording events when the user gives IP/hostname/serial/MAC/next-gen ID instead of canonical Raven CI.
- Read a short Raven timeline before making historical claims.
- Query or request next-gen event details when the report references next-gen or external telemetry.
- Persist facts, actions, and outcomes; do not persist private chain-of-thought or unconfirmed speculation as fact.
- Ask before destructive repair actions.

## Tool Actions

Use these actions in order; prefer MCP when available and CLI fallback otherwise:

| Action | MCP | CLI fallback |
| --- | --- | --- |
| Resolve CI/ref | `raven_resolve_ci_ref` | `raven alias resolve --source <source> --type <type> --value <value>` |
| Read timeline | `raven_get_timeline` | `raven timeline <ci-id>` |
| Confirm CI | `raven_get_ci` / `raven_list_cis` | `raven ci show <ci-id>` / `raven ci list` |
| Record event | `raven_record_event` | `raven event capture` or `raven event ingest` |

Next-gen enrichment is required when next-gen/event telemetry is involved. Prefer the read-only next-gen MCP tools when configured: `nextgen_list_events`, `nextgen_get_event`, `nextgen_search_cis`, `nextgen_get_ci_events`, `nextgen_get_ci_metrics`, and `nextgen_build_raven_event_candidate`. If the adapter is unavailable, ask the user for the event payload/output. Mutating next-gen tools such as diagnostic runs, ack, comments, and close actions are intentionally future work and require explicit operator approval.

## Workflow

1. Extract candidate identifiers: `ci_id`, IP, hostname, serial, MAC, next-gen ID, event/ticket reference.
2. If canonical CI is unknown, resolve with Raven alias tools/CLI. If unresolved, ask one focused question.
3. Read recent context with `raven_get_timeline` or `raven timeline <ci-id>` and keep only relevant facts.
4. Enrich from next-gen when applicable; collect raw payload, `external_id`, observed time, severity, affected reference, `business_context`, `itsm_context`, and evidence.
5. Respect next-gen AI guardrails: cooldowns, no forced close, critical event human escalation, and normal close note shape (`Causa raíz:` + `Nota:`).
6. Record the initial useful event with `raven_record_event` or `raven event capture/ingest` once identity and evidence are sufficient.
7. Support diagnosis/repair, separating observations from hypotheses.
8. On closure, record a concise `resolution`; if unresolved, record `follow_up` with next action.

## Event Type Guide

| Type | Use when |
| --- | --- |
| `network_alert` | Telemetry/next-gen reports a network symptom. |
| `incident` | Active CI/service problem is confirmed. |
| `diagnosis` | A meaningful finding or suspected cause is established. |
| `maintenance` | A repair/configuration/physical action is performed. |
| `resolution` | Recovery is confirmed. |
| `follow_up` | Work remains or more evidence is needed. |

## Output Contract

Return a compact incident status:

```text
CI: <ci-id or unresolved>
Context: <1-3 relevant timeline facts>
Next-gen: <event summary or missing>
Recorded: <event id or not recorded + why>
Status: <triage|repairing|resolved|follow_up>
Next: <single next action>
```

## References

- `../../../docs/design/raven-incident-workflow.md` — full workflow, examples, and open questions.
