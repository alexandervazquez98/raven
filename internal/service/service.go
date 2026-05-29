package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"raven/internal/app"
	"raven/internal/domain"
	"raven/internal/storage"
)

var (
	ErrMissingEventIdentity = errors.New("event requires ci_id or ci_ref")
	ErrMissingEventDedup    = errors.New("event requires external_id or dedup_key")
)

type Service struct {
	ConfigDir string
	Now       func() time.Time
}

type CIRef struct {
	Source string           `json:"source"`
	Type   domain.AliasType `json:"type"`
	Value  string           `json:"value"`
}

type RecordEventInput struct {
	domain.Event
	CIRef          *CIRef `json:"ci_ref,omitempty"`
	SourceOverride string `json:"-"`
}

func New(configDir string) Service {
	return Service{ConfigDir: configDir}
}

func BuildDedupKey(source, externalID, eventID string) string {
	source = strings.TrimSpace(source)
	externalID = strings.TrimSpace(externalID)
	if externalID != "" {
		return source + ":" + externalID
	}
	return source + ":" + eventID
}

func (s Service) ListCIs() ([]domain.Component, error) {
	return storage.LoadComponents(app.ComponentsPath(s.ConfigDir))
}

func (s Service) GetCI(ciID string) (domain.Component, error) {
	_, inventory, err := s.loadInventory()
	if err != nil {
		return domain.Component{}, err
	}
	return inventory.Get(ciID)
}

func (s Service) ResolveCIRef(ref CIRef) (string, error) {
	_, registry, err := s.loadAliasRegistry()
	if err != nil {
		return "", err
	}
	ciID, err := registry.Resolve(ref.AliasKey())
	if err != nil {
		return "", fmt.Errorf("resolve ci_ref %s %s %s: %w", ref.Source, ref.Type, ref.Value, err)
	}
	return ciID, nil
}

func (s Service) RecordEvent(input RecordEventInput) (domain.Event, error) {
	event := input.Event.Normalize()
	if strings.TrimSpace(input.SourceOverride) != "" {
		event.Source = strings.TrimSpace(input.SourceOverride)
	}

	now := s.now()
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt-%d", now.UnixNano())
	}
	if event.Status == "" {
		event.Status = "open"
	}
	if event.ExternalID == "" && event.DedupKey == "" {
		return domain.Event{}, ErrMissingEventDedup
	}
	if event.ExternalID != "" {
		event.DedupKey = BuildDedupKey(event.Source, event.ExternalID, event.ID)
	}
	if event.IngestedAt.IsZero() {
		event.IngestedAt = now
	}
	if event.CIID == "" {
		if input.CIRef == nil {
			return domain.Event{}, ErrMissingEventIdentity
		}
		ciID, err := s.ResolveCIRef(*input.CIRef)
		if err != nil {
			return domain.Event{}, err
		}
		event.CIID = ciID
	}

	_, inventory, err := s.loadInventory()
	if err != nil {
		return domain.Event{}, err
	}
	if _, err := inventory.Get(event.CIID); err != nil {
		return domain.Event{}, err
	}

	events, err := storage.LoadEvents(app.EventsPath(s.ConfigDir))
	if err != nil {
		return domain.Event{}, err
	}
	events = append(events, event)
	if err := storage.SaveEvents(app.EventsPath(s.ConfigDir), events); err != nil {
		return domain.Event{}, err
	}
	return event, nil
}

func (s Service) Timeline(ciID string) ([]domain.Event, error) {
	ciID = strings.TrimSpace(ciID)
	_, inventory, err := s.loadInventory()
	if err != nil {
		return nil, err
	}
	if _, err := inventory.Get(ciID); err != nil {
		return nil, err
	}

	events, err := storage.LoadEvents(app.EventsPath(s.ConfigDir))
	if err != nil {
		return nil, err
	}
	matched := make([]domain.Event, 0)
	for _, event := range events {
		if event.CIID == ciID {
			matched = append(matched, event)
		}
	}
	return matched, nil
}

func (r CIRef) AliasKey() domain.AliasKey {
	return domain.AliasKey{Source: r.Source, Type: r.Type, Value: r.Value}
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) loadInventory() ([]domain.Component, *domain.Inventory, error) {
	components, err := storage.LoadComponents(app.ComponentsPath(s.ConfigDir))
	if err != nil {
		return nil, nil, err
	}

	inventory := domain.NewInventory()
	for _, component := range components {
		if err := inventory.Add(component); err != nil {
			return nil, nil, err
		}
	}
	return components, inventory, nil
}

func (s Service) loadAliasRegistry() ([]domain.Alias, *domain.AliasRegistry, error) {
	aliases, err := storage.LoadAliases(app.AliasesPath(s.ConfigDir))
	if err != nil {
		return nil, nil, err
	}

	registry := domain.NewAliasRegistry()
	for _, alias := range aliases {
		if err := registry.Add(alias); err != nil {
			return nil, nil, err
		}
	}
	return aliases, registry, nil
}
