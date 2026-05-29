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
