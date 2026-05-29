package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"raven/internal/domain"
)

func SaveEvents(path string, events []domain.Event) error {
	events, err := validateEvents(events)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create event storage directory: %w", err)
	}

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write events: %w", err)
	}
	return nil
}

func LoadEvents(path string) ([]domain.Event, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.Event{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read events: %w", err)
	}

	var events []domain.Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}

	events, err = validateEvents(events)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func validateEvents(events []domain.Event) ([]domain.Event, error) {
	seen := make(map[string]struct{}, len(events))
	normalized := make([]domain.Event, 0, len(events))
	for _, event := range events {
		event = event.Normalize()
		if err := event.Validate(); err != nil {
			return nil, err
		}
		if _, exists := seen[event.DedupKey]; exists {
			return nil, domain.ErrDuplicateEventDedupKey
		}
		seen[event.DedupKey] = struct{}{}
		normalized = append(normalized, event)
	}
	return normalized, nil
}
