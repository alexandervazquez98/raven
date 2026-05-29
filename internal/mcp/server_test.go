package ravenmcp

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"raven/internal/app"
	"raven/internal/domain"
	"raven/internal/storage"
)

func TestNewServerRegistersAgentTools(t *testing.T) {
	srv := NewServer(ServerConfig{ConfigDir: t.TempDir()})

	got := ToolNames(srv)
	want := []string{ToolGetCI, ToolGetTimeline, ToolListCIs, ToolRecordEvent, ToolResolveCIRef}
	if len(got) != len(want) {
		t.Fatalf("ToolNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ToolNames() = %v, want %v", got, want)
		}
	}

	tools := srv.ListTools()
	for _, name := range want {
		assertStrictTopLevelSchema(t, tools[name].Tool)
	}
	for _, name := range []string{ToolResolveCIRef, ToolGetTimeline, ToolListCIs, ToolGetCI} {
		assertReadOnlyLocalTool(t, tools[name].Tool)
	}

	recordTool := tools[ToolRecordEvent].Tool
	if recordTool.Annotations.ReadOnlyHint == nil || *recordTool.Annotations.ReadOnlyHint {
		t.Fatalf("record_event readOnlyHint = %v, want false", recordTool.Annotations.ReadOnlyHint)
	}
	if recordTool.Annotations.DestructiveHint == nil || *recordTool.Annotations.DestructiveHint {
		t.Fatalf("record_event destructiveHint = %v, want false", recordTool.Annotations.DestructiveHint)
	}
	if recordTool.Annotations.IdempotentHint == nil || *recordTool.Annotations.IdempotentHint {
		t.Fatalf("record_event idempotentHint = %v, want false", recordTool.Annotations.IdempotentHint)
	}
	if recordTool.Annotations.OpenWorldHint == nil || *recordTool.Annotations.OpenWorldHint {
		t.Fatalf("record_event openWorldHint = %v, want false", recordTool.Annotations.OpenWorldHint)
	}
	for _, required := range []string{"type", "severity", "summary", "source", "observed_at"} {
		if !contains(recordTool.InputSchema.Required, required) {
			t.Fatalf("record_event required = %v, want %q", recordTool.InputSchema.Required, required)
		}
	}

	ciRef, ok := recordTool.InputSchema.Properties["ci_ref"].(map[string]any)
	if !ok {
		t.Fatalf("record_event ci_ref schema = %#v, want object schema", recordTool.InputSchema.Properties["ci_ref"])
	}
	if ciRef["additionalProperties"] != false {
		t.Fatalf("record_event ci_ref additionalProperties = %#v, want false", ciRef["additionalProperties"])
	}
	requiredCIRef, ok := ciRef["required"].([]string)
	if !ok {
		t.Fatalf("record_event ci_ref required = %#v, want []string", ciRef["required"])
	}
	for _, required := range []string{"source", "type", "value"} {
		if !contains(requiredCIRef, required) {
			t.Fatalf("record_event ci_ref required = %v, want %q", requiredCIRef, required)
		}
	}

	resolveTool := tools[ToolResolveCIRef].Tool
	for _, required := range []string{"source", "type", "value"} {
		if !contains(resolveTool.InputSchema.Required, required) {
			t.Fatalf("resolve_ci_ref required = %v, want %q", resolveTool.InputSchema.Required, required)
		}
	}
	assertRequiredFields(t, tools[ToolGetTimeline].Tool, "ci_id")
	assertRequiredFields(t, tools[ToolGetCI].Tool, "ci_id")
}

func TestMCPHandlersResolveRecordAndReadTimeline(t *testing.T) {
	configDir := t.TempDir()
	seedMCPData(t, configDir)
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()
	observedAt := "2026-05-29T20:00:00Z"

	resolveResult := callTool(t, tools[ToolResolveCIRef].Handler, map[string]any{"source": "next-gen", "type": "ci_id", "value": "42"})
	if resolveResult.IsError {
		t.Fatalf("resolve result is error: %s", resultText(resolveResult))
	}
	if got := resolveResult.StructuredContent.(map[string]any)["ci_id"]; got != "FW-MAIN-001" {
		t.Fatalf("resolve ci_id = %v, want FW-MAIN-001", got)
	}

	recordResult := callTool(t, tools[ToolRecordEvent].Handler, map[string]any{
		"ci_ref": map[string]any{"source": "next-gen", "type": "ci_id", "value": "42"},
		"type":   "network_alert", "severity": "warning", "summary": "High packet loss", "source": "next-gen",
		"external_id": "ng-98765", "observed_at": observedAt,
	})
	if recordResult.IsError {
		t.Fatalf("record result is error: %s", resultText(recordResult))
	}
	stored, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(stored) != 1 || stored[0].CIID != "FW-MAIN-001" || stored[0].DedupKey != "next-gen:ng-98765" {
		t.Fatalf("stored events = %#v, want one canonical event", stored)
	}

	timelineResult := callTool(t, tools[ToolGetTimeline].Handler, map[string]any{"ci_id": "FW-MAIN-001"})
	if timelineResult.IsError {
		t.Fatalf("timeline result is error: %s", resultText(timelineResult))
	}
	events := timelineResult.StructuredContent.(map[string]any)["events"].([]domain.Event)
	if len(events) != 1 || events[0].Summary != "High packet loss" {
		t.Fatalf("timeline events = %#v, want recorded event", events)
	}
}

func TestMCPHandlersListAndGetCIs(t *testing.T) {
	configDir := t.TempDir()
	seedMCPData(t, configDir)
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	listResult := callTool(t, tools[ToolListCIs].Handler, nil)
	if listResult.IsError {
		t.Fatalf("list result is error: %s", resultText(listResult))
	}
	cis := listResult.StructuredContent.(map[string]any)["cis"].([]domain.Component)
	if len(cis) != 1 || cis[0].CIID != "FW-MAIN-001" {
		t.Fatalf("cis = %#v, want seeded CI", cis)
	}

	getResult := callTool(t, tools[ToolGetCI].Handler, map[string]any{"ci_id": "FW-MAIN-001"})
	if getResult.IsError {
		t.Fatalf("get result is error: %s", resultText(getResult))
	}
	ci := getResult.StructuredContent.(map[string]any)["ci"].(domain.Component)
	if ci.DisplayName() != "Fortinet FortiGate" {
		t.Fatalf("ci = %#v, want Fortinet FortiGate", ci)
	}
}

func TestMCPHandlersReturnReadableErrors(t *testing.T) {
	configDir := t.TempDir()
	seedMCPData(t, configDir)
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	missingAlias := callTool(t, tools[ToolResolveCIRef].Handler, map[string]any{"source": "next-gen", "type": "ci_id", "value": "missing"})
	if !missingAlias.IsError || !strings.Contains(resultText(missingAlias), "resolve ci_ref next-gen ci_id missing: alias not found") {
		t.Fatalf("missing alias result = %#v text=%q, want readable alias error", missingAlias, resultText(missingAlias))
	}

	missingCI := callTool(t, tools[ToolGetTimeline].Handler, map[string]any{"ci_id": "MISSING"})
	if !missingCI.IsError || !strings.Contains(resultText(missingCI), "component not found") {
		t.Fatalf("missing CI result = %#v text=%q, want readable CI error", missingCI, resultText(missingCI))
	}

	missingIdentity := callTool(t, tools[ToolRecordEvent].Handler, map[string]any{
		"type": "observation", "severity": "info", "summary": "No identity", "source": "agent",
		"external_id": "agent-1", "observed_at": "2026-05-29T20:00:00Z",
	})
	if !missingIdentity.IsError || !strings.Contains(resultText(missingIdentity), "requires ci_id or ci_ref") {
		t.Fatalf("missing identity result = %#v text=%q, want identity error", missingIdentity, resultText(missingIdentity))
	}
}

func callTool(t *testing.T, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
	if err != nil {
		t.Fatalf("handler error = %v, want nil", err)
	}
	return result
}

func resultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if text, ok := result.Content[0].(mcp.TextContent); ok {
		return text.Text
	}
	return ""
}

func seedMCPData(t *testing.T, configDir string) {
	t.Helper()
	if err := storage.SaveComponents(app.ComponentsPath(configDir), []domain.Component{{CIID: "FW-MAIN-001", Category: "network", Manufacturer: "Fortinet", Model: "FortiGate"}}); err != nil {
		t.Fatalf("SaveComponents() error = %v, want nil", err)
	}
	if err := storage.SaveAliases(app.AliasesPath(configDir), []domain.Alias{{CIID: "FW-MAIN-001", Source: "next-gen", Type: domain.AliasTypeCIID, Value: "42"}}); err != nil {
		t.Fatalf("SaveAliases() error = %v, want nil", err)
	}
}

func assertStrictTopLevelSchema(t *testing.T, tool mcp.Tool) {
	t.Helper()
	if tool.InputSchema.AdditionalProperties != false {
		t.Fatalf("%s additionalProperties = %#v, want false", tool.Name, tool.InputSchema.AdditionalProperties)
	}
}

func assertReadOnlyLocalTool(t *testing.T, tool mcp.Tool) {
	t.Helper()
	if tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint {
		t.Fatalf("%s readOnlyHint = %v, want true", tool.Name, tool.Annotations.ReadOnlyHint)
	}
	if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
		t.Fatalf("%s destructiveHint = %v, want false", tool.Name, tool.Annotations.DestructiveHint)
	}
	if tool.Annotations.IdempotentHint == nil || !*tool.Annotations.IdempotentHint {
		t.Fatalf("%s idempotentHint = %v, want true", tool.Name, tool.Annotations.IdempotentHint)
	}
	if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
		t.Fatalf("%s openWorldHint = %v, want false", tool.Name, tool.Annotations.OpenWorldHint)
	}
}

func assertRequiredFields(t *testing.T, tool mcp.Tool, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if !contains(tool.InputSchema.Required, field) {
			t.Fatalf("%s required = %v, want %q", tool.Name, tool.InputSchema.Required, field)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
