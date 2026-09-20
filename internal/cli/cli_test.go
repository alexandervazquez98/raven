package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"raven/internal/app"
	"raven/internal/domain"
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
	if !strings.Contains(stderr.String(), "ci list does not accept positional arguments") {
		t.Fatalf("stderr = %q, want list arg error", stderr.String())
	}
}

func seedInventoryForList(t *testing.T, configDir string) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	add := func(args ...string) {
		t.Helper()
		stdout.Reset()
		stderr.Reset()
		if err := Run(args, configDir, &stdout, &stderr); err != nil {
			t.Fatalf("Run(%v) error = %v, want nil; stderr=%q", args, err, stderr.String())
		}
	}
	add([]string{"ci", "add", "--ci-id", "FW-MAIN-001", "--category", "network", "--manufacturer", "Fortinet", "--model", "FortiGate", "--notes", "Primary edge firewall"}...)
	add([]string{"ci", "add", "--ci-id", "TWR-CORE-001", "--category", "network", "--manufacturer", "Cisco", "--model", "Catalyst 9300", "--notes", "Core switch"}...)
	add([]string{"ci", "add", "--ci-id", "SRV-DB-001", "--category", "server", "--manufacturer", "Dell", "--model", "PowerEdge R750", "--notes", "Database host"}...)
}

func TestRunCIListWithCategory(t *testing.T) {
	configDir := t.TempDir()
	seedInventoryForList(t, configDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"ci", "list", "--category", "network"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list --category) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"FW-MAIN-001", "TWR-CORE-001"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	if strings.Contains(stdout.String(), "SRV-DB-001") {
		t.Fatalf("stdout = %q, did not want SRV-DB-001", stdout.String())
	}
}

func TestRunCIListWithPrefix(t *testing.T) {
	configDir := t.TempDir()
	seedInventoryForList(t, configDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"ci", "list", "--prefix", "TWR-"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list --prefix) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "TWR-CORE-001") {
		t.Fatalf("stdout = %q, want TWR-CORE-001", stdout.String())
	}
	for _, missing := range []string{"FW-MAIN-001", "SRV-DB-001"} {
		if strings.Contains(stdout.String(), missing) {
			t.Fatalf("stdout = %q, did not want %q", stdout.String(), missing)
		}
	}
}

func TestRunCIListWithQuery(t *testing.T) {
	configDir := t.TempDir()
	seedInventoryForList(t, configDir)

	tests := []struct {
		name    string
		query   string
		wantRow string
	}{
		{name: "case-insensitive ci_id", query: "fw-main", wantRow: "FW-MAIN-001"},
		{name: "case-insensitive model", query: "forti", wantRow: "FW-MAIN-001"},
		{name: "case-insensitive notes", query: "core switch", wantRow: "TWR-CORE-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if err := Run([]string{"ci", "list", "--query", tt.query}, configDir, &stdout, &stderr); err != nil {
				t.Fatalf("Run(ci list --query %q) error = %v, want nil; stderr=%q", tt.query, err, stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.wantRow) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.wantRow)
			}
		})
	}
}

func TestRunCIListWithLimit(t *testing.T) {
	configDir := t.TempDir()
	seedInventoryForList(t, configDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"ci", "list", "--limit", "2"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list --limit) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	rows := 0
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if strings.HasPrefix(line, "FW-MAIN-001") || strings.HasPrefix(line, "TWR-CORE-001") || strings.HasPrefix(line, "SRV-DB-001") {
			rows++
		}
	}
	if rows != 2 {
		t.Fatalf("stdout rows under limit = %d, want 2; stdout=%q", rows, stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"ci", "list", "--limit", "0"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list --limit 0) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	rows = 0
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if strings.HasPrefix(line, "FW-MAIN-001") || strings.HasPrefix(line, "TWR-CORE-001") || strings.HasPrefix(line, "SRV-DB-001") {
			rows++
		}
	}
	if rows != 3 {
		t.Fatalf("stdout rows with limit 0 = %d, want 3 (no cap); stdout=%q", rows, stdout.String())
	}
}

func TestRunCIListFiltersCombined(t *testing.T) {
	configDir := t.TempDir()
	seedInventoryForList(t, configDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"ci", "list", "--category", "network", "--prefix", "FW-"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list combined) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "FW-MAIN-001") {
		t.Fatalf("stdout = %q, want FW-MAIN-001", stdout.String())
	}
	if strings.Contains(stdout.String(), "TWR-CORE-001") {
		t.Fatalf("stdout = %q, did not want TWR-CORE-001", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"ci", "list", "--query", "forti", "--limit", "1"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list query+limit) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	rows := 0
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if strings.HasPrefix(line, "FW-MAIN-001") || strings.HasPrefix(line, "TWR-CORE-001") || strings.HasPrefix(line, "SRV-DB-001") {
			rows++
		}
	}
	if rows != 1 {
		t.Fatalf("stdout rows under query+limit = %d, want 1; stdout=%q", rows, stdout.String())
	}
	if !strings.Contains(stdout.String(), "FW-MAIN-001") {
		t.Fatalf("stdout = %q, want FW-MAIN-001", stdout.String())
	}
}

func TestRunCIListNoMatchMessage(t *testing.T) {
	configDir := t.TempDir()
	seedInventoryForList(t, configDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"ci", "list", "--category", "power"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci list no match) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No CIs matched the provided filters.") {
		t.Fatalf("stdout = %q, want filtered-empty message", stdout.String())
	}
	if strings.Contains(stdout.String(), "FW-MAIN-001") {
		t.Fatalf("stdout = %q, did not want any CI rows", stdout.String())
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

func TestRunAliasAddListAndResolve(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "RAVEN-FW-MAIN-001", "--category", "network", "--model", "FortiGate"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	err := Run([]string{"alias", "add", "--ci-id", "RAVEN-FW-MAIN-001", "--source", "next-gen", "--type", "ci_id", "--value", "42"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(alias add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "added alias next-gen ci_id 42 -> RAVEN-FW-MAIN-001") {
		t.Fatalf("stdout = %q, want alias add confirmation", stdout.String())
	}

	aliases, err := storage.LoadAliases(app.AliasesPath(configDir))
	if err != nil {
		t.Fatalf("LoadAliases() error = %v, want nil", err)
	}
	if len(aliases) != 1 || aliases[0].CIID != "RAVEN-FW-MAIN-001" || aliases[0].Source != "next-gen" || aliases[0].Type != "ci_id" || aliases[0].Value != "42" {
		t.Fatalf("stored aliases = %#v, want next-gen ci_id 42 mapping", aliases)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"alias", "resolve", "--source", "next-gen", "--type", "ci_id", "--value", "42"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(alias resolve) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "RAVEN-FW-MAIN-001" {
		t.Fatalf("stdout = %q, want canonical ci id", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"alias", "list"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(alias list) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"Source\tType\tValue\tCI ID", "next-gen\tci_id\t42\tRAVEN-FW-MAIN-001"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunAliasAddRejectsUnknownCI(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"alias", "add", "--ci-id", "MISSING", "--source", "next-gen", "--type", "ci_id", "--value", "42"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(alias add unknown CI) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "component not found") {
		t.Fatalf("stderr = %q, want not found error", stderr.String())
	}
}

func TestRunAliasAddRejectsDuplicateAndConflict(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	for _, args := range [][]string{
		{"ci", "add", "--ci-id", "RAVEN-FW-MAIN-001", "--category", "network", "--model", "FortiGate"},
		{"ci", "add", "--ci-id", "RAVEN-FW-BACKUP-001", "--category", "network", "--model", "FortiGate Backup"},
	} {
		stdout.Reset()
		stderr.Reset()
		if err := Run(args, configDir, &stdout, &stderr); err != nil {
			t.Fatalf("Run(%v) error = %v, want nil; stderr=%q", args, err, stderr.String())
		}
	}

	args := []string{"alias", "add", "--ci-id", "RAVEN-FW-MAIN-001", "--source", "next-gen", "--type", "ci_id", "--value", "42"}
	stdout.Reset()
	stderr.Reset()
	if err := Run(args, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(first alias add) error = %v, want nil; stderr=%q", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err := Run(args, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(duplicate alias add) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "alias already exists") {
		t.Fatalf("stderr = %q, want duplicate alias error", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"alias", "add", "--ci-id", "RAVEN-FW-BACKUP-001", "--source", "next-gen", "--type", "ci_id", "--value", "42"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(conflicting alias add) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "alias conflicts with existing ci id") {
		t.Fatalf("stderr = %q, want conflict alias error", stderr.String())
	}
}

func TestRunAliasListEmpty(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"alias", "list"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(alias list) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No aliases yet.") {
		t.Fatalf("stdout = %q, want empty alias message", stdout.String())
	}
}

func TestRunAliasResolveMissing(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"alias", "resolve", "--source", "next-gen", "--type", "ci_id", "--value", "42"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(alias resolve missing) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "alias not found") {
		t.Fatalf("stderr = %q, want not found error", stderr.String())
	}
}

func TestRunAliasCommandsRejectExtraArgs(t *testing.T) {
	configDir := t.TempDir()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "add", args: []string{"alias", "add", "--ci-id", "RAVEN-FW-MAIN-001", "--source", "next-gen", "--type", "ci_id", "--value", "42", "extra"}, want: "alias add does not accept positional arguments"},
		{name: "list", args: []string{"alias", "list", "extra"}, want: "alias list does not accept arguments"},
		{name: "resolve", args: []string{"alias", "resolve", "--source", "next-gen", "--type", "ci_id", "--value", "42", "extra"}, want: "alias resolve does not accept positional arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := Run(tt.args, configDir, &stdout, &stderr)
			if err == nil {
				t.Fatal("Run(alias extra args) error = nil, want error")
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
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

func TestRunEventIngestResolvesCIRefAlias(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "FW-MAIN-001", "--category", "network", "--model", "FortiGate"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"alias", "add", "--ci-id", "FW-MAIN-001", "--source", "next-gen", "--type", "ci_id", "--value", "42"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(alias add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	payload := `{
		"ci_ref":{"source":"next-gen","type":"ci_id","value":"42"},
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected on WAN link",
		"external_id":"ng-ref-001",
		"observed_at":"2026-05-28T21:00:00Z"
	}`

	stdout.Reset()
	stderr.Reset()
	err := RunWithInput([]string{"event", "ingest", "--source", "next-gen", "--stdin"}, configDir, strings.NewReader(payload), &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunWithInput(event ingest ci_ref) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ingested event") || !strings.Contains(stdout.String(), "FW-MAIN-001") {
		t.Fatalf("stdout = %q, want ingest confirmation with canonical CI", stdout.String())
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(events) != 1 {
		t.Fatalf("events length = %d, want 1", len(events))
	}
	if events[0].CIID != "FW-MAIN-001" || events[0].DedupKey != "next-gen:ng-ref-001" || events[0].ExternalID != "ng-ref-001" {
		t.Fatalf("event = %#v, want canonical CI with unchanged external_id/dedup behavior", events[0])
	}
}

func TestRunEventIngestRejectsUnknownCIRef(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	payload := `{
		"ci_ref":{"source":"next-gen","type":"ci_id","value":"missing"},
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected",
		"external_id":"ng-missing-ref",
		"observed_at":"2026-05-28T21:00:00Z"
	}`

	err := RunWithInput([]string{"event", "ingest", "--source", "next-gen", "--stdin"}, configDir, strings.NewReader(payload), &stdout, &stderr)
	if err == nil {
		t.Fatal("RunWithInput(event ingest unknown ci_ref) error = nil, want error")
	}
	for _, want := range []string{"resolve ci_ref next-gen ci_id missing", "alias not found"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, want %q", stderr.String(), want)
		}
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(events) != 0 {
		t.Fatalf("events length = %d, want 0 after failed ci_ref resolution", len(events))
	}
}

func TestRunEventIngestRejectsInvalidCIRef(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	payload := `{
		"ci_ref":{"source":"next-gen","type":"asset_tag","value":"42"},
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected",
		"external_id":"ng-invalid-ref",
		"observed_at":"2026-05-28T21:00:00Z"
	}`

	err := RunWithInput([]string{"event", "ingest", "--source", "next-gen", "--stdin"}, configDir, strings.NewReader(payload), &stdout, &stderr)
	if err == nil {
		t.Fatal("RunWithInput(event ingest invalid ci_ref) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "unsupported alias type") {
		t.Fatalf("stderr = %q, want unsupported alias type error", stderr.String())
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(events) != 0 {
		t.Fatalf("events length = %d, want 0 after invalid ci_ref", len(events))
	}
}

func TestRunEventIngestRequiresCIIDOrCIRef(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	payload := `{
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected",
		"external_id":"ng-no-ci",
		"observed_at":"2026-05-28T21:00:00Z"
	}`

	err := RunWithInput([]string{"event", "ingest", "--source", "next-gen", "--stdin"}, configDir, strings.NewReader(payload), &stdout, &stderr)
	if err == nil {
		t.Fatal("RunWithInput(event ingest without ci_id or ci_ref) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "event ingest requires ci_id or ci_ref") {
		t.Fatalf("stderr = %q, want ci identity error", stderr.String())
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(events) != 0 {
		t.Fatalf("events length = %d, want 0 without ci identity", len(events))
	}
}

func TestRunEventIngestCIIDWinsOverInvalidCIRef(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"ci", "add", "--ci-id", "FW-MAIN-001", "--category", "network", "--model", "FortiGate"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(ci add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	payload := `{
		"ci_id":"FW-MAIN-001",
		"ci_ref":{"source":"next-gen","type":"asset_tag","value":"42"},
		"type":"network_alert",
		"severity":"warning",
		"summary":"High packet loss detected",
		"external_id":"ng-ciid-wins",
		"observed_at":"2026-05-28T21:00:00Z"
	}`

	stdout.Reset()
	stderr.Reset()
	err := RunWithInput([]string{"event", "ingest", "--source", "next-gen", "--stdin"}, configDir, strings.NewReader(payload), &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunWithInput(event ingest ci_id precedence) error = %v, want nil; stderr=%q", err, stderr.String())
	}

	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(events) != 1 || events[0].CIID != "FW-MAIN-001" {
		t.Fatalf("events = %#v, want event stored under ci_id while ignoring ci_ref", events)
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

func TestRunNextGenMCPRejectsUnexpectedArguments(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"nextgen-mcp", "extra"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(nextgen-mcp extra) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "nextgen-mcp does not accept arguments: extra") {
		t.Fatalf("stderr = %q, want nextgen-mcp argument error", stderr.String())
	}
}

func seedMetadata(t *testing.T, configDir string, sidecar domain.MetadataSidecar) {
	t.Helper()
	if sidecar.Version == 0 {
		sidecar.Version = domain.MetadataSidecarVersion
	}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), sidecar); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}
}

func TestRunMetadataAddCreatesEntry(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"metadata", "add", "--ci-id", "FW-MAIN-001", "--attribute", "azimuth_deg=12.5,monitored=true,label=primary", "--relationship", "RAVEN-CORE-RTR-01=runs-on,RAVEN-RACK-A01=located-at"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(metadata add) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "updated metadata for FW-MAIN-001") {
		t.Fatalf("stdout = %q, want update confirmation", stdout.String())
	}

	sidecar, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(sidecar.Entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(sidecar.Entries))
	}
	entry := sidecar.Entries[0]
	if entry.CIID != "FW-MAIN-001" {
		t.Fatalf("entry CIID = %q, want FW-MAIN-001", entry.CIID)
	}
	if len(entry.Attributes) != 3 {
		t.Fatalf("attributes length = %d, want 3", len(entry.Attributes))
	}
	if v, ok := entry.Attributes["azimuth_deg"]; !ok || v.Raw() != 12.5 {
		t.Fatalf("azimuth_deg = %#v, want NumberValue(12.5)", v)
	}
	if v, ok := entry.Attributes["monitored"]; !ok || v.Raw() != true {
		t.Fatalf("monitored = %#v, want BoolValue(true)", v)
	}
	if v, ok := entry.Attributes["label"]; !ok || v.Raw() != "primary" {
		t.Fatalf("label = %#v, want StringValue(primary)", v)
	}
	if len(entry.Relationships) != 2 {
		t.Fatalf("relationships length = %d, want 2", len(entry.Relationships))
	}
	wantRels := map[string]string{"RAVEN-CORE-RTR-01": "runs-on", "RAVEN-RACK-A01": "located-at"}
	for _, rel := range entry.Relationships {
		want, ok := wantRels[rel.TargetCIID]
		if !ok {
			t.Fatalf("unexpected relationship target %q", rel.TargetCIID)
		}
		if rel.Kind != want {
			t.Fatalf("relationship %q kind = %q, want %q", rel.TargetCIID, rel.Kind, want)
		}
	}
}

func TestRunMetadataAddMergesIntoExistingEntry(t *testing.T) {
	configDir := t.TempDir()
	seedMetadata(t, configDir, domain.MetadataSidecar{
		Entries: []domain.CIMetadataEntry{
			{
				CIID: "FW-MAIN-001",
				Attributes: map[string]domain.TypedValue{
					"azimuth_deg": domain.NumberValue(12.5),
					"label":       domain.StringValue("primary"),
				},
				Relationships: []domain.CIRelationship{{TargetCIID: "RAVEN-RACK-A01", Kind: "located-at"}},
			},
		},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"metadata", "add", "--ci-id", "FW-MAIN-001", "--attribute", "label=edge,monitored=true", "--relationship", "RAVEN-CORE-RTR-01=runs-on"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(metadata add merge) error = %v, want nil; stderr=%q", err, stderr.String())
	}

	sidecar, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(sidecar.Entries) != 1 {
		t.Fatalf("entries length = %d, want 1 after merge", len(sidecar.Entries))
	}
	entry := sidecar.Entries[0]
	if v, ok := entry.Attributes["azimuth_deg"]; !ok || v.Raw() != 12.5 {
		t.Fatalf("azimuth_deg = %#v, want preserved NumberValue(12.5)", v)
	}
	if v, ok := entry.Attributes["label"]; !ok || v.Raw() != "edge" {
		t.Fatalf("label = %#v, want overwritten StringValue(edge)", v)
	}
	if v, ok := entry.Attributes["monitored"]; !ok || v.Raw() != true {
		t.Fatalf("monitored = %#v, want new BoolValue(true)", v)
	}
	if len(entry.Relationships) != 2 {
		t.Fatalf("relationships length = %d, want 2 after merge", len(entry.Relationships))
	}
}

func TestRunMetadataAddRejectsMissingCIID(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"metadata", "add", "--attribute", "label=primary"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(metadata add no ci-id) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "metadata add requires --ci-id") {
		t.Fatalf("stderr = %q, want missing ci-id error", stderr.String())
	}

	sidecar, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(sidecar.Entries) != 0 {
		t.Fatalf("entries length = %d, want 0 after rejected add", len(sidecar.Entries))
	}
}

func TestRunMetadataAddRejectsMalformedAttribute(t *testing.T) {
	configDir := t.TempDir()
	tests := []struct {
		name      string
		attribute string
	}{
		{name: "no value", attribute: "novalue"},
		{name: "empty key", attribute: "=value"},
		{name: "trailing equals", attribute: "label="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := Run([]string{"metadata", "add", "--ci-id", "FW-MAIN-001", "--attribute", tt.attribute}, configDir, &stdout, &stderr)
			if err == nil {
				t.Fatal("Run(metadata add malformed attr) error = nil, want error")
			}
			if !strings.Contains(stderr.String(), "invalid --attribute entry") && !strings.Contains(stderr.String(), "--attribute has empty key") {
				t.Fatalf("stderr = %q, want malformed attribute error", stderr.String())
			}
		})
	}
}

func TestRunMetadataAddRejectsMalformedRelationship(t *testing.T) {
	configDir := t.TempDir()
	tests := []struct {
		name         string
		relationship string
	}{
		{name: "no kind", relationship: "RAVEN-CORE-RTR-01"},
		{name: "empty target", relationship: "=runs-on"},
		{name: "trailing equals", relationship: "RAVEN-CORE-RTR-01="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := Run([]string{"metadata", "add", "--ci-id", "FW-MAIN-001", "--relationship", tt.relationship}, configDir, &stdout, &stderr)
			if err == nil {
				t.Fatal("Run(metadata add malformed rel) error = nil, want error")
			}
			if !strings.Contains(stderr.String(), "invalid --relationship entry") && !strings.Contains(stderr.String(), "--relationship pair") {
				t.Fatalf("stderr = %q, want malformed relationship error", stderr.String())
			}
		})
	}
}

func TestRunMetadataAddAutoTypesValues(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"metadata", "add", "--ci-id", "FW-MAIN-001", "--attribute", "enabled=true,disabled=false,count=45,greeting=hello"}, configDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run(metadata add auto-type) error = %v, want nil; stderr=%q", err, stderr.String())
	}

	sidecar, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(sidecar.Entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(sidecar.Entries))
	}
	entry := sidecar.Entries[0]
	want := map[string]any{
		"enabled":  true,
		"disabled": false,
		"count":    float64(45),
		"greeting": "hello",
	}
	if len(entry.Attributes) != len(want) {
		t.Fatalf("attributes length = %d, want %d", len(entry.Attributes), len(want))
	}
	for key, expected := range want {
		got, ok := entry.Attributes[key]
		if !ok {
			t.Fatalf("missing attribute %q", key)
		}
		if got.Raw() != expected {
			t.Fatalf("attribute %q raw = %#v, want %#v", key, got.Raw(), expected)
		}
	}
}

func TestRunMetadataListEmpty(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Run([]string{"metadata", "list"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(metadata list empty) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No metadata yet.") {
		t.Fatalf("stdout = %q, want empty metadata message", stdout.String())
	}
}

func TestRunMetadataListPrintsTabularSummary(t *testing.T) {
	configDir := t.TempDir()
	seedMetadata(t, configDir, domain.MetadataSidecar{
		Entries: []domain.CIMetadataEntry{
			{
				CIID: "FW-MAIN-001",
				Attributes: map[string]domain.TypedValue{
					"azimuth_deg": domain.NumberValue(12.5),
					"label":       domain.StringValue("primary"),
				},
				Relationships: []domain.CIRelationship{{TargetCIID: "RAVEN-RACK-A01", Kind: "located-at"}},
			},
			{
				CIID:          "SRV-DB-001",
				Attributes:    map[string]domain.TypedValue{"rack": domain.StringValue("A01")},
				Relationships: []domain.CIRelationship{{TargetCIID: "FW-MAIN-001", Kind: "behind"}, {TargetCIID: "RAVEN-RACK-A01", Kind: "located-at"}},
			},
		},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"metadata", "list"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(metadata list) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{"CI ID\tAttributes\tRelationships", "FW-MAIN-001\t2\t1", "SRV-DB-001\t1\t2"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunMetadataListAppliesLimit(t *testing.T) {
	configDir := t.TempDir()
	seedMetadata(t, configDir, domain.MetadataSidecar{
		Entries: []domain.CIMetadataEntry{
			{CIID: "FW-MAIN-001", Attributes: map[string]domain.TypedValue{"label": domain.StringValue("a")}},
			{CIID: "FW-BACKUP-001", Attributes: map[string]domain.TypedValue{"label": domain.StringValue("b")}},
			{CIID: "FW-DR-001", Attributes: map[string]domain.TypedValue{"label": domain.StringValue("c")}},
		},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"metadata", "list", "--limit", "2"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(metadata list --limit) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	rows := 0
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if strings.HasPrefix(line, "FW-MAIN-001") || strings.HasPrefix(line, "FW-BACKUP-001") || strings.HasPrefix(line, "FW-DR-001") {
			rows++
		}
	}
	if rows != 2 {
		t.Fatalf("stdout rows under limit = %d, want 2; stdout=%q", rows, stdout.String())
	}
	if strings.Contains(stdout.String(), "FW-DR-001") {
		t.Fatalf("stdout = %q, did not want FW-DR-001 beyond limit", stdout.String())
	}
}

func TestRunMetadataListRejectsPositionalArgs(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"metadata", "list", "extra"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(metadata list extra) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "metadata list does not accept positional arguments") {
		t.Fatalf("stderr = %q, want positional arg error", stderr.String())
	}
}

func TestRunMetadataShowPrintsEntry(t *testing.T) {
	configDir := t.TempDir()
	seedMetadata(t, configDir, domain.MetadataSidecar{
		Entries: []domain.CIMetadataEntry{
			{
				CIID: "FW-MAIN-001",
				Attributes: map[string]domain.TypedValue{
					"azimuth_deg": domain.NumberValue(12.5),
					"label":       domain.StringValue("primary"),
					"monitored":   domain.BoolValue(true),
				},
				Relationships: []domain.CIRelationship{
					{TargetCIID: "RAVEN-RACK-A01", Kind: "located-at"},
					{TargetCIID: "RAVEN-CORE-RTR-01", Kind: "runs-on"},
				},
			},
		},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"metadata", "show", "FW-MAIN-001"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(metadata show) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	for _, want := range []string{
		"CI ID: FW-MAIN-001",
		"Attributes:",
		"azimuth_deg = 12.5",
		"label = primary",
		"monitored = true",
		"Relationships:",
		"RAVEN-RACK-A01 -> located-at",
		"RAVEN-CORE-RTR-01 -> runs-on",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunMetadataShowRequiresCIID(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"metadata", "show"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(metadata show without ci-id) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "metadata show requires exactly one positional argument") {
		t.Fatalf("stderr = %q, want ci-id requirement error", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"metadata", "show", "FW-MAIN-001", "extra"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(metadata show two positionals) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), "metadata show requires exactly one positional argument") {
		t.Fatalf("stderr = %q, want ci-id requirement error", stderr.String())
	}
}

func TestRunMetadataShowUnknownCIID(t *testing.T) {
	configDir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"metadata", "show", "MISSING"}, configDir, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run(metadata show unknown) error = nil, want error")
	}
	if !strings.Contains(stderr.String(), `no metadata found for CI "MISSING"`) {
		t.Fatalf("stderr = %q, want unknown ci-id error", stderr.String())
	}
}

func TestRunMetadataShowEmptyEntry(t *testing.T) {
	configDir := t.TempDir()
	seedMetadata(t, configDir, domain.MetadataSidecar{
		Entries: []domain.CIMetadataEntry{
			{CIID: "FW-MAIN-001"},
		},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"metadata", "show", "FW-MAIN-001"}, configDir, &stdout, &stderr); err != nil {
		t.Fatalf("Run(metadata show empty) error = %v, want nil; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CI ID: FW-MAIN-001") {
		t.Fatalf("stdout = %q, want ci-id line", stdout.String())
	}
	if !strings.Contains(stdout.String(), "(no attributes or relationships)") {
		t.Fatalf("stdout = %q, want empty placeholder", stdout.String())
	}
	if strings.Contains(stdout.String(), "Attributes:") {
		t.Fatalf("stdout = %q, did not want Attributes section", stdout.String())
	}
	if strings.Contains(stdout.String(), "Relationships:") {
		t.Fatalf("stdout = %q, did not want Relationships section", stdout.String())
	}
}
