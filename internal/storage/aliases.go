package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"raven/internal/domain"
)

func SaveAliases(path string, aliases []domain.Alias) error {
	aliases, err := validateAliases(aliases)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create alias storage directory: %w", err)
	}

	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aliases: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write aliases: %w", err)
	}
	return nil
}

func LoadAliases(path string) ([]domain.Alias, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.Alias{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read aliases: %w", err)
	}

	var aliases []domain.Alias
	if err := json.Unmarshal(data, &aliases); err != nil {
		return nil, fmt.Errorf("decode aliases: %w", err)
	}

	aliases, err = validateAliases(aliases)
	if err != nil {
		return nil, err
	}
	return aliases, nil
}

func validateAliases(aliases []domain.Alias) ([]domain.Alias, error) {
	registry := domain.NewAliasRegistry()
	for _, alias := range aliases {
		if err := registry.Add(alias); err != nil {
			return nil, err
		}
	}
	return registry.List(), nil
}
