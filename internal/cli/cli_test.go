package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"raven/internal/app"
	"raven/internal/storage"
)

func TestRunCIAddAndShow(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"ci", "add", "--ci-id", "LAPTOP-ALEX-001", "--category", "other", "--manufacturer", "Lenovo", "--model", "ThinkPad T14"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "added CI LAPTOP-ALEX-001") {
		t.Fatalf("stdout = %q, want add confirmation", stdout.String())
	}

	components, err := storage.LoadComponents(app.ComponentsPath(configDir))
	if err != nil {
		t.Fatalf("LoadComponents() error = %v, want nil", err)
	}
	if len(components) != 1 || components[0].CIID != "LAPTOP-ALEX-001" {
		t.Fatalf("stored components = %#v, want LAPTOP-ALEX-001", components)
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"ci", "show", "LAPTOP-ALEX-001"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(ci show) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"CI ID: LAPTOP-ALEX-001", "Category: other", "Manufacturer: Lenovo", "Model: ThinkPad T14"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunCIList(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	for _, args := range [][]string{
		{"ci", "add", "--ci-id", "CPU-001", "--category", "cpu", "--model", "Ryzen 7 7800X3D"},
		{"ci", "add", "--ci-id", "SSD-001", "--category", "storage", "--model", "990 Pro"},
	} {
		stdout.Reset()
		stderr.Reset()
		if err := Run(args, configDir, &stdout, &stderr); err != nil {
			t.Fatalf("Run(%v) error = %v, want nil; stderr=%q", args, err, stderr.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"ci", "list"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"CI ID", "CPU-001", "Ryzen 7 7800X3D", "SSD-001", "990 Pro"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunCIAddRejectsDuplicate(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	args := []string{"ci", "add", "--ci-id", "CPU-001", "--category", "cpu", "--model", "Ryzen 7 7800X3D"}

	if err := Run(args, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(first add) error = %v, want nil", err)
	}
	stdout.Reset()
	stderr.Reset()

	err := Run(args, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(duplicate add) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "ci id already exists") {
		t.Fatalf("stderr = %q, want duplicate error", stderr.String())
	}
}

func TestRunCIAddAllowsFlexibleCategory(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"ci", "add", "--ci-id", "UPS-001", "--category", "power", "--model", "Smart-UPS 1500"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(ci add flexible category) error = %v, want nil; stderr=%q", err, stderr.String())
	}

	components, err := storage.LoadComponents(app.ComponentsPath(configDir))
	if err != nil {
		t.Fatalf("LoadComponents() error = %v, want nil", err)
	}
	if len(components) != 1 || components[0].Category != "power" {
		t.Fatalf("stored components = %#v, want category power", components)
	}
}

func TestRunCIAddRejectsExtraArgs(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"ci", "add", "--ci-id", "CPU-001", "--category", "cpu", "--model", "Ryzen 7 7800X3D", "extra"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(ci add extra arg) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "does not accept positional arguments") {
		t.Fatalf("stderr = %q, want positional arg error", stderr.String())
	}
}

func TestRunCIListRejectsExtraArgs(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"ci", "list", "extra"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(ci list extra arg) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "ci list does not accept arguments") {
		t.Fatalf("stderr = %q, want list arg error", stderr.String())
	}
}

func TestRunCIShowMissing(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"ci", "show", "MISSING"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(ci show missing) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "component not found") {
		t.Fatalf("stderr = %q, want not found error", stderr.String())
	}
}

func TestRunCIListEmpty(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "list"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No CIs yet.") {
		t.Fatalf("stdout = %q, want empty message", stdout.String())
	}
}

func TestRunEventAddAndTimeline(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "RAVEN-DEV-001", "--category", "logical", "--model", "Raven local CMDB"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	err := Run([]string{"event", "add", "RAVEN-DEV-001", "--type", "observation", "--severity", "info", "--summary", "Initial event recorded", "--source", "human", "--external-id", "manual-001", "--details", "Created from CLI test."}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(event add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "added event") {
		t.Fatalf("stdout = %q, want event add confirmation", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"timeline", "RAVEN-DEV-001"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(timeline) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"Timeline for RAVEN-DEV-001", "observation", "info", "Initial event recorded"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunEventAddRejectsUnknownCI(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"event", "add", "MISSING", "--type", "observation", "--severity", "info", "--summary", "test", "--source", "human"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(event add unknown CI) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "component not found") {
		t.Fatalf("stderr = %q, want not found error", stderr.String())
	}
}

func TestRunEventAddRejectsDuplicateDedupKey(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "RAVEN-DEV-001", "--category", "logical", "--model", "Raven local CMDB"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil", err)
	}
	args := []string{"event", "add", "RAVEN-DEV-001", "--type", "observation", "--severity", "info", "--summary", "test", "--source", "next-gen", "--external-id", "ng-1"}
	stdout.Reset()
	stderr.Reset()
	if err := Run(args, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(first event add) error = %v, want nil", err)
	}
	stdout.Reset()
	stderr.Reset()
	err := Run(args, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(duplicate event add) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "event dedup key already exists") {
		t.Fatalf("stderr = %q, want duplicate dedup error", stderr.String())
	}
}

func TestRunTimelineEmpty(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "RAVEN-DEV-001", "--category", "logical", "--model", "Raven local CMDB"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"timeline", "RAVEN-DEV-001"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(timeline) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No events yet.") {
		t.Fatalf("stdout = %q, want empty timeline", stdout.String())
	}
}

func TestRunEventCapture(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "RAVEN-DEV-001", "--category", "logical", "--model", "Raven local CMDB"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	err := Run([]string{"event", "capture", "RAVEN-DEV-001", "--source", "gemini-cli", "--text", "Gemini diagnosed packet loss symptoms on WAN.", "--type", "diagnosis", "--severity", "info"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(event capture) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "captured event") {
		t.Fatalf("stdout = %q, want capture confirmation", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"timeline", "RAVEN-DEV-001"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(timeline) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"Timeline for RAVEN-DEV-001", "diagnosis", "info", "Gemini diagnosed packet loss symptoms"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunEventCaptureStoresEventFields(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "RAVEN-DEV-001", "--category", "logical", "--model", "Raven local CMDB"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil", err)
	}
	stdout.Reset()
	stderr.Reset()

	err := Run([]string{"event", "capture", "RAVEN-DEV-001", "--source", "gemini-cli", "--text", "Full diagnosis details", "--type", "diagnosis", "--severity", "warning", "--status", "triaged", "--summary", "Custom diagnosis summary"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(event capture) error = %v, want nil; stderr=%q", err, stderr.String())
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(events) != 1 {
		t.Fatalf("events length = %d, want 1", len(events))
	}
	event := events[0]
	if event.Source != "gemini-cli" || event.Details != "Full diagnosis details" || event.Status != "triaged" || event.Summary != "Custom diagnosis summary" {
		t.Fatalf("event fields = %#v, want source/details/status/summary stored", event)
	}
	if !strings.HasPrefix(event.DedupKey, "gemini-cli:evt-") {
		t.Fatalf("event DedupKey = %q, want generated gemini-cli event key", event.DedupKey)
	}
}

func TestRunEventCaptureDefaultsSummaryFromText(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "RAVEN-DEV-001", "--category", "logical", "--model", "Raven local CMDB"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil", err)
	}
	stdout.Reset()
	stderr.Reset()

	err := Run([]string{"event", "capture", "RAVEN-DEV-001", "--source", "ollama", "--text", "Ollama observed repeated DNS timeout symptoms while checking service health."}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(event capture) error = %v, want nil; stderr=%q", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"timeline", "RAVEN-DEV-001"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(timeline) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"observation", "info", "Ollama observed repeated DNS timeout symptoms"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunEventCaptureRejectsUnknownCI(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"event", "capture", "MISSING", "--source", "gemini-cli", "--text", "diagnosis"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(event capture unknown CI) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "component not found") {
		t.Fatalf("stderr = %q, want not found error", stderr.String())
	}
}

func TestRunEventCaptureRequiresText(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "RAVEN-DEV-001", "--category", "logical", "--model", "Raven local CMDB"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil", err)
	}
	stdout.Reset()
	stderr.Reset()

	err := Run([]string{"event", "capture", "RAVEN-DEV-001", "--source", "gemini-cli"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(event capture without text) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "event capture requires --text") {
		t.Fatalf("stderr = %q, want text error", stderr.String())
	}
}

func TestRunEventIngestFromFile(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "FW-MAIN-001", "--category", "network", "--model", "FortiGate"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	file := writeIngestFile(t, `{
		"ci_id":"FW-MAIN-001",
		"type":"network_alert",
		"severity":"warning",
		"status":"open",
		"summary":"High packet loss detected on WAN link",
		"details":"next-gen reported 18% packet loss for 5 minutes.",
		"external_id":"ng-98765",
		"dedup_key":"next-gen:ng-98765",
		"observed_at":"2026-05-28T21:00:00Z",
		"raw":"{\"packet_loss\":18}"
	}`)

	stdout.Reset()
	stderr.Reset()
	err := Run([]string{"event", "ingest", "--source", "next-gen", "--file", file}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(event ingest) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ingested event") {
		t.Fatalf("stdout = %q, want ingest confirmation", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"timeline", "FW-MAIN-001"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(timeline) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"Timeline for FW-MAIN-001", "network_alert", "warning", "High packet loss detected"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunEventIngestFromStdin(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "FW-MAIN-001", "--category", "network", "--model", "FortiGate"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	payload := `{
		"ci_id":"FW-MAIN-001",
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected on WAN link",
		"external_id":"ng-stdin-001",
		"observed_at":"2026-05-28T21:00:00Z"
	}`

	stdout.Reset()
	stderr.Reset()
	err := RunWithInput([]string{"event", "ingest", "--source", "next-gen", "--stdin"}, configDir, strings.NewReader(payload), &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunWithInput(event ingest --stdin) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ingested event") {
		t.Fatalf("stdout = %q, want ingest confirmation", stdout.String())
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(events) != 1 {
		t.Fatalf("events length = %d, want 1", len(events))
	}
	event := events[0]
	if event.Source != "next-gen" || event.DedupKey != "next-gen:ng-stdin-001" || event.Status != "open" {
		t.Fatalf("event fields = %#v, want source override, recomputed dedup key, and default status", event)
	}
}

func TestRunEventIngestRequiresExactlyOneInput(t *testing.T) {
	configDir := t.TempDir()
	file := writeIngestFile(t, `{"ci_id":"FW-MAIN-001"}`)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing input",
			args: []string{"event", "ingest", "--source", "next-gen"},
		},
		{
			name: "ambiguous file and stdin",
			args: []string{"event", "ingest", "--source", "next-gen", "--file", file, "--stdin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := RunWithInput(tt.args, configDir, strings.NewReader(`{}`), &stdout, &stderr)
			if err == nil {
				t.Fatal("RunWithInput(event ingest input validation) error = nil, want error")
			}
			if !strings.Contains(stderr.String(), "event ingest requires exactly one of --file or --stdin") {
				t.Fatalf("stderr = %q, want exactly-one input error", stderr.String())
			}
		})
	}
}

func TestRunEventIngestRejectsUnknownCI(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	file := writeIngestFile(t, `{
		"ci_id":"MISSING",
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected",
		"external_id":"ng-98765",
		"dedup_key":"next-gen:ng-98765",
		"observed_at":"2026-05-28T21:00:00Z"
	}`)

	err := Run([]string{"event", "ingest", "--source", "next-gen", "--file", file}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(event ingest unknown CI) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "component not found") {
		t.Fatalf("stderr = %q, want not found error", stderr.String())
	}
}

func TestRunEventIngestRejectsDuplicateDedupKey(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "FW-MAIN-001", "--category", "network", "--model", "FortiGate"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil", err)
	}
	file := writeIngestFile(t, `{
		"ci_id":"FW-MAIN-001",
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected",
		"external_id":"ng-98765",
		"dedup_key":"next-gen:ng-98765",
		"observed_at":"2026-05-28T21:00:00Z"
	}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"event", "ingest", "--source", "next-gen", "--file", file}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(first event ingest) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	err := Run([]string{"event", "ingest", "--source", "next-gen", "--file", file}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(duplicate event ingest) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "event dedup key already exists") {
		t.Fatalf("stderr = %q, want duplicate dedup error", stderr.String())
	}
}

func TestRunEventIngestSourceOverrideRecomputesDedupKey(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "FW-MAIN-001", "--category", "network", "--model", "FortiGate"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil", err)
	}
	file := writeIngestFile(t, `{
		"ci_id":"FW-MAIN-001",
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected",
		"source":"payload-source",
		"external_id":"ng-same",
		"dedup_key":"payload-source:ng-same",
		"observed_at":"2026-05-28T21:00:00Z"
	}`)

	if err := Run([]string{"event", "ingest", "--source", "next-gen", "--file", file}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(first event ingest) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	err := Run([]string{"event", "ingest", "--source", "next-gen", "--file", file}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(duplicate source override ingest) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "event dedup key already exists") {
		t.Fatalf("stderr = %q, want duplicate dedup error", stderr.String())
	}
}

func TestRunEventIngestRequiresStableDedupIdentity(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "FW-MAIN-001", "--category", "network", "--model", "FortiGate"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil", err)
	}
	file := writeIngestFile(t, `{
		"ci_id":"FW-MAIN-001",
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected",
		"observed_at":"2026-05-28T21:00:00Z"
	}`)

	err := Run([]string{"event", "ingest", "--source", "next-gen", "--file", file}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(event ingest without external_id/dedup_key) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "event ingest requires external_id or dedup_key") {
		t.Fatalf("stderr = %q, want stable dedup identity error", stderr.String())
	}
}

func writeIngestFile(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/alert.json"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}
	return path
}
