# Raven agent setup plan

Raven instructions should be installed where each AI ecosystem actually reads operational rules. This document tracks the intended insertion points for **project-local setup**. The docs are the source of truth for now; automated `raven setup <agent>` commands can come later.

Current setup scope is safety-first. `raven setup` can create or update approved project-local files and can show user/global guidance, but it must not silently modify global user profiles such as `~/.gemini`, `~/.codex`, Antigravity profiles, or Ollama server environment. Global writes require explicit separate approval and should remain manual unless Raven has a safe supported target.

## Quick path

1. Load the common agent contracts from `.agents/assistant.yaml`, `.agents/policies/tool-policy.yaml`, `.agents/mcp/raven-local.yaml`, and `.agents/mcp/nextgen.yaml` when the AI ecosystem supports structured project instructions.
2. Prefer the MCP server for MCP-compatible agents: `raven mcp`.
3. Read [`docs/ai-usage.md`](ai-usage.md) for the Raven memory rules.
4. Make sure the agent can run the `raven` binary from its shell environment.
5. Test with `raven timeline <ci-id>` or the MCP `raven_list_cis` tool.

## `raven setup` wizard

Run the guided setup flow from the repository root:

```bash
raven setup
```

For command help without launching the TUI:

```bash
raven setup --help
raven setup -h
raven setup help
```

The wizard follows this operator contract:

1. **Detect** supported AI tooling and existing Raven setup files.
2. **Plan** changes before writing anything.
3. **Separate scopes**: project-local writes are distinct from user/global writes.
4. **Ask for approval** before applying the plan; user/global writes require a second approval.
5. **Never write secrets**. Generated files may mention environment variable names such as `NEXTGEN_ACCESS_TOKEN`, but not token values.
6. **Validate** generated artifacts where practical, including JSON parsing and manual smoke-test commands when external tools are absent.
7. **Report rollback identifiers**, including file paths and Raven managed block IDs such as `codex-agents` or `raven-incident-skill`.

Useful smoke checks after setup:

```bash
raven mcp
raven nextgen-mcp        # should show a clear setup error when env vars are missing
ollama show raven-support
bash -n scripts/raven-ollama
```

If setup adds a Raven managed block, remove the matching `BEGIN RAVEN MANAGED` / `END RAVEN MANAGED` block to roll it back manually. If setup creates a whole generated file such as `.gemini/settings.json` or `ollama/Modelfile.raven`, remove that file only after confirming it has no operator-owned content.

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

Use canonical Raven `ci_id` values when already known. If an agent only has an upstream ID, IP, hostname, serial, or MAC address, pass it as a `ci_ref` alias object; upstream IDs are not canonical Raven IDs. next-gen CI IDs are upstream references, not Raven CI IDs, and must be represented as `source=next-gen type=ci_id value=<nextgen_ci_id>` before resolution through Raven aliases.

Raven also exposes a read-only next-gen MCP server for incident enrichment:

```bash
raven nextgen-mcp
```

It requires runtime environment variables, not committed secrets:

```bash
NEXTGEN_BASE_URL=https://nextgen.example.internal
NEXTGEN_ACCESS_TOKEN=<redacted-ai-diagnostic-token>
```

Use `https://` for remote next-gen endpoints. Plain `http://` is accepted only for `localhost` or loopback development URLs.

### Gemini CLI

Project-local MCP config belongs in `.gemini/settings.json`:

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

Add next-gen MCP only when the operator provides environment values outside git-tracked files. Example Gemini project config that reads variables from the agent process environment:

```json
{
  "mcpServers": {
    "raven-nextgen": {
      "command": "bash",
      "args": [
        "-lc",
        "exec env NEXTGEN_BASE_URL=\"$NEXTGEN_BASE_URL\" NEXTGEN_ACCESS_TOKEN=\"$NEXTGEN_ACCESS_TOKEN\" raven nextgen-mcp"
      ]
    }
  }
}
```

Do not replace `$NEXTGEN_ACCESS_TOKEN` with a real token in committed files.

Gemini CLI reads `GEMINI.md` context files by default. For a Raven-enabled repository, add a project `GEMINI.md` or configure `.gemini/settings.json` with `context.fileName` if the repo wants to share another instruction file name such as `AGENTS.md`:

```json
{
  "context": {
    "fileName": ["AGENTS.md", "GEMINI.md"]
  }
}
```

Avoid using Gemini's system prompt override (`GEMINI_SYSTEM_MD`) for the default Raven setup because it can replace the built-in Gemini CLI system instructions entirely. Use normal context files unless an operator intentionally wants a full system override.

### Antigravity

Antigravity preserves Gemini-style workspace context behavior and also recognizes `AGENTS.md`-style workspace rules in current documentation. For project-local Raven setup, prefer shared `AGENTS.md` instructions. Existing Gemini-compatible `GEMINI.md` files can remain in place; if both Gemini and Antigravity must share the same instructions, keep the Raven block in `AGENTS.md` and configure Gemini to load `AGENTS.md` too.

Antigravity MCP configuration is not project-local in the tested CLI. The Raven MCP server must be configured in the user Antigravity profile, for example `~/.gemini/antigravity-cli/mcp_config.json`, or through the Antigravity UI. Do not write `.agents/mcp_config.json` and expect the CLI to load it.

## Instruction insertion points

| Ecosystem | Project instruction/config location | Notes |
| --- | --- | --- |
| Gemini CLI | `GEMINI.md`, optional `.gemini/settings.json` | Gemini loads `GEMINI.md` by default; `context.fileName` can add `AGENTS.md` for shared instructions. Use `.gemini/settings.json` for project MCP config. |
| Antigravity CLI | `AGENTS.md`, `GEMINI.md` | Migration docs say workspace `GEMINI.md` and `AGENTS.md` continue to work. MCP server config is user-profile/UI config, not reliable project-local config. |
| Ollama local models | `ollama/Modelfile.raven`, wrapper script/docs | Ollama does not document AGENTS/GEMINI-style markdown loading. Treat it as a local model backend; inject Raven rules via Modelfile `SYSTEM` and/or wrapper prompt. |
| Codex | `AGENTS.md`, optional `.codex/config.toml` | Codex reads `AGENTS.md` globally and per repo. Project `.codex/config.toml` can add fallback filenames/limits after the project is trusted, but provider definitions belong in user-level config. |
| Gemini proxy | Proxy-side system/developer prompt template | The proxy should inject Raven rules before forwarding user tasks to Gemini. |
| OpenCode | project instructions/plugin/system transform | A future plugin could append Raven protocol to the system prompt. |
| Claude Code | `CLAUDE.md`, plugin skill, or hook-managed prompt | Use a project/user instruction file until a dedicated plugin exists. |
| VS Code/Copilot | Workspace `.instructions.md` | Use workspace custom instructions for Raven command rules. |
| Pi | project docs/skills or package integration | Current Pi session can read these docs directly; future package can inject skills. |

## Minimal instruction block

Use this when no ecosystem-specific integration exists yet:

```text
Use Raven as the local CMDB/timeline tool.
- CI ID is mandatory. Do not invent CI IDs.
- Raven CI IDs are canonical; next-gen CI IDs are upstream references, not Raven CI IDs.
- Resolve upstream refs through Raven aliases before CI-specific reasoning.
- Before diagnosing a known CI, run `raven timeline <ci-id>` when useful.
- Save important diagnostics, observations, maintenance actions, incidents, and resolutions with `raven event capture` only after operator approval.
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

### Codex

Put the minimal instruction block in the repository `AGENTS.md`. Codex loads project `AGENTS.md` files from the repository root down to the current working directory. If the project wants to keep Raven instructions in another file, add a trusted project `.codex/config.toml` fallback:

```toml
project_doc_fallback_filenames = ["RAVEN.md"]
```

Configure Codex MCP servers in user-level `~/.codex/config.toml` or with `codex mcp add`; do not commit operator tokens to this repo. Safe next-gen example:

```bash
codex mcp add raven-nextgen -- bash -lc \
  'exec env NEXTGEN_BASE_URL="$NEXTGEN_BASE_URL" NEXTGEN_ACCESS_TOKEN="$NEXTGEN_ACCESS_TOKEN" raven nextgen-mcp'
```

`NEXTGEN_BASE_URL` must be remote `https://` or localhost/loopback `http://` for local development.

To use local Ollama models from Codex, configure an explicit local provider in user-level `~/.codex/config.toml`. `codex doctor` reports that project-local `model_providers` entries are ignored, so Raven should not install provider definitions into `.codex/config.toml`.

```toml
model_provider = "ollama"
model = "qwen2.5-coder:7b"

[model_providers.ollama]
name = "Ollama"
base_url = "http://localhost:11434/v1"
```

### Ollama Modelfile

Ollama does not load project markdown instructions by itself. Create a project-local Modelfile that bakes the Raven rules into the model system prompt:

```text
FROM qwen3.5:4b
PARAMETER temperature 0.2
PARAMETER num_ctx 8192
SYSTEM """
You are an L1 Incident Assistant for network and endpoint operations.

Raven is not you. Raven is the local MCP/CLI-backed operational memory, CMDB,
CI alias resolver, timeline, and persistence layer.

next-gen is an upstream operational system accessed through a separate MCP proxy.
next-gen CI IDs are upstream refs, not Raven CI IDs.

Use tools for operational context. Do not invent context.

Rules:
- Never invent CI IDs, timeline events, tool results, fixes, or resolutions.
- Resolve upstream refs through Raven before CI-specific reasoning.
- Prefer MCP tools; use CLI only as fallback.
- Use next-gen read-only tools first.
- Persist approved operational memory only in Raven.
- Do not persist anything without explicit operator approval.
- Separate facts, hypotheses, and recommended next checks.
- Do not mutate production or next-gen unless that future capability is explicitly enabled and approved.
- Redact secrets and sensitive payloads.
- Be concise and operational.
- Respond in the operator's language.
"""
```

Build and run the local Raven model:

```bash
ollama create raven-support -f ollama/Modelfile.raven
ollama run raven-support "diagnose FW-MAIN-001"
```

### Ollama wrapper

A wrapper can inject the latest Raven instructions without rebuilding an Ollama model. Keep the wrapper project-local and pass the prompt safely.

Linux/macOS or WSL shell:

```bash
scripts/raven-ollama FW-MAIN-001 diagnose packet loss
```

Windows PowerShell:

```powershell
.\scripts\raven-ollama.ps1 FW-MAIN-001 diagnose packet loss
```

Windows Command Prompt:

```cmd
scripts\raven-ollama.cmd FW-MAIN-001 diagnose packet loss
```

The wrappers should:

1. Read the project Raven instruction block.
2. Run `raven timeline <ci-id>` when a CI ID is provided.
3. Call `ollama run <model>` or the Ollama HTTP API with the Raven rules in the prompt/system message.
4. Ask before recording an event unless the caller explicitly requests capture.

Environment:

| Variable | Purpose |
| --- | --- |
| `RAVEN_OLLAMA_MODEL` | Ollama model to run, default `raven-support`. |
| `OLLAMA_HOST` | Optional Ollama server address honored by the Ollama CLI, for example `http://127.0.0.1:11434`. |

> Avoid piping untrusted model output directly into shell command substitution. Capture model output to a temporary file or variable with proper quoting before calling `raven event capture`.

#### Windows and WSL interoperability

Ollama for Windows serves the API on `localhost:11434` by default and normally binds to `127.0.0.1:11434`. Native Windows wrappers can use that default. Linux/WSL callers are different:

- With WSL mirrored networking on supported Windows 11 builds, WSL can usually reach Windows Ollama through `127.0.0.1:11434`.
- With default WSL2 NAT networking, WSL should call the Windows host IP instead of assuming WSL `localhost` reaches Windows services.
- If exposing Ollama beyond loopback is required, set `OLLAMA_HOST` for the Ollama server and restart Ollama, then restrict access with Windows/Hyper-V firewall rules. Ollama's local API is unauthenticated by default, so do not bind it broadly on untrusted networks.

Example WSL NAT client setup:

```bash
export OLLAMA_HOST="http://$(ip route show default | awk '{print $3}'):11434"
scripts/raven-ollama FW-MAIN-001 diagnose packet loss
```

### next-gen adapter

Read-only incident enrichment uses:

```bash
NEXTGEN_BASE_URL=https://nextgen.example.internal \
NEXTGEN_ACCESS_TOKEN=<redacted-ai-diagnostic-token> \
raven nextgen-mcp
```

Persist only selected, redacted facts into Raven:

```bash
raven event ingest --source next-gen --file /tmp/next-gen-alert.json
```

## Future setup commands

Potential automation:

```bash
raven setup gemini-cli
raven setup antigravity
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
