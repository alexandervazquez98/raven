package setup

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
)

func Plan(env SetupEnv) (PlanResult, error) {
	if env.FS == nil {
		env.FS = OSFileSystem{}
	}

	items := []PlanItem{
		fileItem(env, "ollama-modelfile", EcosystemOllama, filepath.Join(env.ProjectDir, "ollama", "Modelfile.raven"), "managed-file-present", smokeIfPresent(env, "ollama", "ollama show raven-support")),
		fileItem(env, "gemini-settings", EcosystemGeminiCLI, filepath.Join(env.ProjectDir, ".gemini", "settings.json"), "json-parse", smokeIfPresent(env, "gemini", "gemini --version")),
		managedBlockItem(env, "codex-agents", EcosystemCodex, filepath.Join(env.ProjectDir, "AGENTS.md"), "managed-block-present", smokeIfPresent(env, "codex", "codex --version"), "codex-agents"),
		{ID: "antigravity-guidance", Ecosystem: EcosystemAntigravity, Scope: ScopeManual, Action: ActionManual, ManualWarning: "Antigravity user/global configuration is not written by setup without a later explicit safe target."},
		fileItem(env, "raven-agent-contract", EcosystemRavenAgents, filepath.Join(env.ProjectDir, ".agents", "assistant.yaml"), "managed-file-present", ""),
		managedBlockItem(env, "raven-incident-skill", EcosystemRavenAgents, filepath.Join(env.ProjectDir, ".agents", "skills", "raven-incident", "SKILL.md"), "managed-block-present", "", "raven-incident-skill"),
	}

	return PlanResult{Items: items}, nil
}

func fileItem(env SetupEnv, id string, ecosystem Ecosystem, targetPath, validation, smokeCommand string) PlanItem {
	return PlanItem{ID: id, Ecosystem: ecosystem, TargetPath: targetPath, Scope: ScopeProjectLocal, Action: plannedFileAction(env.FS, targetPath), ValidationMethod: validation, SmokeTestCommand: smokeCommand}
}

func managedBlockItem(env SetupEnv, id string, ecosystem Ecosystem, targetPath, validation, smokeCommand, blockID string) PlanItem {
	item := fileItem(env, id, ecosystem, targetPath, validation, smokeCommand)
	item.ManagedBlockID = blockID
	return item
}

func plannedFileAction(files FileSystem, targetPath string) Action {
	content, err := readExistingFile(files, targetPath)
	if errors.Is(err, fs.ErrNotExist) {
		return ActionCreate
	}
	if err != nil {
		return ActionManual
	}
	if strings.Contains(string(content), "BEGIN RAVEN MANAGED") {
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
