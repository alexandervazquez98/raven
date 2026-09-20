package service

import (
	"errors"
	"testing"
	"time"

	"raven/internal/app"
	"raven/internal/domain"
	"raven/internal/storage"
)

func TestRecordEventResolvesCIRefAndStoresEvent(t *testing.T) {
	configDir := t.TempDir()
	seedCIAndAlias(t, configDir)
	now := time.Date(2026, 5, 29, 20, 0, 0, 0, time.UTC)
	svc := Service{ConfigDir: configDir, Now: func() time.Time { return now }}

	event, err := svc.RecordEvent(RecordEventInput{
		Event: domain.Event{
			Type:       "network_alert",
			Severity:   "warning",
			Summary:    "High packet loss detected",
			Source:     "next-gen",
			ExternalID: "ng-98765",
			ObservedAt: now.Add(-time.Minute),
		},
		CIRef: &CIRef{Source: "next-gen", Type: domain.AliasTypeCIID, Value: "42"},
	})
	if err != nil {
		t.Fatalf("RecordEvent() error = %v, want nil", err)
	}
	if event.CIID != "FW-MAIN-001" {
		t.Fatalf("RecordEvent() CIID = %q, want canonical CI", event.CIID)
	}
	if event.ID == "" || event.Status != "open" || event.DedupKey != "next-gen:ng-98765" || !event.IngestedAt.Equal(now) {
		t.Fatalf("RecordEvent() event = %#v, want generated id/default status/dedup/ingested_at", event)
	}

	stored, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(stored) != 1 || stored[0].CIID != "FW-MAIN-001" {
		t.Fatalf("stored events = %#v, want one event for canonical CI", stored)
	}
}

func TestRecordEventRejectsMissingIdentity(t *testing.T) {
	configDir := t.TempDir()
	if err := storage.SaveComponents(app.ComponentsPath(configDir), []domain.Component{{CIID: "FW-MAIN-001", Category: "network", Model: "FortiGate"}}); err != nil {
		t.Fatalf("SaveComponents() error = %v, want nil", err)
	}
	svc := Service{ConfigDir: configDir, Now: fixedNow}

	_, err := svc.RecordEvent(RecordEventInput{Event: validEventWithoutIdentity()})
	if !errors.Is(err, ErrMissingEventIdentity) {
		t.Fatalf("RecordEvent() error = %v, want %v", err, ErrMissingEventIdentity)
	}
}

func TestRecordEventRejectsUnknownAliasWithReadableContext(t *testing.T) {
	configDir := t.TempDir()
	if err := storage.SaveComponents(app.ComponentsPath(configDir), []domain.Component{{CIID: "FW-MAIN-001", Category: "network", Model: "FortiGate"}}); err != nil {
		t.Fatalf("SaveComponents() error = %v, want nil", err)
	}
	svc := Service{ConfigDir: configDir, Now: fixedNow}

	_, err := svc.RecordEvent(RecordEventInput{
		Event: validEventWithoutIdentity(),
		CIRef: &CIRef{Source: "next-gen", Type: domain.AliasTypeCIID, Value: "missing"},
	})
	if !errors.Is(err, domain.ErrAliasNotFound) {
		t.Fatalf("RecordEvent() error = %v, want alias not found", err)
	}
	if got, want := err.Error(), "resolve ci_ref next-gen ci_id missing: alias not found"; got != want {
		t.Fatalf("RecordEvent() error = %q, want %q", got, want)
	}
}

func TestTimelineRejectsUnknownCI(t *testing.T) {
	configDir := t.TempDir()
	svc := New(configDir)

	_, err := svc.Timeline("MISSING")
	if !errors.Is(err, domain.ErrComponentNotFound) {
		t.Fatalf("Timeline() error = %v, want %v", err, domain.ErrComponentNotFound)
	}
}

func TestListCIsWithFilter(t *testing.T) {
	components := []domain.Component{
		{CIID: "FW-MAIN-001", Category: domain.CategoryCPU, Manufacturer: "Fortinet", Model: "FortiGate", Notes: "Primary edge firewall"},
		{CIID: "TWR-CORE-001", Category: "network", Manufacturer: "Cisco", Model: "Catalyst 9300", Notes: "Core switch"},
		{CIID: "SRV-DB-001", Category: "server", Manufacturer: "Dell", Model: "PowerEdge R750", Notes: "Database host"},
	}

	seed := func(t *testing.T) string {
		t.Helper()
		configDir := t.TempDir()
		if err := storage.SaveComponents(app.ComponentsPath(configDir), components); err != nil {
			t.Fatalf("SaveComponents() error = %v, want nil", err)
		}
		return configDir
	}

	t.Run("empty filter returns all", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != len(components) {
			t.Fatalf("len(result) = %d, want %d", len(got), len(components))
		}
	})

	t.Run("category exact match positive", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Category: "network"})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != 1 || got[0].CIID != "TWR-CORE-001" {
			t.Fatalf("result = %#v, want only TWR-CORE-001", got)
		}
	})

	t.Run("category exact match negative", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Category: "missing-category"})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Fatalf("result = %#v, want empty", got)
		}
	})

	t.Run("prefix match", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Prefix: "FW-"})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != 1 || got[0].CIID != "FW-MAIN-001" {
			t.Fatalf("result = %#v, want only FW-MAIN-001", got)
		}
	})

	t.Run("prefix is case sensitive", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Prefix: "fw-"})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Fatalf("result = %#v, want empty (prefix is case sensitive)", got)
		}
	})

	t.Run("query case-insensitive across fields", func(t *testing.T) {
		configDir := seed(t)
		tests := []struct {
			name  string
			query string
			want  string
		}{
			{name: "matches ci_id", query: "fw-main", want: "FW-MAIN-001"},
			{name: "matches model", query: "FORTI", want: "FW-MAIN-001"},
			{name: "matches notes", query: "core switch", want: "TWR-CORE-001"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := New(configDir).ListCIsWithFilter(ListFilter{Query: tt.query})
				if err != nil {
					t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
				}
				if len(got) != 1 || got[0].CIID != tt.want {
					t.Fatalf("result = %#v, want only %q", got, tt.want)
				}
			})
		}
	})

	t.Run("limit caps result", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Limit: 2})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != 2 {
			t.Fatalf("len(result) = %d, want 2", len(got))
		}
	})

	t.Run("limit zero means no cap", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Limit: 0})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != len(components) {
			t.Fatalf("len(result) = %d, want %d (no cap)", len(got), len(components))
		}
	})

	t.Run("limit negative means no cap", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Limit: -3})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != len(components) {
			t.Fatalf("len(result) = %d, want %d (negative limit means no cap)", len(got), len(components))
		}
	})

	t.Run("category plus prefix composes", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Category: "network", Prefix: "FW-"})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Fatalf("result = %#v, want empty (FW-* with category network does not match TWR-CORE-001)", got)
		}
	})

	t.Run("query plus limit composes", func(t *testing.T) {
		configDir := seed(t)
		got, err := New(configDir).ListCIsWithFilter(ListFilter{Query: "o", Limit: 1})
		if err != nil {
			t.Fatalf("ListCIsWithFilter() error = %v, want nil", err)
		}
		if len(got) != 1 {
			t.Fatalf("len(result) = %d, want 1 (limit applied after filter)", len(got))
		}
	})
}

func seedCIAndAlias(t *testing.T, configDir string) {
	t.Helper()
	if err := storage.SaveComponents(app.ComponentsPath(configDir), []domain.Component{{CIID: "FW-MAIN-001", Category: "network", Manufacturer: "Fortinet", Model: "FortiGate"}}); err != nil {
		t.Fatalf("SaveComponents() error = %v, want nil", err)
	}
	if err := storage.SaveAliases(app.AliasesPath(configDir), []domain.Alias{{CIID: "FW-MAIN-001", Source: "next-gen", Type: domain.AliasTypeCIID, Value: "42"}}); err != nil {
		t.Fatalf("SaveAliases() error = %v, want nil", err)
	}
}

func validEventWithoutIdentity() domain.Event {
	now := fixedNow()
	return domain.Event{
		Type:       "observation",
		Severity:   "info",
		Summary:    "test event",
		Source:     "test",
		ExternalID: "test-1",
		ObservedAt: now,
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 29, 20, 0, 0, 0, time.UTC)
}

func TestServiceGetCIMetadataExistingEntry(t *testing.T) {
	configDir := t.TempDir()
	attrs := map[string]domain.TypedValue{"azimuth": domain.StringValue("north")}
	rels := []domain.CIRelationship{{TargetCIID: "SRV-DB-001", Kind: "hosts"}}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: attrs, Relationships: rels}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	entry, err := New(configDir).GetCIMetadata("FW-MAIN-001")
	if err != nil {
		t.Fatalf("GetCIMetadata() error = %v, want nil", err)
	}
	if entry.CIID != "FW-MAIN-001" {
		t.Fatalf("GetCIMetadata() CIID = %q, want FW-MAIN-001", entry.CIID)
	}
	if got := entry.Attributes["azimuth"]; got.String == nil || *got.String != "north" {
		t.Fatalf("GetCIMetadata() Attributes[azimuth] = %#v, want StringValue(north)", got)
	}
	if len(entry.Relationships) != 1 || entry.Relationships[0].TargetCIID != "SRV-DB-001" || entry.Relationships[0].Kind != "hosts" {
		t.Fatalf("GetCIMetadata() Relationships = %#v, want seeded relationship", entry.Relationships)
	}
}

func TestServiceGetCIMetadataMissingReturnsEmpty(t *testing.T) {
	configDir := t.TempDir()

	entry, err := New(configDir).GetCIMetadata("FW-MAIN-001")
	if err != nil {
		t.Fatalf("GetCIMetadata() error = %v, want nil", err)
	}
	if entry.CIID != "FW-MAIN-001" {
		t.Fatalf("GetCIMetadata() CIID = %q, want FW-MAIN-001 (echoed even when missing)", entry.CIID)
	}
	if entry.Attributes != nil && len(entry.Attributes) != 0 {
		t.Fatalf("GetCIMetadata() Attributes = %#v, want empty", entry.Attributes)
	}
	if entry.Relationships != nil && len(entry.Relationships) != 0 {
		t.Fatalf("GetCIMetadata() Relationships = %#v, want empty", entry.Relationships)
	}
}

func TestServiceGetCIMetadataTrimsWhitespace(t *testing.T) {
	configDir := t.TempDir()
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: map[string]domain.TypedValue{"k": domain.StringValue("v")}}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	entry, err := New(configDir).GetCIMetadata("  FW-MAIN-001  ")
	if err != nil {
		t.Fatalf("GetCIMetadata() error = %v, want nil", err)
	}
	if entry.CIID != "FW-MAIN-001" {
		t.Fatalf("GetCIMetadata() CIID = %q, want FW-MAIN-001", entry.CIID)
	}
	if got := entry.Attributes["k"]; got.String == nil || *got.String != "v" {
		t.Fatalf("GetCIMetadata() Attributes[k] = %#v, want StringValue(v)", got)
	}
}

func TestServiceSetCIMetadataCreatesNewEntry(t *testing.T) {
	configDir := t.TempDir()
	svc := New(configDir)

	attrs := map[string]domain.TypedValue{"azimuth": domain.StringValue("north")}
	rels := []domain.CIRelationship{{TargetCIID: "SRV-DB-001", Kind: "hosts"}}
	if err := svc.SetCIMetadata("FW-MAIN-001", &attrs, &rels); err != nil {
		t.Fatalf("SetCIMetadata() error = %v, want nil", err)
	}

	stored, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(stored.Entries) != 1 {
		t.Fatalf("stored entries len = %d, want 1", len(stored.Entries))
	}
	if stored.Entries[0].CIID != "FW-MAIN-001" {
		t.Fatalf("stored CIID = %q, want FW-MAIN-001", stored.Entries[0].CIID)
	}
	if got := stored.Entries[0].Attributes["azimuth"]; got.String == nil || *got.String != "north" {
		t.Fatalf("stored Attributes[azimuth] = %#v, want StringValue(north)", got)
	}
	if len(stored.Entries[0].Relationships) != 1 || stored.Entries[0].Relationships[0].TargetCIID != "SRV-DB-001" {
		t.Fatalf("stored Relationships = %#v, want seeded relationship", stored.Entries[0].Relationships)
	}
}

func TestServiceSetCIMetadataReplacesEntireField(t *testing.T) {
	configDir := t.TempDir()
	seedAttrs := map[string]domain.TypedValue{
		"a": domain.StringValue("x"),
		"b": domain.NumberValue(2),
	}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: seedAttrs}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	newAttrs := map[string]domain.TypedValue{"c": domain.BoolValue(true)}
	if err := New(configDir).SetCIMetadata("FW-MAIN-001", &newAttrs, nil); err != nil {
		t.Fatalf("SetCIMetadata() error = %v, want nil", err)
	}

	stored, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(stored.Entries) != 1 {
		t.Fatalf("stored entries len = %d, want 1", len(stored.Entries))
	}
	got := stored.Entries[0].Attributes
	if _, present := got["a"]; present {
		t.Fatalf("stored Attributes still has %q = %#v, want removed", "a", got["a"])
	}
	if _, present := got["b"]; present {
		t.Fatalf("stored Attributes still has %q = %#v, want removed", "b", got["b"])
	}
	if val, ok := got["c"]; !ok || val.Bool == nil || !*val.Bool {
		t.Fatalf("stored Attributes[c] = %#v, want BoolValue(true)", val)
	}
}

func TestServiceSetCIMetadataNilAttributesPreservesExisting(t *testing.T) {
	configDir := t.TempDir()
	seedAttrs := map[string]domain.TypedValue{"a": domain.StringValue("x")}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: seedAttrs}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	rels := []domain.CIRelationship{{TargetCIID: "SRV-DB-001", Kind: "hosts"}}
	if err := New(configDir).SetCIMetadata("FW-MAIN-001", nil, &rels); err != nil {
		t.Fatalf("SetCIMetadata() error = %v, want nil", err)
	}

	stored, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(stored.Entries) != 1 {
		t.Fatalf("stored entries len = %d, want 1", len(stored.Entries))
	}
	if got := stored.Entries[0].Attributes["a"]; got.String == nil || *got.String != "x" {
		t.Fatalf("stored Attributes[a] = %#v, want preserved StringValue(x)", got)
	}
	if len(stored.Entries[0].Relationships) != 1 || stored.Entries[0].Relationships[0].TargetCIID != "SRV-DB-001" {
		t.Fatalf("stored Relationships = %#v, want seeded relationship", stored.Entries[0].Relationships)
	}
}

func TestServiceSetCIMetadataNilRelationshipsPreservesExisting(t *testing.T) {
	configDir := t.TempDir()
	seedRels := []domain.CIRelationship{{TargetCIID: "SRV-DB-001", Kind: "hosts"}}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Relationships: seedRels}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	attrs := map[string]domain.TypedValue{"a": domain.StringValue("x")}
	if err := New(configDir).SetCIMetadata("FW-MAIN-001", &attrs, nil); err != nil {
		t.Fatalf("SetCIMetadata() error = %v, want nil", err)
	}

	stored, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(stored.Entries) != 1 {
		t.Fatalf("stored entries len = %d, want 1", len(stored.Entries))
	}
	if got := stored.Entries[0].Attributes["a"]; got.String == nil || *got.String != "x" {
		t.Fatalf("stored Attributes[a] = %#v, want StringValue(x)", got)
	}
	if len(stored.Entries[0].Relationships) != 1 || stored.Entries[0].Relationships[0].TargetCIID != "SRV-DB-001" {
		t.Fatalf("stored Relationships = %#v, want preserved seeded relationship", stored.Entries[0].Relationships)
	}
}

func TestServiceSetCIMetadataEmptyMapClearsAttributes(t *testing.T) {
	configDir := t.TempDir()
	seedAttrs := map[string]domain.TypedValue{"a": domain.StringValue("x"), "b": domain.NumberValue(2)}
	if err := storage.SaveMetadata(app.MetadataPath(configDir), domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{CIID: "FW-MAIN-001", Attributes: seedAttrs}},
	}); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	empty := map[string]domain.TypedValue{}
	if err := New(configDir).SetCIMetadata("FW-MAIN-001", &empty, nil); err != nil {
		t.Fatalf("SetCIMetadata() error = %v, want nil", err)
	}

	stored, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(stored.Entries) != 1 {
		t.Fatalf("stored entries len = %d, want 1", len(stored.Entries))
	}
	if len(stored.Entries[0].Attributes) != 0 {
		t.Fatalf("stored Attributes = %#v, want cleared", stored.Entries[0].Attributes)
	}
}

func TestServiceSetCIMetadataRejectsDuplicateCIID(t *testing.T) {
	configDir := t.TempDir()
	svc := New(configDir)

	attrs := map[string]domain.TypedValue{"a": domain.StringValue("x")}
	if err := svc.SetCIMetadata("FW-MAIN-001", &attrs, nil); err != nil {
		t.Fatalf("first SetCIMetadata() error = %v, want nil", err)
	}
	attrs2 := map[string]domain.TypedValue{"b": domain.StringValue("y")}
	if err := svc.SetCIMetadata("FW-MAIN-001", &attrs2, nil); err != nil {
		t.Fatalf("second SetCIMetadata() error = %v, want nil (replace by design)", err)
	}

	stored, err := storage.LoadMetadata(app.MetadataPath(configDir))
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if len(stored.Entries) != 1 {
		t.Fatalf("stored entries len = %d, want 1 (replace semantics)", len(stored.Entries))
	}
	if got := stored.Entries[0].Attributes["a"]; got.String != nil {
		t.Fatalf("stored Attributes[a] = %#v, want removed", got)
	}
	if got := stored.Entries[0].Attributes["b"]; got.String == nil || *got.String != "y" {
		t.Fatalf("stored Attributes[b] = %#v, want StringValue(y)", got)
	}
}

func TestServiceSetCIMetadataRejectsInvalidTypedValue(t *testing.T) {
	configDir := t.TempDir()
	svc := New(configDir)

	tv := domain.TypedValue{Number: func() *float64 { v := 1.0; return &v }(), Bool: func() *bool { v := true; return &v }()}
	attrs := map[string]domain.TypedValue{"bad": tv}
	err := svc.SetCIMetadata("FW-MAIN-001", &attrs, nil)
	if !errors.Is(err, domain.ErrInvalidTypedValue) {
		t.Fatalf("SetCIMetadata() error = %v, want %v", err, domain.ErrInvalidTypedValue)
	}
}

func TestServiceSetCIMetadataRejectsSelfReferentialRelationship(t *testing.T) {
	configDir := t.TempDir()
	svc := New(configDir)

	rels := []domain.CIRelationship{{TargetCIID: "FW-MAIN-001", Kind: "self-ref"}}
	err := svc.SetCIMetadata("FW-MAIN-001", nil, &rels)
	if !errors.Is(err, domain.ErrSelfReferentialRelationship) {
		t.Fatalf("SetCIMetadata() error = %v, want %v", err, domain.ErrSelfReferentialRelationship)
	}
}
