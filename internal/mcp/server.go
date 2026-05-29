package ravenmcp

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"raven/internal/domain"
	"raven/internal/service"
	"raven/internal/version"
)

const (
	ToolResolveCIRef = "raven_resolve_ci_ref"
	ToolRecordEvent  = "raven_record_event"
	ToolGetTimeline  = "raven_get_timeline"
	ToolListCIs      = "raven_list_cis"
	ToolGetCI        = "raven_get_ci"
)

type ServerConfig struct {
	ConfigDir string
}

func ServeStdio(args []string, configDir string, stderr io.Writer) error {
	flags := flag.NewFlagSet("mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("mcp does not accept arguments: %s", strings.Join(flags.Args(), " "))
		fmt.Fprintln(stderr, err)
		return err
	}

	return mcpserver.ServeStdio(NewServer(ServerConfig{ConfigDir: configDir}))
}

func NewServer(config ServerConfig) *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer(
		"raven",
		version.String(),
		mcpserver.WithToolCapabilities(false),
		mcpserver.WithRecovery(),
	)
	registerTools(srv, service.New(config.ConfigDir))
	return srv
}

func ToolNames(srv *mcpserver.MCPServer) []string {
	tools := srv.ListTools()
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func registerTools(srv *mcpserver.MCPServer, svc service.Service) {
	srv.AddTool(mcp.NewTool(ToolResolveCIRef,
		mcp.WithDescription("Resolve an explicit upstream CI reference (source + type + value) to Raven's canonical ci_id."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithSchemaAdditionalProperties(false),
		mcp.WithString("source", mcp.Required(), mcp.Description("Alias source namespace, for example next-gen.")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Alias type."), mcp.Enum(string(domain.AliasTypeCIID), string(domain.AliasTypeIP), string(domain.AliasTypeHostname), string(domain.AliasTypeSerial), string(domain.AliasTypeMAC))),
		mcp.WithString("value", mcp.Required(), mcp.Description("Alias value to resolve.")),
	), handleResolveCIRef(svc))

	srv.AddTool(mcp.NewTool(ToolRecordEvent,
		mcp.WithDescription("Record a Raven event using either canonical ci_id or a ci_ref alias object. Upstream IDs must be passed as ci_ref aliases, not as canonical ci_id."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithSchemaAdditionalProperties(false),
		mcp.WithString("ci_id", mcp.Description("Canonical Raven CI ID when already known.")),
		mcp.WithObject("ci_ref", mcp.Description("Alias lookup object used when canonical ci_id is unknown."), mcp.Properties(ciRefSchemaProperties()), objectRequired("source", "type", "value"), mcp.AdditionalProperties(false)),
		mcp.WithString("id", mcp.Description("Optional event ID. Raven generates one when omitted.")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Event type, for example observation, diagnosis, network_alert, incident, resolution.")),
		mcp.WithString("severity", mcp.Required(), mcp.Description("Event severity, for example info, warning, critical.")),
		mcp.WithString("status", mcp.Description("Optional event status. Defaults to open.")),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Short operator-readable summary.")),
		mcp.WithString("details", mcp.Description("Longer diagnostic text or explanation.")),
		mcp.WithString("source", mcp.Required(), mcp.Description("Producer name, for example gemini-cli, antigravity, ollama, next-gen, or human.")),
		mcp.WithString("external_id", mcp.Description("Stable source event ID. Required unless dedup_key is provided.")),
		mcp.WithString("dedup_key", mcp.Description("Stable replay-prevention key. Required unless external_id is provided.")),
		mcp.WithString("observed_at", mcp.Required(), mcp.Description("RFC3339 timestamp for when the event was observed.")),
		mcp.WithString("ingested_at", mcp.Description("Optional RFC3339 ingest timestamp. Raven fills it when omitted.")),
		mcp.WithString("raw", mcp.Description("Raw source evidence when useful.")),
	), handleRecordEvent(svc))

	srv.AddTool(mcp.NewTool(ToolGetTimeline,
		mcp.WithDescription("Read timeline events for a canonical Raven CI ID."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithSchemaAdditionalProperties(false),
		mcp.WithString("ci_id", mcp.Required(), mcp.Description("Canonical Raven CI ID.")),
	), handleGetTimeline(svc))

	srv.AddTool(mcp.NewTool(ToolListCIs,
		mcp.WithDescription("List known Raven configuration items."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithSchemaAdditionalProperties(false),
	), handleListCIs(svc))

	srv.AddTool(mcp.NewTool(ToolGetCI,
		mcp.WithDescription("Get one known Raven configuration item by canonical CI ID."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithSchemaAdditionalProperties(false),
		mcp.WithString("ci_id", mcp.Required(), mcp.Description("Canonical Raven CI ID.")),
	), handleGetCI(svc))
}

func ciRefSchemaProperties() map[string]any {
	return map[string]any{
		"source": map[string]any{"type": "string", "description": "Alias source namespace, for example next-gen."},
		"type": map[string]any{
			"type":        "string",
			"description": "Alias type.",
			"enum":        []string{string(domain.AliasTypeCIID), string(domain.AliasTypeIP), string(domain.AliasTypeHostname), string(domain.AliasTypeSerial), string(domain.AliasTypeMAC)},
		},
		"value": map[string]any{"type": "string", "description": "Alias value to resolve."},
	}
}

func objectRequired(fields ...string) mcp.PropertyOption {
	return func(schema map[string]any) {
		schema["required"] = fields
	}
}

func handleResolveCIRef(svc service.Service) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		source, err := request.RequireString("source")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		aliasType, err := request.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		value, err := request.RequireString("value")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		ciID, err := svc.ResolveCIRef(service.CIRef{Source: source, Type: domain.AliasType(aliasType), Value: value})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructured(map[string]any{"ci_id": ciID}, ciID), nil
	}
}

func handleRecordEvent(svc service.Service) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input service.RecordEventInput
		if err := request.BindArguments(&input); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid record event arguments: %v", err)), nil
		}
		event, err := svc.RecordEvent(input)
		if err != nil {
			return mcp.NewToolResultError(readableRecordEventError(err)), nil
		}
		return mcp.NewToolResultStructuredOnly(map[string]any{"event": event}), nil
	}
}

func handleGetTimeline(svc service.Service) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ciID, err := request.RequireString("ci_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		events, err := svc.Timeline(ciID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructuredOnly(map[string]any{"ci_id": strings.TrimSpace(ciID), "events": events}), nil
	}
}

func handleListCIs(svc service.Service) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		components, err := svc.ListCIs()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructuredOnly(map[string]any{"cis": components}), nil
	}
}

func handleGetCI(svc service.Service) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ciID, err := request.RequireString("ci_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		component, err := svc.GetCI(ciID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultStructuredOnly(map[string]any{"ci": component}), nil
	}
}

func readableRecordEventError(err error) string {
	switch {
	case errors.Is(err, service.ErrMissingEventIdentity):
		return "raven_record_event requires ci_id or ci_ref"
	case errors.Is(err, service.ErrMissingEventDedup):
		return "raven_record_event requires external_id or dedup_key"
	default:
		return err.Error()
	}
}
