package setup

import (
	"errors"
	"io/fs"
	"path"
	"runtime"
	"strings"
)

func Plan(env SetupEnv) (PlanResult, error) {
	env = withDefaults(env)

	items := []PlanItem{
		completeFileItem(env, "ollama-modelfile", EcosystemOllama, platformJoin(env.GOOS, env.ProjectDir, "ollama", "Modelfile.raven"), "managed-file-present", smokeIfPresent(env, "ollama", "ollama show raven-support"), OllamaModelfile()),
		completeFileItem(env, "gemini-settings", EcosystemGeminiCLI, platformJoin(env.GOOS, env.ProjectDir, ".gemini", "settings.json"), "json-parse", smokeIfPresent(env, "gemini", "gemini --version"), GeminiSettingsJSON()),
		managedBlockItem(env, "codex-agents", EcosystemCodex, platformJoin(env.GOOS, env.ProjectDir, "AGENTS.md"), "managed-block-present", smokeIfPresent(env, "codex", "codex --version"), "codex-agents", CodexAgentsBlock()),
		{
			ID:             "codex-global-guidance",
			Ecosystem:      EcosystemCodex,
			TargetPath:     platformJoin(env.GOOS, env.HomeDir, ".codex", "config.toml"),
			Scope:          ScopeUserGlobal,
			Action:         ActionManual,
			ManualWarning:  "Codex user-global MCP/provider configuration is shown as guidance only; setup does not write it without a future explicit safe target.",
			ManagedBlockID: "",
		},
		{
			ID:            "antigravity-guidance",
			Ecosystem:     EcosystemAntigravity,
			Scope:         ScopeManual,
			Action:        ActionManual,
			ManualWarning: "Antigravity user/global configuration is not written by setup without a later explicit safe target.",
		},
		completeFileItem(env, "raven-agent-contract", EcosystemRavenAgents, platformJoin(env.GOOS, env.ProjectDir, ".agents", "assistant.yaml"), "managed-file-present", "", RavenAssistantYAML()),
		managedBlockItem(env, "raven-incident-skill", EcosystemRavenAgents, platformJoin(env.GOOS, env.ProjectDir, ".agents", "skills", "raven-incident", "SKILL.md"), "managed-block-present", "", "raven-incident-skill", RavenIncidentSkillBlock()),
	}

	return PlanResult{Items: items}, nil
}

func withDefaults(env SetupEnv) SetupEnv {
	if env.FS == nil {
		env.FS = OSFileSystem{}
	}
	if env.GOOS == "" {
		env.GOOS = runtime.GOOS
	}
	return env
}

func completeFileItem(env SetupEnv, id string, ecosystem Ecosystem, targetPath, validation, smokeCommand, generated string) PlanItem {
	return PlanItem{
		ID:               id,
		Ecosystem:        ecosystem,
		TargetPath:       targetPath,
		Scope:            ScopeProjectLocal,
		Action:           plannedCompleteFileAction(env.FS, targetPath, generated),
		ValidationMethod: validation,
		SmokeTestCommand: smokeCommand,
		GeneratedContent: generated,
	}
}

func managedBlockItem(env SetupEnv, id string, ecosystem Ecosystem, targetPath, validation, smokeCommand, blockID, generated string) PlanItem {
	return PlanItem{
		ID:               id,
		Ecosystem:        ecosystem,
		TargetPath:       targetPath,
		Scope:            ScopeProjectLocal,
		Action:           plannedManagedBlockAction(env.FS, targetPath, blockID),
		ValidationMethod: validation,
		SmokeTestCommand: smokeCommand,
		ManagedBlockID:   blockID,
		GeneratedContent: generated,
	}
}

func plannedCompleteFileAction(files FileSystem, targetPath, generated string) Action {
	content, err := readExistingFile(files, targetPath)
	if errors.Is(err, fs.ErrNotExist) {
		return ActionCreate
	}
	if err != nil {
		return ActionManual
	}
	if string(content) == generated || strings.Contains(string(content), RavenManagedMarker) {
		return ActionSkip
	}
	return ActionManual
}

func plannedManagedBlockAction(files FileSystem, targetPath, blockID string) Action {
	content, err := readExistingFile(files, targetPath)
	if errors.Is(err, fs.ErrNotExist) {
		return ActionCreate
	}
	if err != nil {
		return ActionManual
	}
	if strings.Contains(string(content), managedBlockBegin(blockID)) {
		return ActionSkip
	}
	return ActionUpdate
}

func readExistingFile(files FileSystem, targetPath string) ([]byte, error) {
	_, err := files.Stat(targetPath)
	if err != nil {
		return nil, err
	}
	return files.ReadFile(targetPath)
}

func smokeIfPresent(env SetupEnv, commandName, smokeCommand string) string {
	if env.Commands == nil {
		return ""
	}
	if _, ok := env.Commands.LookPath(commandName); ok {
		return smokeCommand
	}
	return ""
}

func platformJoin(goos, root string, parts ...string) string {
	if goos == "windows" {
		segments := append([]string{strings.TrimRight(root, `\\/`)}, parts...)
		return strings.Join(segments, `\`)
	}
	segments := append([]string{root}, parts...)
	return path.Join(segments...)
}
