package setup

import (
	"fmt"
	"strings"
)

func GeminiSettingsJSON() string {
	return strings.Join([]string{
		"{",
		`  "context": {`,
		`    "fileName": ["AGENTS.md", "GEMINI.md"]`,
		"  },",
		`  "mcpServers": {`,
		`    "raven": {`,
		`      "command": "raven",`,
		`      "args": ["mcp"]`,
		"    }",
		"  }",
		"}",
		"",
	}, "\n")
}

func CodexAgentsBlock() string {
	return strings.Join([]string{
		"Use Raven as the local CMDB and operational timeline.",
		"- CI ID is mandatory. Do not invent CI IDs.",
		"- Raven CI IDs are canonical; next-gen CI IDs are upstream references and must be resolved through Raven aliases first.",
		"- Before diagnosing a known CI, inspect prior context with `raven timeline <ci-id>` when useful.",
		"- Prefer `raven mcp`; use Raven CLI commands only as fallback.",
		"- Never write secrets or access tokens into project files.",
		"- Capture important diagnostics or resolutions in Raven only after operator approval.",
		"",
	}, "\n")
}

func OllamaModelfile() string {
	return strings.Join([]string{
		"# BEGIN RAVEN MANAGED: ollama-modelfile",
		"FROM qwen3.5:4b",
		"PARAMETER temperature 0.2",
		"PARAMETER num_ctx 8192",
		"SYSTEM \"\"\"",
		"You are an L1 Incident Assistant for network and endpoint operations.",
		"Raven is the local MCP/CLI-backed CMDB, CI alias resolver, timeline, and operational memory.",
		"next-gen CI IDs are upstream references, not Raven CI IDs; resolve them through Raven aliases before CI-specific reasoning.",
		"Do not invent CI IDs, tool results, fixes, or resolutions.",
		"Prefer Raven MCP tools; use CLI commands only as fallback.",
		"Do not persist anything without explicit operator approval.",
		"Separate facts, hypotheses, and recommended next checks. Be concise and operational.",
		"\"\"\"",
		"# END RAVEN MANAGED: ollama-modelfile",
		"",
	}, "\n")
}

func RavenAssistantYAML() string {
	return strings.Join([]string{
		"# BEGIN RAVEN MANAGED: raven-agent-contract",
		"name: raven-incident-assistant",
		"role: local-cmdb-and-operational-timeline",
		"rules:",
		"  - CI ID is mandatory; do not invent CI IDs.",
		"  - Resolve upstream references through Raven aliases before CI-specific reasoning.",
		"  - Prefer Raven MCP tools; use CLI commands as fallback.",
		"  - Never write secrets or access tokens into project files.",
		"  - Capture operational events only after operator approval.",
		"# END RAVEN MANAGED: raven-agent-contract",
		"",
	}, "\n")
}

func RavenIncidentSkillBlock() string {
	return strings.Join([]string{
		"Raven setup installs this managed reminder for incident workflows:",
		"- Use canonical Raven CI IDs only.",
		"- Treat next-gen IDs as upstream aliases, not Raven identity.",
		"- Preserve evidence and ask before recording events.",
		"",
	}, "\n")
}

func UpsertManagedBlock(existing, blockID, generated string) (string, error) {
	begin := managedBlockBegin(blockID)
	end := managedBlockEnd(blockID)
	beginCount := strings.Count(existing, begin)
	endCount := strings.Count(existing, end)

	if beginCount != endCount {
		return "", fmt.Errorf("managed block %q has mismatched markers", blockID)
	}
	if beginCount > 1 {
		return "", fmt.Errorf("managed block %q appears more than once", blockID)
	}

	block := renderManagedBlock(blockID, generated)
	if beginCount == 0 {
		return appendBlock(existing, block), nil
	}

	beginAt := strings.Index(existing, begin)
	endAt := strings.Index(existing, end)
	if beginAt < 0 || endAt < 0 || endAt < beginAt {
		return "", fmt.Errorf("managed block %q has malformed markers", blockID)
	}
	endAt += len(end)
	return existing[:beginAt] + block + existing[endAt:], nil
}

func renderManagedBlock(blockID, generated string) string {
	return managedBlockBegin(blockID) + "\n" + strings.TrimRight(generated, "\n") + "\n" + managedBlockEnd(blockID)
}

func managedBlockBegin(blockID string) string {
	return fmt.Sprintf("<!-- BEGIN RAVEN MANAGED: %s -->", blockID)
}

func managedBlockEnd(blockID string) string {
	return fmt.Sprintf("<!-- END RAVEN MANAGED: %s -->", blockID)
}

func appendBlock(existing, block string) string {
	if strings.TrimSpace(existing) == "" {
		return block + "\n"
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + block + "\n"
}
