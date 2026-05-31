package nextgenmcp

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"raven/internal/nextgen"
	"raven/internal/version"
)

const (
	ToolListEvents               = "nextgen_list_events"
	ToolGetEvent                 = "nextgen_get_event"
	ToolSearchCIs                = "nextgen_search_cis"
	ToolGetCIEvents              = "nextgen_get_ci_events"
	ToolGetCIMetrics             = "nextgen_get_ci_metrics"
	ToolBuildRavenEventCandidate = "nextgen_build_raven_event_candidate"
)

type Client interface {
	ListEvents(ctx context.Context, status string) (map[string]any, string, error)
	GetEvent(ctx context.Context, eventID string) (map[string]any, string, error)
	SearchCIs(ctx context.Context, query string, limit int) (map[string]any, string, error)
	GetCIEvents(ctx context.Context, ciID string) (map[string]any, string, error)
	GetCIMetrics(ctx context.Context, nodeID string) (map[string]any, string, error)
	BuildRavenEventCandidate(ctx context.Context, input nextgen.EventCandidateInput) (map[string]any, string, error)
}

type ServerConfig struct {
	Client     Client
	SetupError error
	Now        func() time.Time
}

func ServeStdio(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("nextgen-mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("nextgen-mcp does not accept arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}
	cfg, err := nextgen.ConfigFromEnv()
	serverConfig := ServerConfig{}
	if err != nil {
		serverConfig.SetupError = err
	} else {
		client, err := nextgen.NewClient(cfg)
		if err != nil {
			serverConfig.SetupError = err
		} else {
			serverConfig.Client = client
		}
	}
	return mcpserver.ServeStdio(NewServer(serverConfig))
}

func NewServer(config ServerConfig) *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer(
		"raven-nextgen",
		version.String(),
		mcpserver.WithToolCapabilities(false),
		mcpserver.WithRecovery(),
	)
	registerTools(srv, config)
	return srv
}

func registerTools(srv *mcpserver.MCPServer, config ServerConfig) {
	srv.AddTool(nextGenTool(ToolListEvents,
		"List next-gen events by status.",
		mcp.WithString("status", mcp.Description("Optional next-gen event status."), mcp.Enum("OPEN", "ACK", "CLOSED", "RECOVERED", "ACTIVE", "CONSOLE")),
	), handleListEvents(config))

	srv.AddTool(nextGenTool(ToolGetEvent,
		"Fetch one next-gen event detail by event ID.",
		mcp.WithString("event_id", mcp.Required(), mcp.Description("next-gen event ID.")),
	), handleGetEvent(config))

	srv.AddTool(nextGenTool(ToolSearchCIs,
		"Search next-gen CIs by text, IP, hostname, or label.",
		mcp.WithString("query", mcp.Required(), mcp.MinLength(2), mcp.Description("Search text, IP, hostname, or CI label.")),
		mcp.WithNumber("limit", mcp.Min(1), mcp.Max(100), mcp.Description("Optional maximum number of CIs to return.")),
	), handleSearchCIs(config))

	srv.AddTool(nextGenTool(ToolGetCIEvents,
		"List active/open next-gen events for one CI.",
		mcp.WithString("ci_id", mcp.Required(), mcp.Description("next-gen CI ID.")),
	), handleGetCIEvents(config))

	srv.AddTool(nextGenTool(ToolGetCIMetrics,
		"Read applicable next-gen metrics for one CI.",
		mcp.WithString("node_id", mcp.Required(), mcp.Description("next-gen node/CI ID.")),
	), handleGetCIMetrics(config))

	srv.AddTool(nextGenTool(ToolBuildRavenEventCandidate,
		"Build a Raven event candidate from next-gen event detail without persisting it.",
		mcp.WithString("event_id", mcp.Required(), mcp.Description("next-gen event ID.")),
		mcp.WithString("canonical_ci_id", mcp.Description("Optional canonical Raven CI ID. When omitted, candidate uses ci_ref.")),
	), handleBuildRavenEventCandidate(config))
}

func nextGenTool(name string, description string, opts ...mcp.ToolOption) mcp.Tool {
	base := []mcp.ToolOption{
		mcp.WithDescription(description),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithSchemaAdditionalProperties(false),
	}
	base = append(base, opts...)
	return mcp.NewTool(name, base...)
}

func handleListEvents(config ServerConfig) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, ok := configuredClient(config)
		if !ok {
			return setupErrorResult(config.SetupError), nil
		}
		status := optionalString(request.GetString("status", ""))
		if status != "" && !validStatus(status) {
			return validationErrorResult("status must be one of OPEN, ACK, CLOSED, RECOVERED, ACTIVE, CONSOLE"), nil
		}
		data, endpoint, err := client.ListEvents(ctx, status)
		return toolResult(endpoint, data, err, config), nil
	}
}

func handleGetEvent(config ServerConfig) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, ok := configuredClient(config)
		if !ok {
			return setupErrorResult(config.SetupError), nil
		}
		eventID, err := requireNonEmptyString(request, "event_id")
		if err != nil {
			return validationErrorResult(err.Error()), nil
		}
		data, endpoint, err := client.GetEvent(ctx, eventID)
		return toolResult(endpoint, data, err, config), nil
	}
}

func handleSearchCIs(config ServerConfig) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, ok := configuredClient(config)
		if !ok {
			return setupErrorResult(config.SetupError), nil
		}
		query, err := requireNonEmptyString(request, "query")
		if err != nil {
			return validationErrorResult(err.Error()), nil
		}
		if len(query) < 2 {
			return validationErrorResult("query must be at least 2 characters"), nil
		}
		limit, err := optionalLimit(request)
		if err != nil {
			return validationErrorResult(err.Error()), nil
		}
		data, endpoint, err := client.SearchCIs(ctx, query, limit)
		return toolResult(endpoint, data, err, config), nil
	}
}

func handleGetCIEvents(config ServerConfig) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, ok := configuredClient(config)
		if !ok {
			return setupErrorResult(config.SetupError), nil
		}
		ciID, err := requireNonEmptyString(request, "ci_id")
		if err != nil {
			return validationErrorResult(err.Error()), nil
		}
		data, endpoint, err := client.GetCIEvents(ctx, ciID)
		return toolResult(endpoint, data, err, config), nil
	}
}

func handleGetCIMetrics(config ServerConfig) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, ok := configuredClient(config)
		if !ok {
			return setupErrorResult(config.SetupError), nil
		}
		nodeID, err := requireNonEmptyString(request, "node_id")
		if err != nil {
			return validationErrorResult(err.Error()), nil
		}
		data, endpoint, err := client.GetCIMetrics(ctx, nodeID)
		return toolResult(endpoint, data, err, config), nil
	}
}

func handleBuildRavenEventCandidate(config ServerConfig) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client, ok := configuredClient(config)
		if !ok {
			return setupErrorResult(config.SetupError), nil
		}
		eventID, err := requireNonEmptyString(request, "event_id")
		if err != nil {
			return validationErrorResult(err.Error()), nil
		}
		canonicalCIID := optionalString(request.GetString("canonical_ci_id", ""))
		data, endpoint, err := client.BuildRavenEventCandidate(ctx, nextgen.EventCandidateInput{EventID: eventID, CanonicalCIID: canonicalCIID})
		return toolResult(endpoint, data, err, config), nil
	}
}

func configuredClient(config ServerConfig) (Client, bool) {
	if config.Client == nil || config.SetupError != nil {
		return nil, false
	}
	return config.Client, true
}

func setupErrorResult(err error) *mcp.CallToolResult {
	if err == nil {
		err = errors.New("next-gen MCP is not configured")
	}
	structured := map[string]any{
		"ok":     false,
		"source": nextgen.Source,
		"detail": err.Error(),
	}
	return errorResult(structured, err.Error())
}

func validationErrorResult(detail string) *mcp.CallToolResult {
	structured := map[string]any{
		"ok":     false,
		"source": nextgen.Source,
		"detail": detail,
		"type":   "validation_error",
	}
	return errorResult(structured, detail)
}

func toolResult(endpoint string, data map[string]any, err error, config ServerConfig) *mcp.CallToolResult {
	if err != nil {
		structured := map[string]any{"ok": false, "source": nextgen.Source, "endpoint": endpoint, "detail": err.Error()}
		var httpErr nextgen.HTTPError
		if errors.As(err, &httpErr) {
			structured["status_code"] = httpErr.StatusCode
			structured["detail"] = httpErr.Detail
		}
		return errorResult(structured, err.Error())
	}
	envelope := map[string]any{
		"ok":         true,
		"source":     nextgen.Source,
		"endpoint":   endpoint,
		"fetched_at": now(config).Format(time.RFC3339),
		"data":       data,
	}
	return mcp.NewToolResultStructuredOnly(envelope)
}

func errorResult(structured map[string]any, text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content:           []mcp.Content{mcp.TextContent{Type: mcp.ContentTypeText, Text: text}},
		StructuredContent: structured,
		IsError:           true,
	}
}

func now(config ServerConfig) time.Time {
	if config.Now != nil {
		return config.Now().UTC()
	}
	return time.Now().UTC()
}

func optionalString(value string) string {
	return strings.TrimSpace(value)
}

func requireNonEmptyString(request mcp.CallToolRequest, key string) (string, error) {
	value, err := request.RequireString(key)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("argument %q must not be empty", key)
	}
	return value, nil
}

func optionalLimit(request mcp.CallToolRequest) (int, error) {
	args := request.GetArguments()
	if args == nil {
		return 0, nil
	}
	if _, ok := args["limit"]; !ok {
		return 0, nil
	}
	limit, err := request.RequireFloat("limit")
	if err != nil {
		return 0, err
	}
	if limit != float64(int(limit)) {
		return 0, fmt.Errorf("argument %q must be an integer", "limit")
	}
	if limit < 1 || limit > 100 {
		return 0, fmt.Errorf("argument %q must be between 1 and 100", "limit")
	}
	return int(limit), nil
}

func validStatus(status string) bool {
	switch status {
	case "OPEN", "ACK", "CLOSED", "RECOVERED", "ACTIVE", "CONSOLE":
		return true
	default:
		return false
	}
}
