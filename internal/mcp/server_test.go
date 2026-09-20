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
	want := []string{ToolGetCI, ToolGetCIMetadata, ToolGetTimeline, ToolListCIs, ToolRecordEvent, ToolResolveCIRef, ToolSetCIMetadata}
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
	for _, name := range []string{ToolResolveCIRef, ToolGetTimeline, ToolListCIs, ToolGetCI, ToolGetCIMetadata} {
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
	assertRequiredFields(t, tools[ToolGetCIMetadata].Tool, "ci_id")
	assertRequiredFields(t, tools[ToolSetCIMetadata].Tool, "ci_id")

	setMetadataTool := tools[ToolSetCIMetadata].Tool
	if setMetadataTool.Annotations.ReadOnlyHint == nil || *setMetadataTool.Annotations.ReadOnlyHint {
		t.Fatalf("set_ci_metadata readOnlyHint = %v, want false", setMetadataTool.Annotations.ReadOnlyHint)
	}
	if setMetadataTool.Annotations.DestructiveHint == nil || !*setMetadataTool.Annotations.DestructiveHint {
		t.Fatalf("set_ci_metadata destructiveHint = %v, want true", setMetadataTool.Annotations.DestructiveHint)
	}
	if setMetadataTool.Annotations.IdempotentHint == nil || !*setMetadataTool.Annotations.IdempotentHint {
		t.Fatalf("set_ci_metadata idempotentHint = %v, want true", setMetadataTool.Annotations.IdempotentHint)
	}
	if setMetadataTool.Annotations.OpenWorldHint == nil || *setMetadataTool.Annotations.OpenWorldHint {
		t.Fatalf("set_ci_metadata openWorldHint = %v, want false", setMetadataTool.Annotations.OpenWorldHint)
	}
	if _, ok := setMetadataTool.InputSchema.Properties["attributes"]; !ok {
		t.Fatalf("set_ci_metadata schema missing attributes property (properties=%v)", setMetadataTool.InputSchema.Properties)
	}
	if _, ok := setMetadataTool.InputSchema.Properties["relationships"]; !ok {
		t.Fatalf("set_ci_metadata schema missing relationships property (properties=%v)", setMetadataTool.InputSchema.Properties)
	}
	for _, required := range setMetadataTool.InputSchema.Required {
		if required == "attributes" || required == "relationships" {
			t.Fatalf("set_ci_metadata schema unexpectedly requires %q (required=%v)", required, setMetadataTool.InputSchema.Required)
		}
	}
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

func TestMCPListCIsFilters(t *testing.T) {
	configDir := t.TempDir()
	if err := storage.SaveComponents(app.ComponentsPath(configDir), []domain.Component{
		{CIID: "FW-MAIN-001", Category: "network", Manufacturer: "Fortinet", Model: "FortiGate", Notes: "Primary edge firewall"},
		{CIID: "TWR-CORE-001", Category: "network", Manufacturer: "Cisco", Model: "Catalyst 9300", Notes: "Core switch"},
		{CIID: "SRV-DB-001", Category: "server", Manufacturer: "Dell", Model: "PowerEdge R750", Notes: "Database host"},
	}); err != nil {
		t.Fatalf("SaveComponents() error = %v, want nil", err)
	}
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	type wantSet struct {
		ids    []string
		absent []string
	}
	tests := []struct {
		name string
		args map[string]any
		want wantSet
	}{
		{
			name: "no args returns all",
			args: nil,
			want: wantSet{ids: []string{"FW-MAIN-001", "TWR-CORE-001", "SRV-DB-001"}},
		},
		{
			name: "category exact match",
			args: map[string]any{"category": "network"},
			want: wantSet{
				ids:    []string{"FW-MAIN-001", "TWR-CORE-001"},
				absent: []string{"SRV-DB-001"},
			},
		},
		{
			name: "prefix",
			args: map[string]any{"prefix": "TWR-"},
			want: wantSet{
				ids:    []string{"TWR-CORE-001"},
				absent: []string{"FW-MAIN-001", "SRV-DB-001"},
			},
		},
		{
			name: "query case-insensitive on model",
			args: map[string]any{"query": "forti"},
			want: wantSet{
				ids:    []string{"FW-MAIN-001"},
				absent: []string{"TWR-CORE-001", "SRV-DB-001"},
			},
		},
		{
			name: "limit caps",
			args: map[string]any{"limit": float64(2)},
			want: wantSet{ids: []string{"FW-MAIN-001", "TWR-CORE-001"}},
		},
		{
			name: "category plus prefix combines",
			args: map[string]any{"category": "network", "prefix": "FW-"},
			want: wantSet{
				ids:    []string{"FW-MAIN-001"},
				absent: []string{"TWR-CORE-001", "SRV-DB-001"},
			},
		},
		{
			name: "query plus limit combines",
			args: map[string]any{"query": "o", "limit": float64(1)},
			want: wantSet{ids: []string{"FW-MAIN-001"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := callTool(t, tools[ToolListCIs].Handler, tt.args)
			if result.IsError {
				t.Fatalf("list result is error: %s", resultText(result))
			}
			cis := result.StructuredContent.(map[string]any)["cis"].([]domain.Component)
			if len(cis) != len(tt.want.ids) {
				t.Fatalf("cis length = %d, want %d (cis=%#v)", len(cis), len(tt.want.ids), cis)
			}
			seen := make(map[string]bool, len(cis))
			for _, ci := range cis {
				seen[ci.CIID] = true
			}
			for _, want := range tt.want.ids {
				if !seen[want] {
					t.Fatalf("cis = %#v, want to contain %q", cis, want)
				}
			}
			for _, absent := range tt.want.absent {
				if seen[absent] {
					t.Fatalf("cis = %#v, did not want %q", cis, absent)
				}
			}
		})
	}
}

func TestNewServerListCIsSchemaAdvertisesOptionalFilters(t *testing.T) {
	srv := NewServer(ServerConfig{ConfigDir: t.TempDir()})
	listTool := srv.ListTools()[ToolListCIs].Tool

	assertStrictTopLevelSchema(t, listTool)
	for _, field := range []string{"category", "prefix", "query", "limit"} {
		if _, ok := listTool.InputSchema.Properties[field]; !ok {
			t.Fatalf("list_cis schema missing optional field %q (properties=%v)", field, listTool.InputSchema.Properties)
		}
	}
	for _, required := range listTool.InputSchema.Required {
		if required == "category" || required == "prefix" || required == "query" || required == "limit" {
			t.Fatalf("list_cis schema unexpectedly requires %q (required=%v)", required, listTool.InputSchema.Required)
		}
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

func TestMCPGetCIMetadataExistingEntry(t *testing.T) {
	configDir := t.TempDir()
	attrs := map[string]domain.TypedValue{"azimuth": domain.StringValue("north")}
	rels := []domain.CIRelationship{{TargetCIID: "SRV-DB-001", Kind: "hosts"}}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: attrs, Relationships: rels}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolGetCIMetadata].Handler, map[string]any{"ci_id": "FW-MAIN-001"})
	if result.IsError {
		t.Fatalf("get result is error: %s", resultText(result))
	}
	content := result.StructuredContent.(map[string]any)
	if got := content["ci_id"]; got != "FW-MAIN-001" {
		t.Fatalf("get ci_id = %v, want FW-MAIN-001", got)
	}
	gotAttrs, ok := content["attributes"].(map[string]domain.TypedValue)
	if !ok {
		t.Fatalf("get attributes type = %T, want map[string]domain.TypedValue (content=%#v)", content["attributes"], content)
	}
	if got := gotAttrs["azimuth"]; got.String == nil || *got.String != "north" {
		t.Fatalf("get attributes[azimuth] = %#v, want StringValue(north)", got)
	}
	gotRels, ok := content["relationships"].([]domain.CIRelationship)
	if !ok {
		t.Fatalf("get relationships type = %T, want []domain.CIRelationship (content=%#v)", content["relationships"], content)
	}
	if len(gotRels) != 1 || gotRels[0].TargetCIID != "SRV-DB-001" || gotRels[0].Kind != "hosts" {
		t.Fatalf("get relationships = %#v, want seeded relationship", gotRels)
	}
}

func TestMCPGetCIMetadataMissingReturnsEmpty(t *testing.T) {
	configDir := t.TempDir()
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolGetCIMetadata].Handler, map[string]any{"ci_id": "FW-MAIN-001"})
	if result.IsError {
		t.Fatalf("get result is error: %s", resultText(result))
	}
	content := result.StructuredContent.(map[string]any)
	if got := content["ci_id"]; got != "FW-MAIN-001" {
		t.Fatalf("get ci_id = %v, want FW-MAIN-001 (echoed even when missing)", got)
	}
	attrs, _ := content["attributes"].(map[string]domain.TypedValue)
	if len(attrs) != 0 {
		t.Fatalf("get attributes = %#v, want empty", attrs)
	}
	rels, _ := content["relationships"].([]domain.CIRelationship)
	if len(rels) != 0 {
		t.Fatalf("get relationships = %#v, want empty", rels)
	}
}

func TestMCPGetCIMetadataMissingCIIDArgs(t *testing.T) {
	srv := NewServer(ServerConfig{ConfigDir: t.TempDir()})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolGetCIMetadata].Handler, map[string]any{})
	if !result.IsError {
		t.Fatalf("get result IsError = false, want true (result=%#v)", result)
	}
	if !strings.Contains(resultText(result), "ci_id") {
		t.Fatalf("get error text = %q, want mention of ci_id", resultText(result))
	}
}

func TestMCPSetCIMetadataCreatesEntry(t *testing.T) {
	configDir := t.TempDir()
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolSetCIMetadata].Handler, map[string]any{
		"ci_id": "FW-MAIN-001",
		"attributes": map[string]any{
			"azimuth": map[string]any{"string": "north"},
			"rssi":    map[string]any{"number": -42.0},
		},
		"relationships": []any{
			map[string]any{"target_ci_id": "SRV-DB-001", "kind": "hosts"},
		},
	})
	if result.IsError {
		t.Fatalf("set result is error: %s", resultText(result))
	}
	content := result.StructuredContent.(map[string]any)
	if got := content["ci_id"]; got != "FW-MAIN-001" {
		t.Fatalf("set ci_id = %v, want FW-MAIN-001", got)
	}
	attrs, ok := content["attributes"].(map[string]domain.TypedValue)
	if !ok {
		t.Fatalf("set attributes type = %T, want map[string]domain.TypedValue (content=%#v)", content["attributes"], content)
	}
	if got := attrs["azimuth"]; got.String == nil || *got.String != "north" {
		t.Fatalf("set attributes[azimuth] = %#v, want StringValue(north)", got)
	}
	if got := attrs["rssi"]; got.Number == nil || *got.Number != -42.0 {
		t.Fatalf("set attributes[rssi] = %#v, want NumberValue(-42)", got)
	}
	rels, ok := content["relationships"].([]domain.CIRelationship)
	if !ok {
		t.Fatalf("set relationships type = %T, want []domain.CIRelationship (content=%#v)", content["relationships"], content)
	}
	if len(rels) != 1 || rels[0].TargetCIID != "SRV-DB-001" || rels[0].Kind != "hosts" {
		t.Fatalf("set relationships = %#v, want seeded relationship", rels)
	}

	stored, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(stored.Entries) != 1 || stored.Entries[0].CIID != "FW-MAIN-001" {
		t.Fatalf("stored entries = %#v, want single FW-MAIN-001 entry", stored.Entries)
	}
}

func TestMCPSetCIMetadataReplacesAttributes(t *testing.T) {
	configDir := t.TempDir()
	seedAttrs := map[string]domain.TypedValue{"a": domain.StringValue("x"), "b": domain.NumberValue(2)}
	seedRels := []domain.CIRelationship{{TargetCIID: "SRV-DB-001", Kind: "hosts"}}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: seedAttrs, Relationships: seedRels}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolSetCIMetadata].Handler, map[string]any{
		"ci_id": "FW-MAIN-001",
		"attributes": map[string]any{
			"c": map[string]any{"bool": true},
		},
	})
	if result.IsError {
		t.Fatalf("set result is error: %s", resultText(result))
	}
	content := result.StructuredContent.(map[string]any)
	attrs, ok := content["attributes"].(map[string]domain.TypedValue)
	if !ok {
		t.Fatalf("set attributes type = %T, want map[string]domain.TypedValue", content["attributes"])
	}
	if _, present := attrs["a"]; present {
		t.Fatalf("set attributes still has a = %#v, want removed", attrs["a"])
	}
	if _, present := attrs["b"]; present {
		t.Fatalf("set attributes still has b = %#v, want removed", attrs["b"])
	}
	if val, ok := attrs["c"]; !ok || val.Bool == nil || !*val.Bool {
		t.Fatalf("set attributes[c] = %#v, want BoolValue(true)", val)
	}
	rels, ok := content["relationships"].([]domain.CIRelationship)
	if !ok {
		t.Fatalf("set relationships type = %T, want []domain.CIRelationship", content["relationships"])
	}
	if len(rels) != 1 || rels[0].TargetCIID != "SRV-DB-001" {
		t.Fatalf("set relationships = %#v, want preserved seeded relationship", rels)
	}
}

func TestMCPSetCIMetadataNilAttributesPreservesExisting(t *testing.T) {
	configDir := t.TempDir()
	seedAttrs := map[string]domain.TypedValue{"a": domain.StringValue("x")}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: seedAttrs}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolSetCIMetadata].Handler, map[string]any{
		"ci_id": "FW-MAIN-001",
		"relationships": []any{
			map[string]any{"target_ci_id": "SRV-DB-001", "kind": "hosts"},
		},
	})
	if result.IsError {
		t.Fatalf("set result is error: %s", resultText(result))
	}
	content := result.StructuredContent.(map[string]any)
	attrs, ok := content["attributes"].(map[string]domain.TypedValue)
	if !ok {
		t.Fatalf("set attributes type = %T, want map[string]domain.TypedValue", content["attributes"])
	}
	if got := attrs["a"]; got.String == nil || *got.String != "x" {
		t.Fatalf("set attributes[a] = %#v, want preserved StringValue(x)", got)
	}
	rels, ok := content["relationships"].([]domain.CIRelationship)
	if !ok || len(rels) != 1 || rels[0].TargetCIID != "SRV-DB-001" {
		t.Fatalf("set relationships = %#v, want seeded relationship", rels)
	}
}

func TestMCPSetCIMetadataEmptyAttributesClears(t *testing.T) {
	configDir := t.TempDir()
	seedAttrs := map[string]domain.TypedValue{"a": domain.StringValue("x"), "b": domain.NumberValue(2)}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: seedAttrs}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolSetCIMetadata].Handler, map[string]any{
		"ci_id":      "FW-MAIN-001",
		"attributes": map[string]any{},
	})
	if result.IsError {
		t.Fatalf("set result is error: %s", resultText(result))
	}
	content := result.StructuredContent.(map[string]any)
	attrs, ok := content["attributes"].(map[string]domain.TypedValue)
	if !ok {
		t.Fatalf("set attributes type = %T, want map[string]domain.TypedValue", content["attributes"])
	}
	if len(attrs) != 0 {
		t.Fatalf("set attributes = %#v, want empty", attrs)
	}
}

func TestMCPSetCIMetadataReturnsErrorOnInvalidTypedValue(t *testing.T) {
	configDir := t.TempDir()
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolSetCIMetadata].Handler, map[string]any{
		"ci_id": "FW-MAIN-001",
		"attributes": map[string]any{
			"bad": map[string]any{"number": 1.0, "bool": true},
		},
	})
	if !result.IsError {
		t.Fatalf("set result IsError = false, want true (result=%#v)", result)
	}
	if !strings.Contains(resultText(result), "typed value") {
		t.Fatalf("set error text = %q, want mention of typed value validation", resultText(result))
	}
}

func TestMCPSetCIMetadataReturnsErrorOnSelfReferentialRelationship(t *testing.T) {
	configDir := t.TempDir()
	srv := NewServer(ServerConfig{ConfigDir: configDir})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolSetCIMetadata].Handler, map[string]any{
		"ci_id": "FW-MAIN-001",
		"relationships": []any{
			map[string]any{"target_ci_id": "FW-MAIN-001", "kind": "self-ref"},
		},
	})
	if !result.IsError {
		t.Fatalf("set result IsError = false, want true (result=%#v)", result)
	}
	if !strings.Contains(resultText(result), "own ci_id") {
		t.Fatalf("set error text = %q, want mention of self-referential relationship", resultText(result))
	}
}

func TestMCPSetCIMetadataReturnsErrorOnMissingCIID(t *testing.T) {
	srv := NewServer(ServerConfig{ConfigDir: t.TempDir()})
	tools := srv.ListTools()

	result := callTool(t, tools[ToolSetCIMetadata].Handler, map[string]any{
		"attributes": map[string]any{"a": map[string]any{"string": "x"}},
	})
	if !result.IsError {
		t.Fatalf("set result IsError = false, want true (result=%#v)", result)
	}
	if !strings.Contains(resultText(result), "ci id") {
		t.Fatalf("set error text = %q, want mention of ci id", resultText(result))
	}
}

func TestToolNameConstants_NoPrefix(t *testing.T) {
	cases := []struct{ got, want string }{
		{ToolListCIs, "list_cis"},
		{ToolRecordEvent, "record_event"},
		{ToolGetTimeline, "get_timeline"},
		{ToolGetCI, "get_ci"},
		{ToolResolveCIRef, "resolve_ci_ref"},
		{ToolGetCIMetadata, "get_ci_metadata"},
		{ToolSetCIMetadata, "set_ci_metadata"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("tool constant = %q, want %q", c.got, c.want)
		}
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
