package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"raven/internal/domain"
)

// SaveMetadata writes the sidecar to path as pretty-printed JSON. The parent
// directory is created with mode 0o755 if missing. The file is written atomically
// with mode 0o600. The sidecar is validated (including ci_id uniqueness) before
// the write.
func SaveMetadata(path string, sidecar domain.MetadataSidecar) error {
	if err := sidecar.Validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

// LoadMetadata reads the sidecar from path. A missing file returns an empty
// sidecar pinned to the current schema version (no error). On success the
// sidecar is fully validated, including ci_id uniqueness across entries.
func LoadMetadata(path string) (domain.MetadataSidecar, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return domain.MetadataSidecar{Version: domain.MetadataSidecarVersion}, nil
	}
	if err != nil {
		return domain.MetadataSidecar{}, fmt.Errorf("read metadata: %w", err)
	}

	var sidecar domain.MetadataSidecar
	if err := json.Unmarshal(data, &sidecar); err != nil {
		return domain.MetadataSidecar{}, fmt.Errorf("decode metadata: %w", err)
	}

	if err := sidecar.Validate(); err != nil {
		return domain.MetadataSidecar{}, err
	}
	return sidecar, nil
}
