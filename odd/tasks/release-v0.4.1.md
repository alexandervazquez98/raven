# Release: v0.4.1 — MCP tool naming cleanup (#27) + configurable data dir (#28)

## Source

- GitHub issues:
  - [#27](https://github.com/alexandervazquez98/raven/issues/27) — `chore(mcp): drop redundant raven_ prefix from MCP tool names`
  - [#28](https://github.com/alexandervazquez98/raven/issues/28) — `feat(config): configurable data directory via RAVEN_DATA_DIR and --data-dir`
- Both issues captured from the same field test on PR #25 / #23 (Qwen 3.5 9B + NORA sandbox).
- Maintainer decision (2026-09-20): ship both as a batched release on `release/v0.4.1`.
- Env var name locked: **`RAVEN_DATA_DIR`** (issue #28 spec wins over the older `RAVEN_STORAGE_DIR` mention in `docs/integrations/open-webui.md`).

## Goal

Two independent improvements, no code coupling, batched into a single release for narrative and review focus:

1. **#27 — clean rename.** Drop the redundant `raven_` prefix from MCP tool constants so MCP clients (e.g. Open WebUI) that auto-prefix by server name do not surface double-prefixed tool names like `raven_raven_list_cis`. Slice 4 of #23 (PR #31) shipped with the old prefix; this release cleans it up retroactively.
2. **#28 — sandbox-friendly storage path.** Allow overriding the data directory via `RAVEN_DATA_DIR` env var and `--data-dir` global CLI flag, default `~/.config/raven/` unchanged. Supports sandboxed evals, multi-tenant pipelines, and isolated container runs.

## Design decisions (locked)

1. **Single release branch** `release/v0.4.1` cut from `main`.
2. **Two PRs targeting the same release branch.** One concern per PR.
3. **PR order**: #27 first (mechanical, fully reversible rename), then #28 (touches `main()` dispatch and storage path resolution).
4. **Merge to `main` once after both PRs land**, then tag `v0.4.1` (tag is the user's call).
5. **CHANGELOG `[0.4.1]` entry** uses `### Added` for #28 and `### Changed` for #27 (the rename is breaking for any MCP client that cached the old tool names).
6. **Historical slicing docs** (`odd/tasks/feature-metadata-sidecar.md`, `odd/tasks/feature-list-filtering.md`) and `CHANGELOG.md` history stay untouched per project convention.
7. **`internal/app/paths.go` requires no changes** — path functions already accept an arbitrary base directory.
8. **Path precedence**: `--data-dir` flag > `RAVEN_DATA_DIR` env var > `os.UserConfigDir()` default. Documented explicitly in code and help text.

## Slicing

| # | Issue | Scope (summary) | Branch | PR | Commit hash |
|---|---|---|---|---|---|
| 1 | #27 | Drop `raven_` prefix from MCP tool constants + docs + skill + reconcile inconsistent inline strings | `release/v0.4.1` | TBD | TBD |
| 2 | #28 | `--data-dir` global flag + `RAVEN_DATA_DIR` env var + precedence tests + doc updates | `release/v0.4.1` | TBD | TBD |
| 3 | release | CHANGELOG `[0.4.1]` entry + `release/v0.4.1` → `main` merge | `release/v0.4.1` | direct | TBD |

## PR 1 (#27) acceptance criteria

- [ ] All 7 tool constants in `internal/mcp/server.go` lose the `raven_` prefix.
- [ ] The 3 inline string mentions (1 comment + 2 error strings) reconciled to the new convention (current line 312 already drops the prefix; lines 333 and 335 keep it — align all to the new convention).
- [ ] No `raven_*` tool-name mentions remain in `docs/integrations/open-webui.md`, `docs/ai-usage.md`, `docs/agent-setup.md`, `docs/design/raven-incident-workflow.md`, `docs/design/nextgen-mcp-contract.md`.
- [ ] `.agents/skills/raven-incident/SKILL.md` tool table and workflow steps updated.
- [ ] The meta-mention "the five `raven_*` tools" at `docs/integrations/open-webui.md:312` reworded.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` clean.
- [ ] Work-unit commit on `release/v0.4.1`.
- [ ] PR opened against `release/v0.4.1` with description noting operator action required (cached MCP tool references in client configs need refresh).

## PR 2 (#28) acceptance criteria

- [ ] `--data-dir` parsed before `selectRunMode` runs in `cmd/raven/main.go`.
- [ ] `RAVEN_DATA_DIR` env var read at startup with `strings.TrimSpace`.
- [ ] Precedence: flag > env > default.
- [ ] Helper `ResolveDataDir(flagValue string) (string, error)` (or equivalent) with focused unit tests.
- [ ] `docs/integrations/open-webui.md` updated to use `RAVEN_DATA_DIR` (replacing the prior `RAVEN_STORAGE_DIR` references); Windows note added where applicable.
- [ ] `docs/ai-usage.md` line 156 prose mention updated.
- [ ] `docs/design/next-gen-event-ingest.md` lines 111, 112 prose mentions updated.
- [ ] New tests cover: env-only, flag-only, both-set (flag wins), neither-set (default), empty-string env (treated as unset).
- [ ] `go test ./...` passes; `go vet ./...` clean.
- [ ] Existing tests unaffected (all current tests pass `t.TempDir()` directly, bypassing the new resolution path).
- [ ] Work-unit commit on `release/v0.4.1`.
- [ ] PR opened against `release/v0.4.1`.

## Release merge acceptance criteria

- [ ] Both PRs merged into `release/v0.4.1`.
- [ ] `CHANGELOG.md` `[0.4.1]` entry added with `### Added` (#28) and `### Changed` (#27) sections.
- [ ] `release/v0.4.1` merged to `main` (user-authorized).
- [ ] `v0.4.1` git tag (user-authorized, not agent-driven).

## Out of scope (explicit)

- Per-file overrides of the data dir.
- Path validation / security review beyond what `filepath.Join` already provides.
- Migrating existing `~/.config/raven/` contents.
- Renaming `nextgen_*` MCP tools (already correctly prefixed in `internal/nextgenmcp/server.go`).
- Server-name override in MCP clients (clients decide their prefix).
- Backfill of cached tool references in operator MCP client configs (documented as operator action required in PR #27 description).
