package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"raven/internal/domain"
)

func TestSaveAndLoadComponents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "components.json")
	components := []domain.Component{
		{CIID: "cpu-1", Category: domain.CategoryCPU, Manufacturer: "AMD", Model: "Ryzen 7 7800X3D", SerialNumber: "CPU123", Notes: "main rig"},
		{CIID: "storage-1", Category: domain.CategoryStorage, Manufacturer: "Samsung", Model: "990 Pro"},
	}

	if err := SaveComponents(path, components); err != nil {
		t.Fatalf("SaveComponents() error = %v, want nil", err)
	}

	got, err := LoadComponents(path)
	if err != nil {
		t.Fatalf("LoadComponents() error = %v, want nil", err)
	}
	if len(got) != len(components) {
		t.Fatalf("LoadComponents() length = %d, want %d", len(got), len(components))
	}
	for i := range components {
		if got[i] != components[i] {
			t.Fatalf("LoadComponents()[%d] = %#v, want %#v", i, got[i], components[i])
		}
	}
}

func TestSaveComponentsCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "inventory", "components.json")
	components := []domain.Component{{CIID: "case-1", Category: domain.CategoryCase, Manufacturer: "Fractal", Model: "North"}}

	if err := SaveComponents(path, components); err != nil {
		t.Fatalf("SaveComponents() error = %v, want nil", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file stat error = %v, want nil", err)
	}
}

func TestSaveComponentsUsesLowercaseJSONFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "components.json")
	components := []domain.Component{{CIID: "psu-1", Category: domain.CategoryPSU, Manufacturer: "Corsair", Model: "RM850x", SerialNumber: "PSU123", Notes: "quiet"}}

	if err := SaveComponents(path, components); err != nil {
		t.Fatalf("SaveComponents() error = %v, want nil", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}
	content := string(data)
	for _, want := range []string{"\"ci_id\"", "\"category\"", "\"manufacturer\"", "\"model\"", "\"serial_number\"", "\"notes\""} {
		if !strings.Contains(content, want) {
			t.Fatalf("saved JSON = %s, want field %s", content, want)
		}
	}
	for _, unwanted := range []string{"\"ID\"", "\"CIID\"", "\"id\"", "\"Category\"", "\"Manufacturer\"", "\"Model\"", "\"SerialNumber\"", "\"Notes\""} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("saved JSON = %s, must not contain Go field %s", content, unwanted)
		}
	}
}

func TestLoadComponentsMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	got, err := LoadComponents(path)
	if err != nil {
		t.Fatalf("LoadComponents() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("LoadComponents() length = %d, want 0", len(got))
	}
}

func TestLoadComponentsRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "components.json")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	_, err := LoadComponents(path)
	if err == nil {
		t.Fatal("LoadComponents() error = nil, want error")
	}
}

func TestSaveComponentsValidatesComponents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "components.json")
	components := []domain.Component{{CIID: "cpu-1", Category: domain.CategoryCPU}}

	err := SaveComponents(path, components)
	if !errors.Is(err, domain.ErrMissingModel) {
		t.Fatalf("SaveComponents() error = %v, want %v", err, domain.ErrMissingModel)
	}
}

func TestSaveComponentsRejectsDuplicateCIIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "components.json")
	components := []domain.Component{
		{CIID: "gpu-1", Category: domain.CategoryGPU, Model: "RTX 4080"},
		{CIID: " gpu-1 ", Category: domain.CategoryGPU, Model: "RTX 4090"},
	}

	err := SaveComponents(path, components)
	if !errors.Is(err, domain.ErrDuplicateCIID) {
		t.Fatalf("SaveComponents() error = %v, want %v", err, domain.ErrDuplicateCIID)
	}
}

func TestLoadComponentsValidatesComponents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "components.json")
	if err := os.WriteFile(path, []byte(`[{"ci_id":"cpu-1","category":"cpu"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	_, err := LoadComponents(path)
	if !errors.Is(err, domain.ErrMissingModel) {
		t.Fatalf("LoadComponents() error = %v, want %v", err, domain.ErrMissingModel)
	}
}

func TestLoadComponentsRejectsDuplicateCIIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "components.json")
	if err := os.WriteFile(path, []byte(`[
		{"ci_id":"memory-1","category":"memory","model":"Trident Z5"},
		{"ci_id":" memory-1 ","category":"memory","model":"Vengeance"}
	]`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	_, err := LoadComponents(path)
	if !errors.Is(err, domain.ErrDuplicateCIID) {
		t.Fatalf("LoadComponents() error = %v, want %v", err, domain.ErrDuplicateCIID)
	}
}
