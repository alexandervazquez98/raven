package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingEventID         = errors.New("event id is required")
	ErrMissingEventType       = errors.New("event type is required")
	ErrMissingEventSeverity   = errors.New("event severity is required")
	ErrMissingEventSummary    = errors.New("event summary is required")
	ErrMissingEventSource     = errors.New("event source is required")
	ErrMissingEventDedupKey   = errors.New("event dedup key is required")
	ErrMissingObservedAt      = errors.New("event observed_at is required")
	ErrMissingIngestedAt      = errors.New("event ingested_at is required")
	ErrDuplicateEventDedupKey = errors.New("event dedup key already exists")
)

type Event struct {
	ID         string    `json:"id"`
	CIID       string    `json:"ci_id"`
	Type       string    `json:"type"`
	Severity   string    `json:"severity"`
	Status     string    `json:"status,omitempty"`
	Summary    string    `json:"summary"`
	Details    string    `json:"details,omitempty"`
	Source     string    `json:"source"`
	ExternalID string    `json:"external_id,omitempty"`
	DedupKey   string    `json:"dedup_key"`
	ObservedAt time.Time `json:"observed_at"`
	IngestedAt time.Time `json:"ingested_at"`
	Raw        string    `json:"raw,omitempty"`
}

func (e Event) Validate() error {
	e = e.Normalize()
	if e.ID == "" {
		return ErrMissingEventID
	}
	if e.CIID == "" {
		return ErrMissingCIID
	}
	if e.Type == "" {
		return ErrMissingEventType
	}
	if e.Severity == "" {
		return ErrMissingEventSeverity
	}
	if e.Summary == "" {
		return ErrMissingEventSummary
	}
	if e.Source == "" {
		return ErrMissingEventSource
	}
	if e.DedupKey == "" {
		return ErrMissingEventDedupKey
	}
	if e.ObservedAt.IsZero() {
		return ErrMissingObservedAt
	}
	if e.IngestedAt.IsZero() {
		return ErrMissingIngestedAt
	}
	return nil
}

func (e Event) Normalize() Event {
	e.ID = strings.TrimSpace(e.ID)
	e.CIID = strings.TrimSpace(e.CIID)
	e.Type = strings.TrimSpace(e.Type)
	e.Severity = strings.TrimSpace(e.Severity)
	e.Status = strings.TrimSpace(e.Status)
	e.Summary = strings.TrimSpace(e.Summary)
	e.Details = strings.TrimSpace(e.Details)
	e.Source = strings.TrimSpace(e.Source)
	e.ExternalID = strings.TrimSpace(e.ExternalID)
	e.DedupKey = strings.TrimSpace(e.DedupKey)
	e.Raw = strings.TrimSpace(e.Raw)
	return e
}
