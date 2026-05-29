# Raven agent setup plan

Raven instructions should be installed where each AI ecosystem actually reads operational rules. This document tracks the intended insertion points. The docs are the source of truth for now; automated `raven setup <agent>` commands can come later.

## Quick path

1. Prefer the MCP server for MCP-compatible agents: `raven mcp`.
2. Read [`docs/ai-usage.md`](ai-usage.md) for the Raven memory rules.
3. Make sure the agent can run the `raven` binary from its shell environment.
4. Test with `raven timeline <ci-id>` or the MCP `raven_list_cis` tool.

## MCP setup

Raven exposes an agent-facing stdio MCP server:

```bash
raven mcp
```

The server provides these tools:

- `raven_resolve_ci_ref`
- `raven_record_event`
- `raven_get_timeline`
- `raven_list_cis`
- `raven_get_ci`

Use canonical Raven `ci_id` values when already known. If an agent only has an upstream ID, IP, hostname, serial, or MAC address, pass it as a `ci_ref` alias object; upstream IDs are not canonical Raven IDs.

### Gemini CLI

Add Raven to `~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "raven": {
      "command": "raven",
      "args": ["mcp"]
    }
  }
}
```

If `raven` is not on the agent's `PATH`, use the absolute path to the binary in `command`.

### Antigravity

Add Raven to Antigravity's raw MCP config, for example `~/.gemini/antigravity-cli/mcp_config.json` or the equivalent **Manage MCP Servers** raw config UI:

```json
{
  "mcpServers": {
    "raven": {
      "command": "raven",
      "args": ["mcp"]
    }
  }
}
```

## Instruction insertion points

| Ecosystem | Suggested instruction location | Notes |
| --- | --- | --- |
| Gemini CLI | `~/.gemini/system.md` | Gemini must be configured to load system markdown, e.g. `GEMINI_SYSTEM_MD=1`, following the same pattern Engram uses. |
| Gemini proxy | Proxy-side system/developer prompt template | The proxy should inject Raven rules before forwarding user tasks to Gemini. |
| Ollama local agents | Wrapper system prompt or Modelfile/template | Prefer a wrapper that injects Raven rules and validates generated commands. |
| Codex | `~/.codex/raven-instructions.md` or configured instruction file | Mirror the agent-specific file pattern used by other tools. |
| OpenCode | plugin/system transform or global instructions | A future plugin could append Raven protocol to the system prompt. |
| Claude Code | `CLAUDE.md`, plugin skill, or hook-managed prompt | Use a project/user instruction file until a dedicated plugin exists. |
| VS Code/Copilot | User prompts `.instructions.md` | Use workspace/user custom instructions for Raven command rules. |
| Pi | project docs/skills or package integration | Current Pi session can read these docs directly; future package can inject skills. |

## Minimal instruction block

Use this when no ecosystem-specific integration exists yet:

```text
Use Raven as the local CMDB/timeline tool.
- CI ID is mandatory. Do not invent CI IDs.
- Before diagnosing a known CI, run `raven timeline <ci-id>` when useful.
- Save important diagnostics, observations, maintenance actions, incidents, and resolutions with `raven event capture`.
- Use `raven event ingest --source <source> --file <json>` only when you have normalized event data with `external_id` or `dedup_key`.
- Prefer `event capture` over losing context when structured JSON is too costly.
- Preserve source/evidence and keep summaries short.
```

## Agent command examples

### Gemini CLI

```bash
raven event capture RAVEN-DEV-001 \
  --source gemini-cli \
  --type diagnosis \
  --severity info \
  --text "Gemini diagnosed packet loss symptoms and recommended checking the WAN link."
```

### Ollama wrapper

```bash
ollama run support-diagnoser "diagnose FW-MAIN-001" \
  | raven event capture FW-MAIN-001 --source ollama --text "$(cat)"
```

> The exact wrapper should avoid unsafe shell interpolation; this is a conceptual example.

### next-gen adapter

```bash
raven event ingest --source next-gen --file /tmp/next-gen-alert.json
```

## Future setup commands

Potential automation:

```bash
raven setup gemini-cli
raven setup codex
raven setup opencode
raven setup ollama
```

Each setup command should:

1. Locate the agent instruction file/config.
2. Insert or update a clearly marked Raven block.
3. Avoid overwriting user content.
4. Verify `raven` is available on `PATH`.
5. Print a smoke-test command.

## Design principle

Keep adapters thin. Raven's validation belongs in Raven commands/storage. Agent-specific prompts should teach when to call Raven, not duplicate Raven's business rules.
