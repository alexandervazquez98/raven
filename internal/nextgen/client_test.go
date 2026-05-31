package nextgen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConfigFromEnvRequiresBaseURLAndToken(t *testing.T) {
	_, err := configFromEnv(func(string) string { return "" })
	if err == nil {
		t.Fatal("configFromEnv() error = nil, want missing env error")
	}
	if got := err.Error(); !strings.Contains(got, "NEXTGEN_BASE_URL") || !strings.Contains(got, "NEXTGEN_ACCESS_TOKEN") {
		t.Fatalf("configFromEnv() error = %q, want both missing env names", got)
	}
}

func TestConfigFromEnvRejectsNonPositiveTimeout(t *testing.T) {
	_, err := configFromEnv(func(key string) string {
		switch key {
		case "NEXTGEN_BASE_URL":
			return "https://nextgen.example.test"
		case "NEXTGEN_ACCESS_TOKEN":
			return "token"
		case "NEXTGEN_TIMEOUT":
			return "-1s"
		default:
			return ""
		}
	})
	if err == nil || !strings.Contains(err.Error(), "NEXTGEN_TIMEOUT") || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("configFromEnv() error = %v, want non-positive timeout error", err)
	}
}

func TestNewClientRejectsPlainHTTPExceptLoopback(t *testing.T) {
	if _, err := NewClient(Config{BaseURL: "http://example.com", AccessToken: "token"}); err == nil || !strings.Contains(err.Error(), "http is allowed only for localhost") {
		t.Fatalf("NewClient(remote http) error = %v, want plaintext rejection", err)
	}
	for _, baseURL := range []string{"http://localhost:8000", "http://127.0.0.1:8000", "http://[::1]:8000"} {
		t.Run(baseURL, func(t *testing.T) {
			if _, err := NewClient(Config{BaseURL: baseURL, AccessToken: "token"}); err != nil {
				t.Fatalf("NewClient(%q) error = %v, want nil", baseURL, err)
			}
		})
	}
}

func TestClientReadMethodsUseNextGenRESTAPI(t *testing.T) {
	requests := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
			t.Fatalf("Authorization = %q, want %q", got, want)
		}
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.RequestURI() {
		case "/api/events?status=ACTIVE":
			json.NewEncoder(w).Encode([]map[string]any{{"id": "evt-1", "ci_id": "42"}})
		case "/api/events/evt-1":
			json.NewEncoder(w).Encode(map[string]any{"event": map[string]any{"id": "evt-1", "ci_id": "42", "severity": "WARNING", "message": "High packet loss", "created_at": "2026-05-30T00:00:00Z"}})
		case "/api/nodes/search?q=fw-main":
			json.NewEncoder(w).Encode([]map[string]any{{"id": "42"}, {"id": "43"}})
		case "/api/events/related/42":
			json.NewEncoder(w).Encode([]map[string]any{{"id": "evt-1"}})
		case "/api/nodes/42/metrics":
			json.NewEncoder(w).Encode([]map[string]any{{"id": "packet_loss"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, AccessToken: "test-token", Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if data, endpoint, err := client.ListEvents(context.Background(), "ACTIVE"); err != nil || endpoint != "GET /api/events?status=ACTIVE" || len(data["events"].([]map[string]any)) != 1 {
		t.Fatalf("ListEvents() data=%#v endpoint=%q err=%v, want one event", data, endpoint, err)
	}
	if data, endpoint, err := client.GetEvent(context.Background(), "evt-1"); err != nil || endpoint != "GET /api/events/evt-1" || data["event"] == nil {
		t.Fatalf("GetEvent() data=%#v endpoint=%q err=%v, want event detail", data, endpoint, err)
	}
	if data, _, err := client.SearchCIs(context.Background(), "fw-main", 1); err != nil || data["count"] != 1 {
		t.Fatalf("SearchCIs() data=%#v err=%v, want limited count 1", data, err)
	}
	if data, _, err := client.GetCIEvents(context.Background(), "42"); err != nil || data["ci_id"] != "42" || len(data["events"].([]map[string]any)) != 1 {
		t.Fatalf("GetCIEvents() data=%#v err=%v, want one related event", data, err)
	}
	if data, _, err := client.GetCIMetrics(context.Background(), "42"); err != nil || data["node_id"] != "42" || len(data["metrics"].([]map[string]any)) != 1 {
		t.Fatalf("GetCIMetrics() data=%#v err=%v, want one metric", data, err)
	}

	wantRequests := []string{
		"GET /api/events?status=ACTIVE",
		"GET /api/events/evt-1",
		"GET /api/nodes/search?q=fw-main",
		"GET /api/events/related/42",
		"GET /api/nodes/42/metrics",
	}
	if strings.Join(requests, "\n") != strings.Join(wantRequests, "\n") {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestClientPreservesBasePathAndEscapesPathParametersOnce(t *testing.T) {
	var gotRequestURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestURI = r.URL.RequestURI()
		json.NewEncoder(w).Encode(map[string]any{"event": map[string]any{"id": "evt/with space%"}})
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL + "/proxy", AccessToken: "test-token", Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	_, endpoint, err := client.GetEvent(context.Background(), "evt/with space%")
	if err != nil {
		t.Fatalf("GetEvent() error = %v, want nil", err)
	}
	if got, want := gotRequestURI, "/proxy/api/events/evt%2Fwith%20space%25"; got != want {
		t.Fatalf("request URI = %q, want %q", got, want)
	}
	if got, want := endpoint, "GET /api/events/evt%2Fwith%20space%25"; got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}
}

func TestClientReturnsReadableHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"detail": "Not authorized to view events"})
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, AccessToken: "test-token", Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}
	_, _, err = client.GetEvent(context.Background(), "evt-1")
	httpErr, ok := err.(HTTPError)
	if !ok {
		t.Fatalf("GetEvent() error = %T %v, want HTTPError", err, err)
	}
	if httpErr.StatusCode != http.StatusForbidden || httpErr.Detail != "Not authorized to view events" {
		t.Fatalf("HTTPError = %#v, want status/detail", httpErr)
	}
}

func TestClientCapsResponseBodiesAndTruncatesErrors(t *testing.T) {
	t.Run("success body too large", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(strings.Repeat("x", maxResponseBodyBytes+1)))
		}))
		defer server.Close()
		client, err := NewClient(Config{BaseURL: server.URL, AccessToken: "test-token", Timeout: time.Second})
		if err != nil {
			t.Fatalf("NewClient() error = %v, want nil", err)
		}
		_, _, err = client.GetEvent(context.Background(), "evt-1")
		if err == nil || !strings.Contains(err.Error(), "response exceeds") {
			t.Fatalf("GetEvent() error = %v, want response size error", err)
		}
	})

	t.Run("error detail truncated", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(strings.Repeat("e", maxResponseBodyBytes+100)))
		}))
		defer server.Close()
		client, err := NewClient(Config{BaseURL: server.URL, AccessToken: "test-token", Timeout: time.Second})
		if err != nil {
			t.Fatalf("NewClient() error = %v, want nil", err)
		}
		_, _, err = client.GetEvent(context.Background(), "evt-1")
		httpErr, ok := err.(HTTPError)
		if !ok {
			t.Fatalf("GetEvent() error = %T %v, want HTTPError", err, err)
		}
		if !strings.HasSuffix(httpErr.Detail, "... [truncated]") || len(httpErr.Detail) > maxErrorDetailBytes+len("... [truncated]") {
			t.Fatalf("HTTPError detail length/suffix = %d/%q, want truncated detail", len(httpErr.Detail), httpErr.Detail[len(httpErr.Detail)-20:])
		}
	})
}

func TestBuildRavenEventCandidateUsesCIRefOrCanonicalCI(t *testing.T) {
	detail := map[string]any{
		"event": map[string]any{
			"id":         "evt-1",
			"severity":   "WARNING",
			"message":    "High packet loss detected",
			"created_at": "2026-05-30T00:00:00Z",
			"ci_ref":     map[string]any{"id": "42", "hostname": "10.53.1.22"},
		},
		"business_context": map[string]any{"source": "resolved"},
	}
	candidate, err := BuildRavenEventCandidate(detail, "")
	if err != nil {
		t.Fatalf("BuildRavenEventCandidate() error = %v, want nil", err)
	}
	if candidate["source"] != Source || candidate["external_id"] != "evt-1" || candidate["severity"] != "warning" || candidate["observed_at"] != "2026-05-30T00:00:00Z" {
		t.Fatalf("candidate = %#v, want normalized Raven fields", candidate)
	}
	ciRef := candidate["ci_ref"].(map[string]any)
	if ciRef["source"] != Source || ciRef["type"] != "ci_id" || ciRef["value"] != "42" {
		t.Fatalf("ci_ref = %#v, want next-gen ci_id ref", ciRef)
	}
	candidate, err = BuildRavenEventCandidate(detail, "RAVEN-FW-MAIN-001")
	if err != nil {
		t.Fatalf("BuildRavenEventCandidate(canonical) error = %v, want nil", err)
	}
	if candidate["ci_id"] != "RAVEN-FW-MAIN-001" {
		t.Fatalf("candidate ci_id = %#v, want canonical Raven CI", candidate["ci_id"])
	}
	if _, ok := candidate["ci_ref"]; ok {
		t.Fatalf("candidate = %#v, must not include ci_ref when ci_id is present", candidate)
	}
}

func TestBuildRavenEventCandidateClassifiesHostFallbackAsIPOrHostname(t *testing.T) {
	for _, tt := range []struct {
		name      string
		host      string
		wantType  string
		fieldName string
	}{
		{name: "ip from ci_ref hostname", host: "10.53.1.22", wantType: "ip", fieldName: "ci_ref"},
		{name: "hostname from ci_ref hostname", host: "fw-main", wantType: "hostname", fieldName: "ci_ref"},
		{name: "ip from ci_hostname", host: "10.53.1.22", wantType: "ip", fieldName: "ci_hostname"},
		{name: "hostname from ci_hostname", host: "fw-main", wantType: "hostname", fieldName: "ci_hostname"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			event := map[string]any{"id": "evt-1", "created_at": "2026-05-30T00:00:00Z"}
			if tt.fieldName == "ci_ref" {
				event["ci_ref"] = map[string]any{"hostname": tt.host}
			} else {
				event["ci_hostname"] = tt.host
			}
			candidate, err := BuildRavenEventCandidate(map[string]any{"event": event}, "")
			if err != nil {
				t.Fatalf("BuildRavenEventCandidate() error = %v, want nil", err)
			}
			ciRef := candidate["ci_ref"].(map[string]any)
			if ciRef["type"] != tt.wantType || ciRef["value"] != tt.host {
				t.Fatalf("ci_ref = %#v, want type %q value %q", ciRef, tt.wantType, tt.host)
			}
		})
	}
}
