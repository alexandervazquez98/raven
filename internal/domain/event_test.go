package domain

import (
	"errors"
	"testing"
	"time"
)

func TestEventValidate(t *testing.T) {
	now := time.Date(2026, 5, 28, 21, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		in   Event
		want error
	}{
		{
			name: "valid event",
			in: Event{
				ID:         "evt-001",
				CIID:       "RAVEN-DEV-001",
				Type:       "observation",
				Severity:   "info",
				Status:     "open",
				Summary:    "Initial Raven event recorded",
				Source:     "human",
				DedupKey:   "human:evt-001",
				ObservedAt: now,
				IngestedAt: now,
			},
		},
		{name: "missing id", in: Event{CIID: "RAVEN-DEV-001", Type: "observation", Severity: "info", Summary: "test", Source: "human", DedupKey: "human:test", ObservedAt: now, IngestedAt: now}, want: ErrMissingEventID},
		{name: "missing ci id", in: Event{ID: "evt-001", Type: "observation", Severity: "info", Summary: "test", Source: "human", DedupKey: "human:test", ObservedAt: now, IngestedAt: now}, want: ErrMissingCIID},
		{name: "missing type", in: Event{ID: "evt-001", CIID: "RAVEN-DEV-001", Severity: "info", Summary: "test", Source: "human", DedupKey: "human:test", ObservedAt: now, IngestedAt: now}, want: ErrMissingEventType},
		{name: "missing severity", in: Event{ID: "evt-001", CIID: "RAVEN-DEV-001", Type: "observation", Summary: "test", Source: "human", DedupKey: "human:test", ObservedAt: now, IngestedAt: now}, want: ErrMissingEventSeverity},
		{name: "missing summary", in: Event{ID: "evt-001", CIID: "RAVEN-DEV-001", Type: "observation", Severity: "info", Source: "human", DedupKey: "human:test", ObservedAt: now, IngestedAt: now}, want: ErrMissingEventSummary},
		{name: "missing source", in: Event{ID: "evt-001", CIID: "RAVEN-DEV-001", Type: "observation", Severity: "info", Summary: "test", DedupKey: "human:test", ObservedAt: now, IngestedAt: now}, want: ErrMissingEventSource},
		{name: "missing dedup key", in: Event{ID: "evt-001", CIID: "RAVEN-DEV-001", Type: "observation", Severity: "info", Summary: "test", Source: "human", ObservedAt: now, IngestedAt: now}, want: ErrMissingEventDedupKey},
		{name: "missing observed at", in: Event{ID: "evt-001", CIID: "RAVEN-DEV-001", Type: "observation", Severity: "info", Summary: "test", Source: "human", DedupKey: "human:test", IngestedAt: now}, want: ErrMissingObservedAt},
		{name: "missing ingested at", in: Event{ID: "evt-001", CIID: "RAVEN-DEV-001", Type: "observation", Severity: "info", Summary: "test", Source: "human", DedupKey: "human:test", ObservedAt: now}, want: ErrMissingIngestedAt},
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

func TestEventNormalize(t *testing.T) {
	now := time.Date(2026, 5, 28, 21, 0, 0, 0, time.UTC)
	event := Event{ID: " evt-001 ", CIID: " RAVEN-DEV-001 ", Type: " observation ", Severity: " info ", Status: " open ", Summary: " test ", Source: " human ", ExternalID: " ext-1 ", DedupKey: " human:ext-1 ", ObservedAt: now, IngestedAt: now}

	got := event.Normalize()
	if got.ID != "evt-001" || got.CIID != "RAVEN-DEV-001" || got.Type != "observation" || got.DedupKey != "human:ext-1" {
		t.Fatalf("Normalize() = %#v, want trimmed identity fields", got)
	}
}
