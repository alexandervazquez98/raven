package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"raven/internal/domain"
)

func TestSaveAndLoadAliases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	aliases := []domain.Alias{
		{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: domain.AliasTypeCIID, Value: "42"},
		{CIID: "RAVEN-FW-MAIN-001", Source: "manual", Type: domain.AliasTypeHostname, Value: "fw-main"},
	}

	if err := SaveAliases(path, aliases); err != nil {
		t.Fatalf("SaveAliases() error = %v, want nil", err)
	}

	got, err := LoadAliases(path)
	if err != nil {
		t.Fatalf("LoadAliases() error = %v, want nil", err)
	}
	if len(got) != len(aliases) {
		t.Fatalf("LoadAliases() length = %d, want %d", len(got), len(aliases))
	}
	if got[0].Source != "manual" || got[0].Type != domain.AliasTypeHostname || got[1].Source != "next-gen" {
		t.Fatalf("LoadAliases() = %#v, want deterministic order", got)
	}
}

func TestSaveAliasesCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "aliases", "aliases.json")
	aliases := []domain.Alias{{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: domain.AliasTypeCIID, Value: "42"}}

	if err := SaveAliases(path, aliases); err != nil {
		t.Fatalf("SaveAliases() error = %v, want nil", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file stat error = %v, want nil", err)
	}
}

func TestSaveAliasesUsesLowercaseJSONFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	aliases := []domain.Alias{{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: domain.AliasTypeCIID, Value: "42"}}

	if err := SaveAliases(path, aliases); err != nil {
		t.Fatalf("SaveAliases() error = %v, want nil", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}
	content := string(data)
	for _, want := range []string{"\"ci_id\"", "\"source\"", "\"type\"", "\"value\""} {
		if !strings.Contains(content, want) {
			t.Fatalf("saved JSON = %s, want field %s", content, want)
		}
	}
	for _, unwanted := range []string{"\"CIID\"", "\"Source\"", "\"Type\"", "\"Value\""} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("saved JSON = %s, must not contain Go field %s", content, unwanted)
		}
	}
}

func TestLoadAliasesMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	got, err := LoadAliases(path)
	if err != nil {
		t.Fatalf("LoadAliases() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadAliases() length = %d, want 0", len(got))
	}
}

func TestLoadAliasesRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	_, err := LoadAliases(path)
	if err == nil {
		t.Fatal("LoadAliases() error = nil, want error")
	}
}

func TestSaveAliasesValidatesAliases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	aliases := []domain.Alias{{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: domain.AliasTypeCIID}}

	err := SaveAliases(path, aliases)
	if !errors.Is(err, domain.ErrMissingAliasValue) {
		t.Fatalf("SaveAliases() error = %v, want %v", err, domain.ErrMissingAliasValue)
	}
}

func TestSaveAliasesRejectsDuplicateKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	aliases := []domain.Alias{
		{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: domain.AliasTypeCIID, Value: "42"},
		{CIID: "RAVEN-FW-MAIN-001", Source: " next-gen ", Type: domain.AliasType("CI_ID"), Value: " 42 "},
	}

	err := SaveAliases(path, aliases)
	if !errors.Is(err, domain.ErrDuplicateAliasKey) {
		t.Fatalf("SaveAliases() error = %v, want %v", err, domain.ErrDuplicateAliasKey)
	}
}

func TestLoadAliasesRejectsConflictingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	if err := os.WriteFile(path, []byte(`[
		{"ci_id":"RAVEN-FW-MAIN-001","source":"next-gen","type":"ci_id","value":"42"},
		{"ci_id":"RAVEN-FW-BACKUP-001","source":"next-gen","type":"ci_id","value":"42"}
	]`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	_, err := LoadAliases(path)
	if !errors.Is(err, domain.ErrConflictingAliasMapping) {
		t.Fatalf("LoadAliases() error = %v, want %v", err, domain.ErrConflictingAliasMapping)
	}
}
