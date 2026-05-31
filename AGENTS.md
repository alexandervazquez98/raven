# Raven project instructions

Raven is the local CMDB and operational timeline for configuration items (CIs).

## Raven rules

- CI ID is mandatory. Do not invent CI IDs.
- If an upstream identifier is provided, resolve it through Raven aliases before treating it as canonical:
  - `raven alias resolve --source <source> --type <ci_id|ip|hostname|serial|mac> --value <value>`
- Before diagnosing a known CI, inspect prior context when useful:
  - `raven timeline <ci-id>`
- Save important diagnostics, observations, maintenance actions, incidents, and resolutions:
  - `raven event capture <ci-id> --source <agent> --type <type> --severity <severity> --text "..."`
- Use `raven event ingest --source <source> --file <json>` only for normalized event JSON with `external_id` or `dedup_key`.
- Preserve source evidence. Keep summaries short and operator-readable.
- Prefer Raven MCP tools when available; otherwise use the Raven CLI.

## Agent source names

Use these `--source` values when recording events:

- Gemini CLI: `gemini-cli`
- Antigravity CLI: `antigravity`
- Codex: `codex`
- Ollama local model/wrapper: `ollama`
- Human/operator notes: `human`

## Project skills

- Use `.agents/skills/raven-incident/SKILL.md` when the user reports an operational incident, alert, CI problem, IP/hostname reference, next-gen event, diagnosis, repair, or resolution workflow.

## Local validation

- Run `go test ./...` after changing Go code.
- For setup/config-only changes, validate JSON/TOML/shell syntax where possible.
