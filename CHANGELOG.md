# Changelog

All notable changes to Raven are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

> **Note**: Raven is in an early testing phase (`0.x`). Minor versions may include
> breaking changes; the project will stabilize at `1.0.0` when the public surface
> is frozen.

## [Unreleased]

## [0.4.0] - 2026-09-20

### Added

- **Metadata sidecar at `~/.config/raven/metadata.json` (issue #23)** — optional, opt-in sidecar for typed attributes and topological relationships per CI. Keeps `domain.Component` immutable; CIs that don't need metadata never see or pay for it. Implements Option D as approved in the maintainer consensus comment. Verified end-to-end against the NetOps Orchestrator LLM (Qwen 3.5 9B + NORA RF telemetry on Cambium PMP 450i): 5/5 multi-turn reasoning gates PASS across all four slices.

  - **Slice 1 (PR #26)**: domain types in `internal/domain/metadata.go` — `MetadataSidecar`, `CIMetadataEntry`, `TypedValue` (sum type: `string | number | bool | enum`), `CIRelationship`; `Validate()` methods with sentinel errors (`ErrInvalidTypedValue`, `ErrMissingAttributeKey`, `ErrMissingRelationshipTargetCIID`, `ErrMissingRelationshipKind`, `ErrSelfReferentialRelationship`, `ErrUnsupportedSidecarVersion`); reuses shared `ErrDuplicateCIID`. Public constructors `StringValue`/`NumberValue`/`BoolValue`/`EnumValue` and `Raw() any` accessor on `TypedValue`. Per-entry validation; `ci_id` uniqueness across the sidecar; rejection of self-referential relationships.

  - **Slice 2 (PR #29)**: storage helpers `SaveMetadata` and `LoadMetadata` in `internal/storage/metadata.go`, mirroring `SaveComponents`/`LoadComponents`. Snake_case JSON with `0o600` perms, `0o755` parent dirs, `MetadataSidecarVersion = 1` schema version. Missing file returns an empty sidecar pinned to the current version. 10 unit tests covering roundtrip, parent-dir creation, tag discipline, missing file, invalid JSON, validation, unsupported version, duplicate ci_id, self-referential relationship.

  - **Slice 3 (PR #30)**: CLI commands — `raven metadata add --ci-id <id> [--attribute "k=v,k=v"] [--relationship "target=kind,target=kind"]` (upsert with MERGE semantics; attribute keys overwrite by key, relationships append), `raven metadata list [--limit N]` (tabular summary), `raven metadata show <ci-id>` (human-readable dump). Attribute values auto-type: `true`/`false` → BoolValue, parseable as float → NumberValue, else StringValue. New `app.MetadataPath` and a `loadMetadata` helper in `internal/cli/cli.go`. 14 tests + 6 sub-tests.

  - **Slice 4 (PR #31)**: MCP tools — `raven_get_ci_metadata(ci_id)` (read-only, idempotent; returns empty entry when no metadata exists) and `raven_set_ci_metadata(ci_id, attributes?, relationships?)` (destructive, idempotent; REPLACE semantics — omit preserves, empty clears, non-empty sets). Service layer gains `GetCIMetadata` and `SetCIMetadata` methods using `*map`/`*slice` pointer contracts to distinguish "not provided" from "provided as empty". 22 new tests + 1 extended.

## [0.3.0] - 2026-09-20

### Added

- **Optional filtering on `raven ci list` and `raven_list_cis` (issue #24 / PR #25)** —
  CLI command `raven ci list` gains four optional flags: `--category` (exact
  match), `--prefix` (case-sensitive prefix on `ci_id`), `--query`
  (case-insensitive substring across `ci_id`, `model`, and `notes`), and
  `--limit` (cap results after filtering). The MCP `raven_list_cis` tool
  schema gains the same four optional parameters. Backed by the new
  `service.ListFilter` struct and `Service.ListCIsWithFilter` method; the
  existing `Service.ListCIs` is preserved. Filters compose with AND. Empty
  filter values mean "no filter". Default no-arg behavior is byte-identical
  to v0.2.0.

## [0.2.0] - 2026-09-12

### Added

- **Setup AI integrations slice (issue #15)** — `raven setup` now includes the
  `raven-local-ai-guidance` slice that writes a managed block into `AGENTS.md`
  covering Gemini CLI, Ollama, and Codex project-local guidance. Hardened
  validation detects malformed, duplicate, and stale managed blocks via the
  new `validateManagedFilePresent` and `validateManagedBlockPresent` paths.
  `ActionManual` items surface as `ValidationManual` with operator-readable
  reasons instead of failing silently.
- **Main TUI dashboard (issue #19)** — The main TUI is now a navigable menu
  with five sections: Install, Inventory, Recent Memories, Search, Exit.
  Recent Memories shows the last 5 local Raven events sorted by `ObservedAt`
  (fallback to `IngestedAt`). Search filters events across 7 fields (CI ID,
  summary, details, source, type, severity, status) with a 20-match cap and
  total-count reporting. `cmd/raven/main.go` loads both components and events
  via the new `buildDashboardModel` wiring.
- **Agent contract template (issue #21)** — Shipped `.agents/assistant.yaml`
  declaring a `raven-incident-assistant` profile for MCP-aware clients. Uses
  the same managed-block markers as `raven setup` so future automation can
  adopt it without rewriting.

### Changed

- Documentation clarified that issue #15 first slice is project-local
  (`docs/agent-setup.md`, `docs/ai-usage.md`).
- Setup TUI plan review and apply summary now show per-item reasons for
  `ActionSkip` and `ActionManual` items, with separate `Validation failed`
  and `Manual` counts in the summary.

### Documentation (untagged since v0.1.0)

- **Open WebUI integration guide (PR #16 / PR #17)** — Added
  `docs/integrations/open-webui.md` covering how to wire Raven's local MCP
  server into Open WebUI as persistent memory for LLM-driven diagnostic
  workflows.
- **Memory model documentation (commit `2127491`)** — Added
  `How Raven memory works` section to `README.md` describing the
  CI-centered model: canonical `ci_id` → aliases → upstream references.

## [0.1.0] - 2026-06-07

Initial tagged release covering the setup TUI foundation and CI gates.

### Added

- Setup TUI tracker (PR #7)
- Setup TUI planner (PR #8)
- Approved apply and validation (PR #9)
- Setup TUI wizard model (PR #10)
- Setup templates and docs (PR #11)
- PR quality gates (PR #13) — required CI for `status:approved` linkage,
  `type:*` labels, and ShellCheck
