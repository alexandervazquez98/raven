package setup

import "testing"

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
