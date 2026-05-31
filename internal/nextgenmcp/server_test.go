package nextgenmcp

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"raven/internal/nextgen"
)

type fakeClient struct {
	listEventsInput string
	searchLimit     int
	err             error
}

func (f *fakeClient) ListEvents(ctx context.Context, status string) (map[string]any, string, error) {
	f.listEventsInput = status
	return map[string]any{"events": []map[string]any{{"id": "evt-1"}}}, "GET /api/events?status=" + status, f.err
}

func (f *fakeClient) GetEvent(ctx context.Context, eventID string) (map[string]any, string, error) {
	return map[string]any{"event": map[string]any{"id": eventID}}, "GET /api/events/" + eventID, f.err
}

func (f *fakeClient) SearchCIs(ctx context.Context, query string, limit int) (map[string]any, string, error) {
	f.searchLimit = limit
	return map[string]any{"cis": []map[string]any{{"id": "42"}}, "count": 1}, "GET /api/nodes/search?q=" + query, f.err
}

func (f *fakeClient) GetCIEvents(ctx context.Context, ciID string) (map[string]any, string, error) {
	return map[string]any{"ci_id": ciID, "events": []map[string]any{{"id": "evt-1"}}}, "GET /api/events/related/" + ciID, f.err
}

func (f *fakeClient) GetCIMetrics(ctx context.Context, nodeID string) (map[string]any, string, error) {
	return map[string]any{"node_id": nodeID, "metrics": []map[string]any{{"id": "packet_loss"}}}, "GET /api/nodes/" + nodeID + "/metrics", f.err
}

func (f *fakeClient) BuildRavenEventCandidate(ctx context.Context, input nextgen.EventCandidateInput) (map[string]any, string, error) {
	return map[string]any{"candidate": map[string]any{"external_id": input.EventID, "ci_id": input.CanonicalCIID}}, "GET /api/events/" + input.EventID, f.err
}

func TestNewServerRegistersReadOnlyNextGenTools(t *testing.T) {
	srv := NewServer(ServerConfig{Client: &fakeClient{}})
	got := toolNames(srv.ListTools())
	want := []string{ToolBuildRavenEventCandidate, ToolGetCIEvents, ToolGetCIMetrics, ToolGetEvent, ToolListEvents, ToolSearchCIs}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tool names = %v, want %v", got, want)
	}
	for _, name := range want {
		tool := srv.ListTools()[name].Tool
		assertStrictTopLevelSchema(t, tool)
		assertReadOnlyOpenWorldTool(t, tool)
	}
	assertRequiredFields(t, srv.ListTools()[ToolGetEvent].Tool, "event_id")
	assertRequiredFields(t, srv.ListTools()[ToolSearchCIs].Tool, "query")
	assertRequiredFields(t, srv.ListTools()[ToolGetCIEvents].Tool, "ci_id")
	assertRequiredFields(t, srv.ListTools()[ToolGetCIMetrics].Tool, "node_id")
	assertRequiredFields(t, srv.ListTools()[ToolBuildRavenEventCandidate].Tool, "event_id")
}

func TestHandlersReturnStructuredEnvelope(t *testing.T) {
	fake := &fakeClient{}
	srv := NewServer(ServerConfig{Client: fake, Now: func() time.Time { return time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC) }})
	result := callTool(t, srv.ListTools()[ToolListEvents].Handler, map[string]any{"status": "ACTIVE"})
	if result.IsError {
		t.Fatalf("list result is error: %s", resultText(result))
	}
	envelope := result.StructuredContent.(map[string]any)
	if envelope["ok"] != true || envelope["source"] != nextgen.Source || envelope["endpoint"] != "GET /api/events?status=ACTIVE" || envelope["fetched_at"] != "2026-05-30T00:00:00Z" {
		t.Fatalf("envelope = %#v, want common next-gen envelope", envelope)
	}
	data := envelope["data"].(map[string]any)
	if len(data["events"].([]map[string]any)) != 1 {
		t.Fatalf("data = %#v, want one event", data)
	}
	if fake.listEventsInput != "ACTIVE" {
		t.Fatalf("ListEvents status = %q, want ACTIVE", fake.listEventsInput)
	}
}

func TestHandlersUseArgumentsAndCandidateHelper(t *testing.T) {
	fake := &fakeClient{}
	srv := NewServer(ServerConfig{Client: fake})
	tools := srv.ListTools()
	cases := []struct {
		name string
		tool string
		args map[string]any
		key  string
	}{
		{name: "get event", tool: ToolGetEvent, args: map[string]any{"event_id": "evt-1"}, key: "event"},
		{name: "search cis", tool: ToolSearchCIs, args: map[string]any{"query": "fw-main", "limit": 5}, key: "cis"},
		{name: "ci events", tool: ToolGetCIEvents, args: map[string]any{"ci_id": "42"}, key: "events"},
		{name: "ci metrics", tool: ToolGetCIMetrics, args: map[string]any{"node_id": "42"}, key: "metrics"},
		{name: "candidate", tool: ToolBuildRavenEventCandidate, args: map[string]any{"event_id": "evt-1", "canonical_ci_id": "RAVEN-FW-MAIN-001"}, key: "candidate"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := callTool(t, tools[tt.tool].Handler, tt.args)
			if result.IsError {
				t.Fatalf("result is error: %s", resultText(result))
			}
			data := result.StructuredContent.(map[string]any)["data"].(map[string]any)
			if data[tt.key] == nil {
				t.Fatalf("data = %#v, want key %q", data, tt.key)
			}
		})
	}
	if fake.searchLimit != 5 {
		t.Fatalf("SearchCIs limit = %d, want 5", fake.searchLimit)
	}
}

func TestHandlersValidateArgumentsWithStructuredErrors(t *testing.T) {
	srv := NewServer(ServerConfig{Client: &fakeClient{}})
	tools := srv.ListTools()
	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{name: "bad status", tool: ToolListEvents, args: map[string]any{"status": "BAD"}, want: "status must be one of"},
		{name: "empty event id", tool: ToolGetEvent, args: map[string]any{"event_id": "  "}, want: "must not be empty"},
		{name: "short query", tool: ToolSearchCIs, args: map[string]any{"query": "a"}, want: "at least 2"},
		{name: "limit too low", tool: ToolSearchCIs, args: map[string]any{"query": "fw-main", "limit": 0}, want: "between 1 and 100"},
		{name: "limit too high", tool: ToolSearchCIs, args: map[string]any{"query": "fw-main", "limit": 101}, want: "between 1 and 100"},
		{name: "fractional limit", tool: ToolSearchCIs, args: map[string]any{"query": "fw-main", "limit": 1.5}, want: "must be an integer"},
		{name: "empty ci id", tool: ToolGetCIEvents, args: map[string]any{"ci_id": ""}, want: "must not be empty"},
		{name: "empty node id", tool: ToolGetCIMetrics, args: map[string]any{"node_id": ""}, want: "must not be empty"},
		{name: "empty candidate event id", tool: ToolBuildRavenEventCandidate, args: map[string]any{"event_id": ""}, want: "must not be empty"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := callTool(t, tools[tt.tool].Handler, tt.args)
			if !result.IsError || !strings.Contains(resultText(result), tt.want) {
				t.Fatalf("result = %#v text=%q, want validation error containing %q", result, resultText(result), tt.want)
			}
			structured := result.StructuredContent.(map[string]any)
			if structured["ok"] != false || structured["type"] != "validation_error" || !strings.Contains(structured["detail"].(string), tt.want) {
				t.Fatalf("structured = %#v, want structured validation error", structured)
			}
		})
	}
}

func TestHandlersReturnSetupErrorWhenUnconfigured(t *testing.T) {
	srv := NewServer(ServerConfig{SetupError: nextgen.SetupError{Missing: []string{"NEXTGEN_BASE_URL", "NEXTGEN_ACCESS_TOKEN"}}})
	result := callTool(t, srv.ListTools()[ToolListEvents].Handler, map[string]any{"status": "ACTIVE"})
	if !result.IsError || !strings.Contains(resultText(result), "NEXTGEN_BASE_URL") || !strings.Contains(resultText(result), "NEXTGEN_ACCESS_TOKEN") {
		t.Fatalf("result = %#v text=%q, want setup error", result, resultText(result))
	}
	structured := result.StructuredContent.(map[string]any)
	if structured["ok"] != false || structured["source"] != nextgen.Source {
		t.Fatalf("structured error = %#v, want next-gen error envelope", structured)
	}
}

func TestHandlersReturnReadableClientErrors(t *testing.T) {
	fake := &fakeClient{err: nextgen.HTTPError{Endpoint: "GET /api/events/evt-1", StatusCode: 404, Detail: "Event not found"}}
	srv := NewServer(ServerConfig{Client: fake})
	result := callTool(t, srv.ListTools()[ToolGetEvent].Handler, map[string]any{"event_id": "evt-1"})
	if !result.IsError || !strings.Contains(resultText(result), "Event not found") {
		t.Fatalf("result = %#v text=%q, want readable HTTP error", result, resultText(result))
	}
	structured := result.StructuredContent.(map[string]any)
	if structured["status_code"] != 404 || structured["detail"] != "Event not found" {
		t.Fatalf("structured error = %#v, want status/detail", structured)
	}

	fake.err = errors.New("network down")
	result = callTool(t, srv.ListTools()[ToolGetEvent].Handler, map[string]any{"event_id": "evt-1"})
	if !result.IsError || !strings.Contains(resultText(result), "network down") {
		t.Fatalf("result text = %q, want generic error", resultText(result))
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

func toolNames(tools map[string]*mcpserver.ServerTool) []string {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func assertStrictTopLevelSchema(t *testing.T, tool mcp.Tool) {
	t.Helper()
	if tool.InputSchema.AdditionalProperties != false {
		t.Fatalf("%s additionalProperties = %#v, want false", tool.Name, tool.InputSchema.AdditionalProperties)
	}
}

func assertReadOnlyOpenWorldTool(t *testing.T, tool mcp.Tool) {
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
	if tool.Annotations.OpenWorldHint == nil || !*tool.Annotations.OpenWorldHint {
		t.Fatalf("%s openWorldHint = %v, want true", tool.Name, tool.Annotations.OpenWorldHint)
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
