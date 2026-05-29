package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"raven/internal/domain"
)

func SaveComponents(path string, components []domain.Component) error {
	if err := validateComponents(components); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}

	data, err := json.MarshalIndent(components, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal components: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write components: %w", err)
	}
	return nil
}

func LoadComponents(path string) ([]domain.Component, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.Component{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read components: %w", err)
	}

	var components []domain.Component
	if err := json.Unmarshal(data, &components); err != nil {
		return nil, fmt.Errorf("decode components: %w", err)
	}

	if err := validateComponents(components); err != nil {
		return nil, err
	}
	return components, nil
}

func validateComponents(components []domain.Component) error {
	inventory := domain.NewInventory()
	for _, component := range components {
		if err := inventory.Add(component); err != nil {
			return err
		}
	}
	return nil
}
