package nextgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"raven/internal/version"
)

const Source = "next-gen"

const defaultTimeout = 10 * time.Second
const maxResponseBodyBytes = 1 << 20
const maxErrorDetailBytes = 4096

type Config struct {
	BaseURL     string
	AccessToken string
	Timeout     time.Duration
	UserAgent   string
	HTTPClient  *http.Client
}

type SetupError struct {
	Missing []string
}

func (e SetupError) Error() string {
	missing := append([]string(nil), e.Missing...)
	sort.Strings(missing)
	return fmt.Sprintf("next-gen MCP is not configured: missing %s", strings.Join(missing, ", "))
}

func ConfigFromEnv() (Config, error) {
	return configFromEnv(os.Getenv)
}

func configFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{
		BaseURL:     strings.TrimSpace(getenv("NEXTGEN_BASE_URL")),
		AccessToken: strings.TrimSpace(getenv("NEXTGEN_ACCESS_TOKEN")),
		UserAgent:   strings.TrimSpace(getenv("NEXTGEN_USER_AGENT")),
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "raven-nextgen-mcp/" + version.String()
	}
	if timeout := strings.TrimSpace(getenv("NEXTGEN_TIMEOUT")); timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return Config{}, fmt.Errorf("invalid NEXTGEN_TIMEOUT: %w", err)
		}
		if parsed <= 0 {
			return Config{}, fmt.Errorf("invalid NEXTGEN_TIMEOUT: must be greater than zero")
		}
		cfg.Timeout = parsed
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}

	missing := make([]string, 0, 2)
	if cfg.BaseURL == "" {
		missing = append(missing, "NEXTGEN_BASE_URL")
	}
	if cfg.AccessToken == "" {
		missing = append(missing, "NEXTGEN_ACCESS_TOKEN")
	}
	if len(missing) > 0 {
		return Config{}, SetupError{Missing: missing}
	}
	return cfg, nil
}

type Client struct {
	baseURL     *url.URL
	accessToken string
	userAgent   string
	httpClient  *http.Client
}

func NewClient(cfg Config) (*Client, error) {
	base, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("invalid NEXTGEN_BASE_URL: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("invalid NEXTGEN_BASE_URL: absolute URL is required")
	}
	if err := validateBaseURLScheme(base); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.AccessToken) == "" {
		return nil, SetupError{Missing: []string{"NEXTGEN_ACCESS_TOKEN"}}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("invalid NEXTGEN_TIMEOUT: must be greater than zero")
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "raven-nextgen-mcp/" + version.String()
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{baseURL: base, accessToken: strings.TrimSpace(cfg.AccessToken), userAgent: cfg.UserAgent, httpClient: hc}, nil
}

func validateBaseURLScheme(base *url.URL) error {
	switch strings.ToLower(base.Scheme) {
	case "https":
		return nil
	case "http":
		if isLocalhostOrLoopback(base.Hostname()) {
			return nil
		}
		return fmt.Errorf("invalid NEXTGEN_BASE_URL: http is allowed only for localhost or loopback hosts")
	default:
		return fmt.Errorf("invalid NEXTGEN_BASE_URL: scheme must be https, or http for localhost/loopback development")
	}
}

func isLocalhostOrLoopback(host string) bool {
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type HTTPError struct {
	Endpoint   string
	StatusCode int
	Detail     string
}

func (e HTTPError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("next-gen %s returned HTTP %d", e.Endpoint, e.StatusCode)
	}
	return fmt.Sprintf("next-gen %s returned HTTP %d: %s", e.Endpoint, e.StatusCode, e.Detail)
}

func (c *Client) ListEvents(ctx context.Context, status string) (map[string]any, string, error) {
	query := url.Values{}
	if strings.TrimSpace(status) != "" {
		query.Set("status", strings.TrimSpace(status))
	}
	var events []map[string]any
	endpoint, err := c.getJSON(ctx, []string{"api", "events"}, query, &events)
	if err != nil {
		return nil, endpoint, err
	}
	return map[string]any{"events": events}, endpoint, nil
}

func (c *Client) GetEvent(ctx context.Context, eventID string) (map[string]any, string, error) {
	var detail map[string]any
	endpoint, err := c.getJSON(ctx, []string{"api", "events", strings.TrimSpace(eventID)}, nil, &detail)
	if err != nil {
		return nil, endpoint, err
	}
	return detail, endpoint, nil
}

func (c *Client) SearchCIs(ctx context.Context, queryText string, limit int) (map[string]any, string, error) {
	query := url.Values{"q": []string{strings.TrimSpace(queryText)}}
	var cis []map[string]any
	endpoint, err := c.getJSON(ctx, []string{"api", "nodes", "search"}, query, &cis)
	if err != nil {
		return nil, endpoint, err
	}
	if limit > 0 && len(cis) > limit {
		cis = cis[:limit]
	}
	return map[string]any{"cis": cis, "count": len(cis)}, endpoint, nil
}

func (c *Client) GetCIEvents(ctx context.Context, ciID string) (map[string]any, string, error) {
	var events []map[string]any
	endpoint, err := c.getJSON(ctx, []string{"api", "events", "related", strings.TrimSpace(ciID)}, nil, &events)
	if err != nil {
		return nil, endpoint, err
	}
	return map[string]any{"ci_id": strings.TrimSpace(ciID), "events": events}, endpoint, nil
}

func (c *Client) GetCIMetrics(ctx context.Context, nodeID string) (map[string]any, string, error) {
	var metrics []map[string]any
	endpoint, err := c.getJSON(ctx, []string{"api", "nodes", strings.TrimSpace(nodeID), "metrics"}, nil, &metrics)
	if err != nil {
		return nil, endpoint, err
	}
	return map[string]any{"node_id": strings.TrimSpace(nodeID), "metrics": metrics}, endpoint, nil
}

type EventCandidateInput struct {
	EventID       string
	CanonicalCIID string
}

func (c *Client) BuildRavenEventCandidate(ctx context.Context, input EventCandidateInput) (map[string]any, string, error) {
	detail, endpoint, err := c.GetEvent(ctx, input.EventID)
	if err != nil {
		return nil, endpoint, err
	}
	candidate, err := BuildRavenEventCandidate(detail, input.CanonicalCIID)
	if err != nil {
		return nil, endpoint, err
	}
	return map[string]any{"candidate": candidate}, endpoint, nil
}

func BuildRavenEventCandidate(detail map[string]any, canonicalCIID string) (map[string]any, error) {
	event, ok := detail["event"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("next-gen event detail missing event object")
	}
	eventID := firstString(event["id"])
	if eventID == "" {
		return nil, fmt.Errorf("next-gen event detail missing event.id")
	}
	summary := firstNonEmptyString(event["message"], event["summary"], event["event_type"])
	if summary == "" {
		summary = "next-gen event " + eventID
	}
	candidate := map[string]any{
		"type":        "network_alert",
		"severity":    normalizeSeverity(firstString(event["severity"])),
		"summary":     summary,
		"source":      Source,
		"external_id": eventID,
	}
	observedAt := firstNonEmptyString(event["created_at"], event["last_seen"])
	if observedAt == "" {
		return nil, fmt.Errorf("next-gen event detail missing observed timestamp")
	}
	candidate["observed_at"] = observedAt
	if details := buildDetails(event, detail); details != "" {
		candidate["details"] = details
	}
	raw, err := json.Marshal(detail)
	if err != nil {
		return nil, fmt.Errorf("encode next-gen raw detail: %w", err)
	}
	candidate["raw"] = string(raw)
	if strings.TrimSpace(canonicalCIID) != "" {
		candidate["ci_id"] = strings.TrimSpace(canonicalCIID)
		return candidate, nil
	}
	ciRef := nextGenCIRef(event)
	if ciRef == nil {
		return nil, fmt.Errorf("next-gen event detail missing CI reference")
	}
	candidate["ci_ref"] = ciRef
	return candidate, nil
}

func (c *Client) getJSON(ctx context.Context, pathSegments []string, query url.Values, out any) (string, error) {
	requestURL, endpoint, err := c.buildURL(pathSegments, query)
	if err != nil {
		return endpoint, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return endpoint, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return endpoint, err
	}
	defer resp.Body.Close()
	body, truncated, err := readLimited(resp.Body)
	if err != nil {
		return endpoint, err
	}
	if truncated && resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return endpoint, fmt.Errorf("next-gen %s response exceeds %d bytes", endpoint, maxResponseBodyBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return endpoint, HTTPError{Endpoint: endpoint, StatusCode: resp.StatusCode, Detail: responseDetail(body, truncated)}
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(out); err != nil {
		return endpoint, fmt.Errorf("decode next-gen response %s: %w", endpoint, err)
	}
	return endpoint, nil
}

func (c *Client) buildURL(pathSegments []string, query url.Values) (string, string, error) {
	escapedAPIPath := escapedPath(pathSegments)
	logicalPath := "/" + escapedAPIPath
	requestPath := joinEscapedPaths(strings.TrimRight(c.baseURL.EscapedPath(), "/"), escapedAPIPath)
	decodedPath, err := url.PathUnescape(requestPath)
	if err != nil {
		return "", "GET " + logicalPath, fmt.Errorf("build next-gen request path: %w", err)
	}
	u := *c.baseURL
	u.Path = decodedPath
	if decodedPath != requestPath {
		u.RawPath = requestPath
	} else {
		u.RawPath = ""
	}
	u.RawQuery = query.Encode()
	endpoint := "GET " + logicalPath
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	return u.String(), endpoint, nil
}

func escapedPath(segments []string) string {
	escaped := make([]string, 0, len(segments))
	for _, segment := range segments {
		escaped = append(escaped, url.PathEscape(segment))
	}
	return strings.Join(escaped, "/")
}

func joinEscapedPaths(basePath, apiPath string) string {
	if strings.TrimSpace(basePath) == "" {
		return "/" + apiPath
	}
	return basePath + "/" + apiPath
}

func readLimited(reader io.Reader) ([]byte, bool, error) {
	limited := io.LimitReader(reader, maxResponseBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if len(body) > maxResponseBodyBytes {
		return body[:maxResponseBodyBytes], true, nil
	}
	return body, false, nil
}

func responseDetail(body []byte, truncated bool) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		if detail := firstString(payload["detail"]); detail != "" {
			return truncateDetail(detail, truncated)
		}
		if message := firstString(payload["message"]); message != "" {
			return truncateDetail(message, truncated)
		}
	}
	return truncateDetail(strings.TrimSpace(string(body)), truncated)
}

func truncateDetail(detail string, bodyTruncated bool) string {
	detail = strings.TrimSpace(detail)
	truncated := bodyTruncated
	if len(detail) > maxErrorDetailBytes {
		detail = detail[:maxErrorDetailBytes]
		truncated = true
	}
	if truncated {
		return detail + "... [truncated]"
	}
	return detail
}

func nextGenCIRef(event map[string]any) map[string]any {
	if ciRef, ok := event["ci_ref"].(map[string]any); ok {
		if id := firstString(ciRef["id"]); id != "" {
			return map[string]any{"source": Source, "type": "ci_id", "value": id}
		}
		if hostname := firstString(ciRef["hostname"]); hostname != "" {
			return hostCIRef(hostname)
		}
	}
	if ciID := firstString(event["ci_id"]); ciID != "" {
		return map[string]any{"source": Source, "type": "ci_id", "value": ciID}
	}
	if hostname := firstString(event["ci_hostname"]); hostname != "" {
		return hostCIRef(hostname)
	}
	return nil
}

func hostCIRef(host string) map[string]any {
	aliasType := "hostname"
	if ip := net.ParseIP(host); ip != nil {
		aliasType = "ip"
	}
	return map[string]any{"source": Source, "type": aliasType, "value": host}
}

func normalizeSeverity(severity string) string {
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "CRITICAL", "CRIT":
		return "critical"
	case "WARNING", "WARN":
		return "warning"
	case "INFO", "INFORMATIONAL":
		return "info"
	default:
		if strings.TrimSpace(severity) == "" {
			return "info"
		}
		return strings.ToLower(strings.TrimSpace(severity))
	}
}

func buildDetails(event map[string]any, detail map[string]any) string {
	lines := make([]string, 0, 8)
	for _, key := range []string{"status", "metric_name", "metric_protocol", "event_type", "source_protocol", "last_seen"} {
		if value := firstString(event[key]); value != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", key, value))
		}
	}
	if businessContext, ok := detail["business_context"]; ok && businessContext != nil {
		if encoded, err := json.Marshal(businessContext); err == nil && string(encoded) != "null" && string(encoded) != "{}" {
			lines = append(lines, "business_context: "+string(encoded))
		}
	}
	if itsmContext, ok := detail["itsm_context"]; ok && itsmContext != nil {
		if encoded, err := json.Marshal(itsmContext); err == nil && string(encoded) != "null" && string(encoded) != "{}" {
			lines = append(lines, "itsm_context: "+string(encoded))
		}
	}
	return strings.Join(lines, "\n")
}

func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		if text := firstString(value); text != "" {
			return text
		}
	}
	return ""
}

func firstString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
