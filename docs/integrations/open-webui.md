# Raven MCP for Open WebUI

> Outcome: connect Open WebUI to Raven's MCP server so an AI agent can read the CI timeline, resolve upstream identifiers to canonical `ci_id` values, and record diagnostic / incident / resolution events against a stable local memory.

This guide is written for an AI agent executing the wiring end-to-end. It assumes a human operator is available to approve setup-time writes but not to drive each command.

## Quick path

1. Confirm the host satisfies the preconditions.
2. Build the Raven binary.
3. Register the Raven MCP server in Open WebUI's admin panel.
4. Verify the agent can list tools, resolve a known alias, and record one event.
5. Read the operational patterns before the first real run.

## Preconditions

| Requirement | How to verify | Fallback |
|---|---|---|
| Go 1.26.3 or newer | `go version` | Install from `https://go.dev/dl/` |
| Raven repository cloned | `ls cmd/raven/main.go` | `git clone <raven-repo>` |
| Open WebUI running and reachable | `curl -fsS http://localhost:3000/api/health` | Follow `https://docs.openwebui.com/getting-started/` |
| Open WebUI admin access | Login as admin user | Required to register the MCP server |
| A canonical Raven CI ID available | `raven ci list` or operator input | Do not invent IDs. See [Identity discipline](#identity-discipline) |
| An existing alias to resolve | `raven alias list` | Add one via `raven alias add` or stop and ask the operator |

## Step 1 — Build and verify Raven

```bash
cd <raven-repo>
go build -o raven ./cmd/raven
./raven version
./raven --help
go test ./...
```

Expected: version string prints, `--help` lists `ci`, `alias`, `event`, `timeline`, `mcp`, `nextgen-mcp`, tests pass.

If `go test ./...` fails, **stop**. Do not register an unverified binary with Open WebUI.

## Step 2 — Confirm local storage paths

Raven writes JSON files under a `raven/` subdirectory of the OS user config dir. The actual path is platform-dependent:

| OS | Storage directory |
|---|---|
| macOS | `~/Library/Application Support/raven/` |
| Linux | `${XDG_CONFIG_HOME:-$HOME/.config}/raven/` |
| Windows | `%AppData%\raven\` |

Resolve the path the agent will reference. Cache the resolved value and reuse it in every later step:

> **Override precedence** (see issue #28): `--data-dir <path>` global CLI flag wins, otherwise the `RAVEN_DATA_DIR` environment variable is used, otherwise Raven falls back to the OS user config directory (`~/.config/raven/` on Linux, `~/Library/Application Support/raven/` on macOS, `%AppData%\raven\` on Windows). When both are set, the flag wins. Whitespace-only values are treated as unset.

```bash
# Detect once, reuse everywhere
case "$(uname -s)" in
  Darwin) RAVEN_DATA_DIR="$HOME/Library/Application Support/raven" ;;
  Linux)  RAVEN_DATA_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/raven" ;;
  *)      echo "Unsupported OS — derive from Go's os.UserConfigDir()" >&2; exit 1 ;;
esac
export RAVEN_DATA_DIR

# Trigger lazy directory creation
./raven alias list

# Verify
ls -la "$RAVEN_DATA_DIR"
```

```powershell
# Windows (PowerShell) — equivalent of the bash block above
$env:RAVEN_DATA_DIR = Join-Path $env:APPDATA "raven"
# Trigger lazy directory creation
.\raven.exe alias list
# Verify
Get-ChildItem $env:RAVEN_DATA_DIR
```

Expected files after first use: `components.json`, `events.json`, `aliases.json`. The directory is created lazily on first write. If the directory does not exist yet, that is normal — `raven alias list` creates it.

All subsequent steps in this guide use `$RAVEN_DATA_DIR` to refer to the resolved path.

## Step 3 — Pre-seed at least one CI and one alias

Raven will not accept events for an unknown CI. Before connecting Open WebUI, the operator must seed:

```bash
./raven ci add --ci-id RAVEN-FW-MAIN-001 --notes "Main firewall"
./raven alias add \
  --ci-id RAVEN-FW-MAIN-001 \
  --source next-gen \
  --type ci_id \
  --value 42
./raven alias list
```

If the operator cannot provide a canonical `ci_id` and an upstream reference, **stop and ask**. Never invent identifiers.

## Step 4 — Register the MCP server in Open WebUI

In the Open WebUI admin panel, navigate to the MCP server configuration surface (commonly **Settings → Tools → MCP Servers** or **Settings → Connections → Tool Servers**, depending on the Open WebUI version) and add a new server with the following fields:

| Field | Value |
|---|---|
| Name | `raven` |
| Command | absolute path to the built binary, e.g. `/Users/<user>/raven/raven` |
| Args | `["mcp"]` |
| Environment | empty unless operator specifies overrides |
| Type | `stdio` |
| Auto-load on new chat | `true` (recommended) |

3. Save and trigger Open WebUI to reload tools.

Do not register `raven nextgen-mcp` unless the operator explicitly requires nextgen API access. The local memory works without it.

## Step 5 — Verify the integration

Open a new chat in Open WebUI and ask the agent to perform each of the following. Each step must succeed before proceeding to the next.

| # | Ask the agent to | Expected result |
|---|---|---|
| 1 | "List the available Raven tools" | Seven tools reported: `resolve_ci_ref`, `record_event`, `get_timeline`, `list_cis`, `get_ci`, `get_ci_metadata`, `set_ci_metadata` |
| 2 | "Resolve the alias next-gen ci_id 42" | Returns the canonical `ci_id` registered in Step 3 |
| 3 | "Show the timeline for that CI" | Returns an empty array or existing events |
| 4 | "Record an observation that this verification ran" | Returns the persisted event; `$RAVEN_DATA_DIR/events.json` updated on disk |

If step 1 fails, the MCP server did not register correctly. Return to Step 4.

If step 2 returns `unknown alias`, the seed in Step 3 was skipped. Return to Step 3.

If step 4 does not produce a new entry in `$RAVEN_DATA_DIR/events.json`, the agent may have called the wrong tool or omitted a required parameter. Re-read [Tool reference](#tool-reference) and retry once.

## Identity discipline

Raven distinguishes between canonical `ci_id` and upstream references. This is not a stylistic choice — it is enforced at write time.

| Identifier shape | Use it with | Never use it with |
|---|---|---|
| Canonical `ci_id` (e.g. `RAVEN-FW-MAIN-001`) | `record_event` when the agent already knows it from a prior lookup or operator input | Anything that was not created via `raven ci add` |
| Upstream ID, IP, hostname, serial, MAC | Wrap as `ci_ref` object: `{"source": "<source>", "type": "<type>", "value": "<value>"}` and pass to `record_event` or `resolve_ci_ref` | As a literal `ci_id` in `record_event` |

When in doubt: resolve first, then write.

```text
upstream reference
  → resolve_ci_ref
  → canonical ci_id (or rejection if unknown)
  → record_event with ci_id
```

Do not skip the resolve step even if the agent "thinks" the ID is canonical. Raven will reject unknown `ci_id` values and that rejection is the signal to resolve first.

## Tool reference

### `resolve_ci_ref`

Purpose: map an upstream identifier to a canonical Raven `ci_id`.
Inputs:

| Parameter | Type | Required | Notes |
|---|---|---|---|
| `source` | string | yes | Alias source namespace, e.g. `next-gen`, `prometheus`, `human` |
| `type` | string enum | yes | One of: `ci_id`, `ip`, `hostname`, `serial`, `mac` |
| `value` | string | yes | Exact value after whitespace trim |

Returns: `{ "ci_id": "<canonical>" }` or an error if no alias matches.
Idempotent. Read-only.

### `record_event`

Purpose: append one event to a CI's timeline.
Inputs (all required unless marked optional):

| Parameter | Type | Required | Notes |
|---|---|---|---|
| `ci_id` | string | conditional | Canonical Raven CI ID. Provide when known. |
| `ci_ref` | object | conditional | `{source, type, value}` to resolve upstream. Provide when `ci_id` is unknown. |
| `type` | string | yes | One of: `observation`, `diagnosis`, `network_alert`, `incident`, `resolution`, or any operator-approved value |
| `severity` | string | yes | One of: `info`, `warning`, `critical` (free-form allowed) |
| `summary` | string | yes | Short, operator-readable, single sentence |
| `source` | string | yes | Producer name. Use the agent's stable identity, e.g. `open-webui`, `claude-desktop`, `next-gen`, `human` |
| `observed_at` | string (RFC3339) | yes | When the event was observed, not when it was recorded |
| `external_id` | string | conditional | Stable source event ID. Required if `dedup_key` is omitted. |
| `dedup_key` | string | conditional | Stable replay-prevention key. Required if `external_id` is omitted. |
| `id` | string | optional | Raven generates one when omitted |
| `status` | string | optional | Defaults to `open`. Use `closed` for resolutions. |
| `details` | string | optional | Longer diagnostic text |
| `ingested_at` | string | optional | Raven fills when omitted |
| `raw` | string | optional | Raw source evidence |

Exactly one of `ci_id` or `ci_ref` must be present. Exactly one of `external_id` or `dedup_key` must be present.
Not idempotent. Use `dedup_key` to make replays safe.

### `get_timeline`

Purpose: read all events for one CI in chronological order.
Inputs:

| Parameter | Type | Required | Notes |
|---|---|---|---|
| `ci_id` | string | yes | Canonical Raven CI ID |

Returns: `{ "ci_id": "...", "events": [...] }`. Read-only, idempotent.

### `list_cis`

Purpose: list every known CI.
Inputs:

| Parameter | Type | Required | Default | Notes |
|---|---|---|---|---|
| `category` | string | no | `""` | Exact-match filter on `Component.Category` (for example `network`, `hardware`). Empty means no filter. |
| `prefix` | string | no | `""` | Case-sensitive prefix filter on `ci_id` (for example `TWR-`). Empty means no filter. |
| `query` | string | no | `""` | Case-insensitive substring search across `ci_id`, `model`, and `notes`. Empty means no filter. |
| `limit` | number | no | `0` | Maximum number of CIs to return after filtering. `0` or negative means no cap. |

Returns: `{ "cis": [...] }`. Read-only, idempotent. Filters compose with AND semantics.

### `get_ci`

Purpose: fetch one CI by canonical ID.
Inputs:

| Parameter | Type | Required | Notes |
|---|---|---|---|
| `ci_id` | string | yes | Canonical Raven CI ID |

Returns: `{ "ci": {...} }`. Read-only, idempotent.

### `get_ci_metadata`

Purpose: read the metadata sidecar entry for one canonical Raven CI. Returns an empty entry (no error) when the CI has no metadata yet.
Inputs:

| Parameter | Type | Required | Notes |
|---|---|---|---|
| `ci_id` | string | yes | Canonical Raven CI ID |

Returns: `{ "ci_id": "...", "attributes": {...}, "relationships": [...] }`. Read-only, idempotent.

### `set_ci_metadata`

Purpose: upsert a metadata sidecar entry for one canonical Raven CI. Each field uses REPLACE semantics: omit a field to preserve it, supply an empty object/array to clear it, supply a non-empty value to set it.
Inputs:

| Parameter | Type | Required | Notes |
|---|---|---|---|
| `ci_id` | string | yes | Canonical Raven CI ID |
| `attributes` | object map | optional | Map of attribute name to TypedValue (`{"string": "x"}`, `{"number": 45}`, `{"bool": true}`, `{"enum": "x"}`). Omit to preserve. Empty object clears all attributes. |
| `relationships` | array of objects | optional | Array of `{target_ci_id, kind}` pairs. Replaces all existing relationships for the CI when supplied. Self-referential relationships (target equals `ci_id`) are rejected. |

Returns: the resulting sidecar entry. Idempotent (replace semantics).

## Operational patterns

### Diagnosis → record → resolve

```text
1. Operator reports an issue with an upstream identifier (IP, hostname, next-gen ci_id).
2. Agent calls resolve_ci_ref with the upstream identifier.
   - On success → continue with returned canonical ci_id.
   - On failure → stop. Tell the operator the upstream identifier is unknown to Raven and ask for a canonical ci_id or an alias to add.
3. Agent calls get_timeline with the canonical ci_id to inspect prior context.
4. Agent performs diagnostic work (using its own tools, NOT Raven).
5. Agent calls record_event with:
   - ci_id (already resolved)
   - type: "diagnosis"
   - severity: matches the finding
   - summary: one-sentence operator-readable description
   - source: stable agent identity
   - observed_at: current RFC3339 timestamp
   - external_id or dedup_key: stable across retries
6. After resolution, agent calls record_event again with:
   - same ci_id
   - type: "resolution"
   - status: "closed"
   - summary describing the fix
   - dedup_key derived from the diagnosis external_id (e.g. "<diagnosis-external-id>:resolution")
7. Optionally call get_timeline to confirm both events are now visible.
```

Never delete or edit the diagnosis event when posting the resolution. The timeline is append-only. Corrections are new events that reference the prior `external_id` or `dedup_key`.

### Recording events from the CLI instead of MCP

When the operator is at a terminal and not running an agent, two CLI commands cover the same write paths as `record_event`:

| Command | Use when | Required args | Notable fields |
|---|---|---|---|
| `raven event add <ci-id>` | The agent has full structured data and needs `--external-id` for cross-system dedup | CI ID positional; `--type`, `--severity`, `--summary`, `--source` required | Supports `--external-id` and `--raw` |
| `raven event capture <ci-id>` | The agent or operator has free-form text describing the event | CI ID positional; `--text` required; `--source` required | Auto-derives `summary` from the first line of `--text`; no `external_id`, so dedup is per-host random |

Prefer `raven event add` whenever cross-system replay prevention matters. Prefer `raven event capture` for quick operator notes.

### Searching across CIs

Raven does not currently support free-text event search. Two acceptable alternatives:

1. List CIs via `list_cis`, then call `get_timeline` on each candidate. Use this when the search space is small.
2. Filter events client-side after `get_timeline` returns the full timeline of a known CI.

If a query is unanswerable with these patterns, stop and tell the operator that Raven lacks cross-CI search.

## Failure modes and recovery

| Symptom | Likely cause | Recovery |
|---|---|---|
| `unknown alias` from `resolve_ci_ref` | The alias was never registered, or source/type/value do not match exactly | Confirm with `raven alias list`; re-register if missing |
| `unknown ci_id` from `record_event` | Agent passed an upstream ID as `ci_id` instead of `ci_ref` | Re-call with `ci_ref`; do not retry with the same argument |
| Tool not appearing in Open WebUI | MCP server not registered or reload not triggered | Return to Step 4; verify command path and args |
| Tool appears but fails immediately | Binary not executable, or wrong working directory for `$RAVEN_DATA_DIR` | `chmod +x <raven-binary>`; verify storage path |
| `events.json` not updating on disk | Storage path mismatch or read-only mount | Confirm path via `ls -la "$RAVEN_DATA_DIR"` |
| Concurrent writes from multiple agents | JSON storage is not concurrency-safe | Serialize writes; limit to one writer process per host |

## What NOT to do

| Anti-pattern | Why it fails |
|---|---|
| Inventing a canonical `ci_id` | Rejected at write time; corrupts downstream references |
| Passing an upstream ID as `ci_id` | Same — use `ci_ref` |
| Deleting or rewriting past events | Timeline is append-only by design |
| Calling `record_event` without `external_id` or `dedup_key` | Rejected at write time |
| Concurrent writes from multiple processes | JSON files have no transactional safety |
| Storing secrets in `details` or `raw` fields | Raven is a timeline, not a vault — redact before write |
| Treating Raven as the source of truth for live state | Raven is for facts about CIs over time. Live state belongs to monitoring systems |
| Relying on cross-CI free-text search | Not implemented. Use the two patterns in [Searching across CIs](#searching-across-cis) |

## Current limitations

These are project-level constraints, not bugs. Plan around them.

- Storage is JSON files under the OS user config directory (`$RAVEN_DATA_DIR`). SQLite migration is on the roadmap but not implemented.
- No update or delete operations on events.
- No free-text search across events; only per-CI timeline reads.
- No concurrent-write safety. One writer process per host.
- Open WebUI's own persistent memory (per-user facts across chats) is separate from Raven. Do not confuse the two layers.

## Verification checklist

Before handing the integration back to the operator, confirm:

- [ ] `raven version` prints a version string
- [ ] `go test ./...` passes
- [ ] At least one canonical `ci_id` exists in `$RAVEN_DATA_DIR/components.json`
- [ ] At least one alias exists in `$RAVEN_DATA_DIR/aliases.json` and resolves correctly
- [ ] The MCP server is registered in Open WebUI's admin panel
- [ ] The agent can list the seven MCP tools in a new chat
- [ ] `resolve_ci_ref` returns the expected `ci_id`
- [ ] `record_event` appends a new event visible in `get_timeline`
- [ ] `$RAVEN_DATA_DIR/events.json` reflects the new entry
- [ ] The operator has been told the JSON storage caveat and the append-only constraint

## Next step

After verification, point the operator at:

- `docs/ai-usage.md` — full memory contract for agents and adapters.
- `docs/agent-setup.md` — what the `raven setup` wizard does and does not touch.
- `docs/design/raven-incident-workflow.md` — incident intake pattern built on these tools.
- The `raven-incident` skill at `.agents/skills/raven-incident/SKILL.md` for incident-specific workflows.
