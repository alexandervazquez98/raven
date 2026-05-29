package domain

import (
	"errors"
	"testing"
)

func TestAliasValidate(t *testing.T) {
	tests := []struct {
		name string
		in   Alias
		want error
	}{
		{
			name: "valid next-gen ci id alias",
			in:   Alias{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: AliasTypeCIID, Value: "42"},
		},
		{
			name: "missing ci id",
			in:   Alias{Source: "next-gen", Type: AliasTypeCIID, Value: "42"},
			want: ErrMissingCIID,
		},
		{
			name: "missing source",
			in:   Alias{CIID: "RAVEN-FW-MAIN-001", Type: AliasTypeCIID, Value: "42"},
			want: ErrMissingAliasSource,
		},
		{
			name: "missing type",
			in:   Alias{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Value: "42"},
			want: ErrMissingAliasType,
		},
		{
			name: "unsupported type",
			in:   Alias{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: AliasType("asset_tag"), Value: "42"},
			want: ErrUnsupportedAliasType,
		},
		{
			name: "missing value",
			in:   Alias{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: AliasTypeCIID},
			want: ErrMissingAliasValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestIsSupportedAliasTypeIncludesIssueTypes(t *testing.T) {
	for _, aliasType := range []AliasType{AliasTypeCIID, AliasTypeIP, AliasTypeHostname, AliasTypeSerial, AliasTypeMAC} {
		t.Run(string(aliasType), func(t *testing.T) {
			if !IsSupportedAliasType(aliasType) {
				t.Fatalf("IsSupportedAliasType(%q) = false, want true", aliasType)
			}
		})
	}
}

func TestAliasRegistryAddResolveAndList(t *testing.T) {
	registry := NewAliasRegistry()
	aliases := []Alias{
		{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: AliasTypeCIID, Value: "42"},
		{CIID: "RAVEN-FW-MAIN-001", Source: "manual", Type: AliasTypeHostname, Value: "fw-main"},
	}
	for _, alias := range aliases {
		if err := registry.Add(alias); err != nil {
			t.Fatalf("Add(%#v) error = %v, want nil", alias, err)
		}
	}

	got, err := registry.Resolve(AliasKey{Source: " next-gen ", Type: AliasType("CI_ID"), Value: " 42 "})
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}
	if got != "RAVEN-FW-MAIN-001" {
		t.Fatalf("Resolve() = %q, want RAVEN-FW-MAIN-001", got)
	}

	listed := registry.List()
	if len(listed) != 2 {
		t.Fatalf("List() length = %d, want 2", len(listed))
	}
	if listed[0].Source != "manual" || listed[0].Type != AliasTypeHostname || listed[1].Source != "next-gen" {
		t.Fatalf("List() = %#v, want deterministic source/type/value order", listed)
	}
}

func TestAliasRegistryRejectsDuplicateKey(t *testing.T) {
	registry := NewAliasRegistry()
	alias := Alias{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: AliasTypeCIID, Value: "42"}
	if err := registry.Add(alias); err != nil {
		t.Fatalf("Add(first) error = %v, want nil", err)
	}

	err := registry.Add(alias)
	if !errors.Is(err, ErrDuplicateAliasKey) {
		t.Fatalf("Add(duplicate) error = %v, want %v", err, ErrDuplicateAliasKey)
	}
}

func TestAliasRegistryRejectsConflictingMapping(t *testing.T) {
	registry := NewAliasRegistry()
	if err := registry.Add(Alias{CIID: "RAVEN-FW-MAIN-001", Source: "next-gen", Type: AliasTypeCIID, Value: "42"}); err != nil {
		t.Fatalf("Add(first) error = %v, want nil", err)
	}

	err := registry.Add(Alias{CIID: "RAVEN-FW-BACKUP-001", Source: "next-gen", Type: AliasTypeCIID, Value: "42"})
	if !errors.Is(err, ErrConflictingAliasMapping) {
		t.Fatalf("Add(conflict) error = %v, want %v", err, ErrConflictingAliasMapping)
	}
}

func TestAliasRegistryResolveMissing(t *testing.T) {
	registry := NewAliasRegistry()

	_, err := registry.Resolve(AliasKey{Source: "next-gen", Type: AliasTypeCIID, Value: "42"})
	if !errors.Is(err, ErrAliasNotFound) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrAliasNotFound)
	}
}

func TestAliasRegistryResolveValidatesKey(t *testing.T) {
	registry := NewAliasRegistry()

	_, err := registry.Resolve(AliasKey{Source: "next-gen", Type: AliasType("asset_tag"), Value: "42"})
	if !errors.Is(err, ErrUnsupportedAliasType) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrUnsupportedAliasType)
	}
}
