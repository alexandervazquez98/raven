package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateGeneratedJSONContent(t *testing.T) {
	item := PlanItem{ID: "gemini", ValidationMethod: "json-parse", GeneratedContent: `{"mcpServers":{"raven":{}}}`}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}})

	result := requireValidationResult(t, results, "gemini")
	if result.Status != ValidationPassed {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationPassed, result)
	}
}

func TestValidateRejectsConcreteSecretsButAllowsPlaceholders(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    ValidationStatus
	}{
		{name: "placeholder", content: "token_env: NEXTGEN_ACCESS_TOKEN\n", want: ValidationPassed},
		{name: "shell placeholder", content: "token: ${NEXTGEN_ACCESS_TOKEN}\n", want: ValidationPassed},
		{name: "concrete fake token", content: "token: concrete-token-value-for-tests-1234567890abcdef\n", want: ValidationFailed},
		{name: "password assignment", content: "password = not-a-real-password-for-tests\n", want: ValidationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := Validate([]PlanItem{{ID: tt.name, GeneratedContent: tt.content}}, SetupEnv{Commands: fakeCommands{}})
			result := requireValidationResult(t, results, tt.name)
			if result.Status != tt.want {
				t.Fatalf("status = %q, want %q; result=%#v", result.Status, tt.want, result)
			}
		})
	}
}

func TestValidateReportsMissingExternalToolAsManualSmokeTest(t *testing.T) {
	item := PlanItem{ID: "ollama", SmokeTestCommand: "ollama show raven-support"}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}})

	result := requireValidationResult(t, results, "ollama")
	if result.Status != ValidationManual {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationManual, result)
	}
	if result.SmokeTestCommand != item.SmokeTestCommand {
		t.Fatalf("smoke command = %q, want %q", result.SmokeTestCommand, item.SmokeTestCommand)
	}
}

func TestValidateManualItemReportsManualInsteadOfFailed(t *testing.T) {
	target := filepath.Join(t.TempDir(), "Modelfile.raven")
	if err := os.WriteFile(target, []byte("operator-owned content\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	item := PlanItem{ID: "ollama-modelfile", Action: ActionManual, TargetPath: target, ValidationMethod: "managed-file-present", GeneratedContent: OllamaModelfile()}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}, FS: OSFileSystem{}})

	result := requireValidationResult(t, results, item.ID)
	if result.Status != ValidationManual {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationManual, result)
	}
	if result.Message != "existing file differs; review manually" {
		t.Fatalf("message = %q, want manual review reason", result.Message)
	}
}

func TestValidateManagedFilePresent(t *testing.T) {
	target := filepath.Join(t.TempDir(), "assistant.yaml")
	if err := os.WriteFile(target, []byte("name: raven\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	item := PlanItem{ID: "raven-agent-contract", TargetPath: target, ValidationMethod: "managed-file-present"}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}, FS: OSFileSystem{}})

	result := requireValidationResult(t, results, item.ID)
	if result.Status != ValidationPassed {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationPassed, result)
	}
}

func TestValidateManagedFilePresentFailsWhenMissing(t *testing.T) {
	item := PlanItem{ID: "raven-agent-contract", TargetPath: filepath.Join(t.TempDir(), "missing.yaml"), ValidationMethod: "managed-file-present"}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}, FS: OSFileSystem{}})

	result := requireValidationResult(t, results, item.ID)
	if result.Status != ValidationFailed {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationFailed, result)
	}
}

func TestValidateManagedFilePresentFailsWhenContentIsStale(t *testing.T) {
	target := filepath.Join(t.TempDir(), "assistant.yaml")
	if err := os.WriteFile(target, []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	item := PlanItem{ID: "raven-agent-contract", TargetPath: target, ValidationMethod: "managed-file-present", GeneratedContent: RavenAssistantYAML()}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}, FS: OSFileSystem{}})

	result := requireValidationResult(t, results, item.ID)
	if result.Status != ValidationFailed {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationFailed, result)
	}
}

func TestValidateManagedBlockPresent(t *testing.T) {
	target := filepath.Join(t.TempDir(), "AGENTS.md")
	content := renderManagedBlock("raven-local-ai-guidance", RavenLocalAIGuidance())
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	item := PlanItem{ID: "raven-local-ai-guidance", TargetPath: target, ValidationMethod: "managed-block-present", ManagedBlockID: "raven-local-ai-guidance"}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}, FS: OSFileSystem{}})

	result := requireValidationResult(t, results, item.ID)
	if result.Status != ValidationPassed {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationPassed, result)
	}
}

func TestValidateManagedBlockPresentFailsWhenMissing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := os.WriteFile(target, []byte("# Raven\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	item := PlanItem{ID: "raven-local-ai-guidance", TargetPath: target, ValidationMethod: "managed-block-present", ManagedBlockID: "raven-local-ai-guidance"}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}, FS: OSFileSystem{}})

	result := requireValidationResult(t, results, item.ID)
	if result.Status != ValidationFailed {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationFailed, result)
	}
}

func TestValidateManagedBlockPresentFailsWhenContentIsStale(t *testing.T) {
	target := filepath.Join(t.TempDir(), "AGENTS.md")
	content := renderManagedBlock("raven-local-ai-guidance", "stale guidance\n")
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	item := PlanItem{ID: "raven-local-ai-guidance", TargetPath: target, ValidationMethod: "managed-block-present", ManagedBlockID: "raven-local-ai-guidance", GeneratedContent: RavenLocalAIGuidance()}

	results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}, FS: OSFileSystem{}})

	result := requireValidationResult(t, results, item.ID)
	if result.Status != ValidationFailed {
		t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationFailed, result)
	}
}

func TestValidateManagedBlockPresentFailsWhenMalformed(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "begin without end", content: managedBlockBegin("raven-local-ai-guidance") + "\nmissing end marker\n"},
		{name: "end before begin", content: managedBlockEnd("raven-local-ai-guidance") + "\n" + managedBlockBegin("raven-local-ai-guidance") + "\n"},
		{name: "duplicate blocks", content: renderManagedBlock("raven-local-ai-guidance", "one") + "\n" + renderManagedBlock("raven-local-ai-guidance", "two")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "AGENTS.md")
			if err := os.WriteFile(target, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("write target: %v", err)
			}
			item := PlanItem{ID: "raven-local-ai-guidance", TargetPath: target, ValidationMethod: "managed-block-present", ManagedBlockID: "raven-local-ai-guidance"}

			results := Validate([]PlanItem{item}, SetupEnv{Commands: fakeCommands{}, FS: OSFileSystem{}})

			result := requireValidationResult(t, results, item.ID)
			if result.Status != ValidationFailed {
				t.Fatalf("status = %q, want %q; result=%#v", result.Status, ValidationFailed, result)
			}
		})
	}
}

func requireValidationResult(t *testing.T, results []ValidationResult, itemID string) ValidationResult {
	t.Helper()
	for _, result := range results {
		if result.ItemID == itemID {
			return result
		}
	}
	t.Fatalf("validation result %q not found in %#v", itemID, results)
	return ValidationResult{}
}
