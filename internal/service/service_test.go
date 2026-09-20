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
