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

<!-- BEGIN RAVEN MANAGED: codex-agents -->
Use Raven as the local CMDB and operational timeline.
- CI ID is mandatory. Do not invent CI IDs.
- Raven CI IDs are canonical; next-gen CI IDs are upstream references and must be resolved through Raven aliases first.
- Before diagnosing a known CI, inspect prior context with `raven timeline <ci-id>` when useful.
- Prefer `raven mcp`; use Raven CLI commands only as fallback.
- Never write secrets or access tokens into project files.
- Capture important diagnostics or resolutions in Raven only after operator approval.
<!-- END RAVEN MANAGED: codex-agents -->

<!-- BEGIN RAVEN MANAGED: raven-local-ai-guidance -->
This setup slice focuses on project-local AI guidance only.
- Raven MCP is the shared operational surface; configure `raven mcp` and MCP clients from the project.
- For Gemini CLI, setup `AGENTS.md` plus project `.gemini/settings.json` MCP settings.
- For Ollama, setup writes local `ollama/Modelfile.raven` and optional wrapper scripts.
- For Codex and other tools, keep provider-level runtime settings in user-global profiles; this wizard only writes repository-local guidance.
- After plan/apply, run the local smoke checks shown in the validation summary.
<!-- END RAVEN MANAGED: raven-local-ai-guidance -->
