# Raven

Raven is a local CMDB and operational timeline for configuration items (CIs). It helps operators and AI integrations resolve CI identity, inspect prior context, and preserve important operational evidence in one local workflow.

## Quick path

1. From the repository root, run the guided setup flow:

   ```bash
   raven setup
   ```

2. Use a known canonical Raven CI ID to inspect prior context:

   ```bash
   raven timeline <ci-id>
   ```

3. If an upstream identifier is provided, resolve it through Raven aliases before treating it as canonical:

   ```bash
   raven alias resolve --source <source> --type <ci_id|ip|hostname|serial|mac> --value <value>
   ```

## What Raven is for

Raven is for local operations work where CI identity and history matter:

- Keep a small CMDB of configuration items.
- Read the operational timeline for a known CI before diagnosing.
- Capture approved diagnostics, observations, maintenance actions, incidents, and resolutions.
- Give supported AI tools the same safety rules and context surfaces operators use.

## How Raven memory works

Raven memory is CI-centered:

1. A canonical `ci_id` identifies the configuration item.
2. Aliases map upstream references, such as hostnames, IPs, serials, MACs, or external system IDs, to that `ci_id`.
3. Events record approved observations, diagnostics, maintenance actions, incidents, and resolutions.
4. The timeline reads those events before future diagnosis or historical claims.

```text
canonical ci_id
├── aliases / upstream references
└── events
    └── timeline
```

For the full command and AI contract, see [AI usage](docs/ai-usage.md). For the ingest flow diagram, see [Raven event ingest flow](docs/design/next-gen-event-ingest.md).

## Core rules

- CI ID is mandatory. Do not invent CI IDs.
- If you have an upstream identifier, resolve it through Raven aliases before treating it as canonical.
- Before diagnosing a known CI, inspect prior context with `raven timeline <ci-id>` when useful.
- Record important approved diagnostics, observations, maintenance actions, incidents, and resolutions with `raven event capture`.
- Use `raven event ingest --source <source> --file <json>` only for normalized event JSON with `external_id` or `dedup_key`.
- Preserve source evidence, redact secrets, and keep summaries short.

## Guides

- [AI usage](docs/ai-usage.md) — Raven memory rules for humans, AI agents, MCP tools, and adapters.
- [Agent setup plan](docs/agent-setup.md) — setup wizard behavior and project-local AI integration targets.
- [Raven incident workflow](docs/design/raven-incident-workflow.md) — incident intake, enrichment, capture, and resolution flow.
- [next-gen event ingest](docs/design/next-gen-event-ingest.md) — how next-gen signals become Raven timeline events.

## Verification checklist

- [ ] You can run `raven setup` from the repository root.
- [ ] You have a known canonical Raven CI ID before running `raven timeline <ci-id>`.
- [ ] You know which guide to open next for AI usage, agent setup, incidents, or next-gen ingest.

## Scope boundary

This documentation describes current Raven behavior and does not add product support by itself. Current AI support claims are limited to the setup and integration surfaces documented in this repository.
