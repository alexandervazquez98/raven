package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"raven/internal/domain"
)

func TestSaveAndLoadEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.json")
	events := []domain.Event{testEvent("evt-001", "RAVEN-DEV-001", "human:evt-001")}

	if err := SaveEvents(path, events); err != nil {
		t.Fatalf("SaveEvents() error = %v, want nil", err)
	}

	got, err := LoadEvents(path)
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0] != events[0] {
		t.Fatalf("LoadEvents() = %#v, want %#v", got, events)
	}
}

func TestLoadEventsMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	got, err := LoadEvents(path)
	if err != nil {
		t.Fatalf("LoadEvents() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadEvents() length = %d, want 0", len(got))
	}
}

func TestSaveEventsRejectsDuplicateDedupKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.json")
	events := []domain.Event{
		testEvent("evt-001", "RAVEN-DEV-001", "next-gen:123"),
		testEvent("evt-002", "RAVEN-DEV-001", " next-gen:123 "),
	}

	err := SaveEvents(path, events)
	if !errors.Is(err, domain.ErrDuplicateEventDedupKey) {
		t.Fatalf("SaveEvents() error = %v, want %v", err, domain.ErrDuplicateEventDedupKey)
	}
}

func TestLoadEventsRejectsInvalidEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.json")
	if err := os.WriteFile(path, []byte(`[{"id":"evt-001","ci_id":"RAVEN-DEV-001"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	_, err := LoadEvents(path)
	if !errors.Is(err, domain.ErrMissingEventType) {
		t.Fatalf("LoadEvents() error = %v, want %v", err, domain.ErrMissingEventType)
	}
}

func testEvent(id, ciID, dedupKey string) domain.Event {
	now := time.Date(2026, 5, 28, 21, 0, 0, 0, time.UTC)
	return domain.Event{
		ID:         id,
		CIID:       ciID,
		Type:       "observation",
		Severity:   "info",
		Status:     "open",
		Summary:    "Raven event recorded",
		Source:     "human",
		DedupKey:   dedupKey,
		ObservedAt: now,
		IngestedAt: now,
	}
}
