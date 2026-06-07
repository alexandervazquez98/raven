package setup

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type ValidationStatus string

const (
	ValidationPassed ValidationStatus = "passed"
	ValidationFailed ValidationStatus = "failed"
	ValidationManual ValidationStatus = "manual"
)

type ValidationResult struct {
	ItemID           string
	TargetPath       string
	Status           ValidationStatus
	Message          string
	SmokeTestCommand string
}

func Validate(items []PlanItem, env SetupEnv) []ValidationResult {
	env = withDefaults(env)
	results := make([]ValidationResult, 0, len(items))
	for _, item := range items {
		result := ValidationResult{ItemID: item.ID, TargetPath: item.TargetPath, Status: ValidationPassed}
		if err := rejectSecrets(item.GeneratedContent); err != nil {
			result.Status = ValidationFailed
			result.Message = err.Error()
			results = append(results, result)
			continue
		}
		if item.ValidationMethod == "json-parse" && strings.TrimSpace(item.GeneratedContent) != "" {
			if !json.Valid([]byte(item.GeneratedContent)) {
				result.Status = ValidationFailed
				result.Message = "generated JSON does not parse"
				results = append(results, result)
				continue
			}
		}
		if item.SmokeTestCommand != "" && !commandAvailable(env.Commands, item.SmokeTestCommand) {
			result.Status = ValidationManual
			result.Message = "external tool is not available; run smoke test manually"
			result.SmokeTestCommand = item.SmokeTestCommand
		}
		results = append(results, result)
	}
	return results
}

func commandAvailable(commands CommandDetector, smokeCommand string) bool {
	if commands == nil {
		return false
	}
	fields := strings.Fields(smokeCommand)
	if len(fields) == 0 {
		return false
	}
	_, ok := commands.LookPath(fields[0])
	return ok
}

func rejectSecrets(content string) error {
	for _, line := range strings.Split(content, "\n") {
		if !looksSensitiveLine(line) {
			continue
		}
		value, ok := sensitiveValue(line)
		if !ok {
			continue
		}
		if isAllowedPlaceholder(value) {
			continue
		}
		return fmt.Errorf("generated setup content must not contain concrete secret values")
	}
	return nil
}

func looksSensitiveLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "credential") || strings.Contains(lower, "access_key")
}

func sensitiveValue(line string) (string, bool) {
	for _, sep := range []string{"=", ":"} {
		if before, after, ok := strings.Cut(line, sep); ok && looksSensitiveLine(before) {
			return strings.Trim(strings.TrimSpace(after), `"'`), true
		}
	}
	return "", false
}

var envNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func isAllowedPlaceholder(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if value == "REDACTED" || strings.HasPrefix(value, "<") || strings.HasPrefix(value, "YOUR_") {
		return true
	}
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		return envNamePattern.MatchString(strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}"))
	}
	if strings.HasPrefix(value, "$") {
		return envNamePattern.MatchString(strings.TrimPrefix(value, "$"))
	}
	if strings.HasPrefix(value, "%") && strings.HasSuffix(value, "%") {
		return envNamePattern.MatchString(strings.TrimSuffix(strings.TrimPrefix(value, "%"), "%"))
	}
	return envNamePattern.MatchString(value)
}
