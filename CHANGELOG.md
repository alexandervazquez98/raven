# Changelog

All notable changes to Raven are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

> **Note**: Raven is in an early testing phase (`0.x`). Minor versions may include
> breaking changes; the project will stabilize at `1.0.0` when the public surface
> is frozen.

## [Unreleased]

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
