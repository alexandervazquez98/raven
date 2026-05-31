# Review work units

The current Raven workstream mixes agent setup docs, project skills, research artifacts, and next-gen MCP implementation. Split it before committing or opening review.

## Suggested slices

| Slice | Include | Keep out |
| --- | --- | --- |
| Agent setup docs | `docs/agent-setup.md`, `docs/ai-usage.md`, project-local setup examples | Go MCP implementation |
| Raven incident skill | `AGENTS.md`, `.agents/skills/raven-incident/SKILL.md`, `docs/design/raven-incident-workflow.md` | next-gen client/server code |
| next-gen MCP contract | `docs/design/nextgen-mcp-contract.md` | unrelated setup/research files |
| next-gen MCP implementation | `internal/nextgen/**`, `internal/nextgenmcp/**`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | global/local agent config, Ollama wrapper, research notes |
| Local experiment artifacts | `.gemini/`, `.codex/`, `ollama/`, `scripts/`, `research/` | production code unless promoted intentionally |

## Review rule

Each slice should pass its own validation. For Go slices run:

```bash
go test ./...
git diff --check
```

For docs/setup-only slices, run `git diff --check` and validate JSON/TOML/shell snippets where practical. Do not commit secrets or user/global config changes.
