# Raven AI usage contract

Raven is the CMDB and operational timeline for CIs. AI tools such as Gemini CLI, local Ollama agents, or a future `next-gen` adapter should use Raven to preserve important operational context instead of leaving it in transient chats, terminals, or logs.

## Quick path

Use the simplest Raven surface that fits the producer. The shared operational contracts live under `.agents/`: `assistant.yaml`, `policies/tool-policy.yaml`, `mcp/raven-local.yaml`, `mcp/nextgen.yaml`, and `skills/raven-incident/SKILL.md`.

| Situation | Use |
| --- | --- |
| MCP-compatible AI agent | `raven mcp` and the `raven_*` MCP tools |
| Gemini CLI or Antigravity project setup | Project MCP config plus `GEMINI.md`/`AGENTS.md` Raven instructions |
| Codex project setup | Repository `AGENTS.md` Raven instructions; optional trusted `.codex/config.toml` for project doc-loading settings |
| Ollama local model | Project Modelfile `SYSTEM` prompt or wrapper that injects Raven rules; Ollama does not read project `.md` instructions itself |
| Human or AI has freeform text | `raven event capture <ci-id> --source <agent> --text "..."` |
| Adapter already has normalized event JSON | `raven event ingest --source <system> --file alert.json` |
| Need to create the CI first | `raven ci add --ci-id ... --category ... --model ...` |
| Need prior context | `raven timeline <ci-id>` or MCP `raven_get_timeline` |
| Need to configure local AI integrations | `raven setup` from the repository root, then review/apply the setup plan |

## Core rules

1. **CI ID is the topic.** Every Raven CI and event is anchored to canonical Raven `ci_id`.
2. **Do not invent CI IDs.** If the CI is unknown, ask the user or resolve through aliases when that feature exists.
3. **next-gen IDs are upstream refs.** next-gen CI IDs are not Raven CI IDs; represent them as `source=next-gen type=ci_id value=<nextgen_ci_id>` and resolve through Raven before CI-specific reasoning.
4. **Categories are flexible.** `hardware`, `logical`, `network`, `power`, `service`, `database`, `firewall`, and other CMDB labels are valid when non-empty.
5. **Preserve source.** Always set `--source` to the producer, such as `gemini-cli`, `ollama`, `next-gen`, `human`, or a proxy name.
6. **Capture decisions and diagnostics with approval.** If an AI diagnosis, operational finding, maintenance action, or resolution would help future support, propose recording it and ask before writing.
7. **Separate summary from evidence.** The summary may be AI-generated, but raw/source evidence should be preserved in details or normalized event fields when available.
8. **Prefer capture before silence.** If structured ingest is too hard, use `event capture` with clear text after approval.

## MCP tools

Start the stdio MCP server with:

```bash
raven mcp
```

Initial tools:

| Tool | Purpose |
| --- | --- |
| `raven_resolve_ci_ref` | Resolve `source + type + value` to canonical Raven `ci_id`. |
| `raven_record_event` | Record an event with either canonical `ci_id` or a `ci_ref` alias object. |
| `raven_get_timeline` | Read timeline events for a canonical CI. |
| `raven_list_cis` | List known CIs. |
| `raven_get_ci` | Read one CI by canonical ID. |

`raven_record_event` follows the same identity rule as CLI ingest: use `ci_id` only when it is already a Raven canonical ID; otherwise pass upstream identifiers as `ci_ref`.

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

Required normalized fields for adapter ingest include either Raven's canonical `ci_id` or an explicit `ci_ref` alias reference:

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
  "external_id": "ng-98765",
  "observed_at": "2026-05-28T21:00:00Z"
}
```

If `ci_id` is present, Raven uses it directly. If `ci_id` is absent and `ci_ref` is present, Raven resolves `source + type + value` through aliases before storing the event.

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
| `ci_id` | Stable Raven topic/CI identity. Use when already known. |
| `ci_ref` | Optional alias lookup object (`source`, `type`, `value`) for ingest payloads that do not yet know canonical `ci_id`. |
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
Before recording an event, identify the canonical Raven CI ID. Do not invent one. next-gen CI IDs are upstream references, not Raven CI IDs. Represent next-gen CI IDs as source=next-gen type=ci_id value=<nextgen_ci_id> and resolve them before CI-specific Raven reasoning. If you have an upstream ID or operational identifier, either include it as `ci_ref` in normalized ingest JSON or resolve it through aliases first:
  raven alias resolve --source <source> --type <ci_id|ip|hostname|serial|mac> --value <value>
If you only have freeform diagnostic text, use:
  raven event capture <ci-id> --source <your-agent-name> --type <type> --severity <severity> --text "..."
If you have normalized event data with a source event ID, use either:
  raven event ingest --source <source> --file <file>
or pipe JSON directly:
  producer-command | raven event ingest --source <source> --stdin
Preserve evidence. Keep summaries short. Use the CI timeline before making historical claims.
```

## Alias/reference resolution

Raven owns canonical `ci_id` values. Upstream identifiers such as `next-gen` CI IDs are references, not Raven identity. Store them as aliases before relying on them in adapters:

```bash
raven alias add --ci-id RAVEN-FW-MAIN-001 --source next-gen --type ci_id --value 42
raven alias add --ci-id RAVEN-FW-MAIN-001 --source next-gen --type hostname --value fw-main
raven alias list
raven alias resolve --source next-gen --type ci_id --value 42
```

Aliases are stored in `~/.config/raven/aliases.json`. The unique key is `source + type + value`; Raven rejects unknown canonical CIs and duplicate or conflicting mappings. Alias values are exact-match after trimming whitespace, so adapters should pass a consistent hostname, MAC, IP, serial, or upstream ID format.

## Current limitations

- Raven does not create unresolved events yet; ingest fails if neither `ci_id` nor a resolvable `ci_ref` identifies the CI.
- SQLite is not implemented yet; Raven currently stores local JSON files under the user config directory.
- Ollama is only a local model runtime in this contract. Raven must provide project Modelfiles, wrappers, or client configuration for instruction injection.
- Windows Ollama can be used from native PowerShell/CMD wrappers; WSL/Linux clients may need `OLLAMA_HOST` pointed at the Windows host IP unless WSL mirrored networking makes `127.0.0.1:11434` work.
- `raven setup` is available for guided AI integration setup, but the TUI is intentionally safety-first and minimal.
- Project-local setup should not silently edit global AI tool profiles; provider definitions and secrets stay in user-level tool configuration unless the operator explicitly approves a supported user-global write.

## Next steps

1. Add a dedicated next-gen adapter command or script that emits normalized `ci_ref` event payloads.
2. Polish ecosystem-specific setup templates as Gemini CLI, Antigravity CLI, Codex, and Ollama config formats evolve.
3. Add non-interactive `raven setup plan/apply/validate` subcommands if operators need scripted setup.
4. Migrate storage to SQLite after CIs, events, aliases, and ingest contracts stabilize.
