package storage

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"raven/internal/domain"
)

func newSampleSidecar() domain.MetadataSidecar {
	return domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{
			{
				CIID: "TWR-TIJ-01",
				Attributes: map[string]domain.TypedValue{
					"azimuth_deg": domain.NumberValue(45),
					"channel_mhz": domain.NumberValue(5800),
					"gps_sync":    domain.BoolValue(true),
					"status":      domain.EnumValue("active"),
				},
				Relationships: []domain.CIRelationship{
					{TargetCIID: "SITE-TIJ-CENTRO", Kind: "located-at"},
				},
			},
			{
				CIID: "AP-TIJ-SEC1",
				Relationships: []domain.CIRelationship{
					{TargetCIID: "TWR-TIJ-01", Kind: "mounted-on"},
				},
			},
		},
	}
}

func TestSaveAndLoadMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	sidecar := newSampleSidecar()

	if err := SaveMetadata(path, sidecar); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	got, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if got.Version != sidecar.Version {
		t.Fatalf("LoadMetadata() Version = %d, want %d", got.Version, sidecar.Version)
	}
	if len(got.Entries) != len(sidecar.Entries) {
		t.Fatalf("LoadMetadata() entries length = %d, want %d", len(got.Entries), len(sidecar.Entries))
	}
	for i := range sidecar.Entries {
		if got.Entries[i].CIID != sidecar.Entries[i].CIID {
			t.Fatalf("LoadMetadata() Entries[%d].CIID = %q, want %q", i, got.Entries[i].CIID, sidecar.Entries[i].CIID)
		}
		if len(got.Entries[i].Attributes) != len(sidecar.Entries[i].Attributes) {
			t.Fatalf("LoadMetadata() Entries[%d].Attributes length = %d, want %d", i, len(got.Entries[i].Attributes), len(sidecar.Entries[i].Attributes))
		}
		for key, wantVal := range sidecar.Entries[i].Attributes {
			gotVal, ok := got.Entries[i].Attributes[key]
			if !ok {
				t.Fatalf("LoadMetadata() Entries[%d].Attributes missing key %q", i, key)
			}
			if !reflect.DeepEqual(gotVal, wantVal) {
				t.Fatalf("LoadMetadata() Entries[%d].Attributes[%q] = %#v, want %#v", i, key, gotVal, wantVal)
			}
		}
		if len(got.Entries[i].Relationships) != len(sidecar.Entries[i].Relationships) {
			t.Fatalf("LoadMetadata() Entries[%d].Relationships length = %d, want %d", i, len(got.Entries[i].Relationships), len(sidecar.Entries[i].Relationships))
		}
		for j := range sidecar.Entries[i].Relationships {
			if got.Entries[i].Relationships[j] != sidecar.Entries[i].Relationships[j] {
				t.Fatalf("LoadMetadata() Entries[%d].Relationships[%d] = %#v, want %#v", i, j, got.Entries[i].Relationships[j], sidecar.Entries[i].Relationships[j])
			}
		}
	}
}

func TestSaveMetadataCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "metadata", "metadata.json")
	sidecar := newSampleSidecar()

	if err := SaveMetadata(path, sidecar); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file stat error = %v, want nil", err)
	}
}

func TestSaveMetadataUsesLowercaseJSONFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	sidecar := newSampleSidecar()

	if err := SaveMetadata(path, sidecar); err != nil {
		t.Fatalf("SaveMetadata() error = %v, want nil", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}
	content := string(data)
	for _, want := range []string{"\"ci_id\"", "\"target_ci_id\"", "\"kind\"", "\"attributes\"", "\"relationships\"", "\"entries\"", "\"version\""} {
		if !strings.Contains(content, want) {
			t.Fatalf("saved JSON = %s, want field %s", content, want)
		}
	}
	for _, unwanted := range []string{"\"CIID\"", "\"TargetCIID\"", "\"TypedValue\"", "\"CIRelationship\"", "\"CIMetadataEntry\"", "\"MetadataSidecar\"", "\"String\"", "\"Number\"", "\"Bool\"", "\"Enum\""} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("saved JSON = %s, must not contain Go field %s", content, unwanted)
		}
	}
}

func TestLoadMetadataMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	got, err := LoadMetadata(path)
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v, want nil", err)
	}
	if got.Version != domain.MetadataSidecarVersion {
		t.Fatalf("LoadMetadata() Version = %d, want %d", got.Version, domain.MetadataSidecarVersion)
	}
	if len(got.Entries) != 0 {
		t.Fatalf("LoadMetadata() entries length = %d, want 0", len(got.Entries))
	}
}

func TestLoadMetadataRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	_, err := LoadMetadata(path)
	if err == nil {
		t.Fatal("LoadMetadata() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "decode metadata") {
		t.Fatalf("LoadMetadata() error = %v, want substring %q", err, "decode metadata")
	}
}

func TestSaveMetadataValidatesMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")

	// Empty sidecar with version=1 is valid.
	empty := domain.MetadataSidecar{Version: domain.MetadataSidecarVersion}
	if err := SaveMetadata(path, empty); err != nil {
		t.Fatalf("SaveMetadata(empty) error = %v, want nil", err)
	}

	// Invalid entry (missing ci_id) must be rejected.
	invalid := domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{{}},
	}
	err := SaveMetadata(path, invalid)
	if !errors.Is(err, domain.ErrMissingCIID) {
		t.Fatalf("SaveMetadata(invalid) error = %v, want %v", err, domain.ErrMissingCIID)
	}
}

func TestLoadMetadataValidatesMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"entries":[{}]}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	_, err := LoadMetadata(path)
	if !errors.Is(err, domain.ErrMissingCIID) {
		t.Fatalf("LoadMetadata() error = %v, want %v", err, domain.ErrMissingCIID)
	}
}

func TestLoadMetadataRejectsUnsupportedVersion(t *testing.T) {
	for _, badVersion := range []int{0, 2} {
		t.Run("version", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metadata.json")
			body := []byte(`{"version":` + strconv.Itoa(badVersion) + `,"entries":[]}`)
			if err := os.WriteFile(path, body, 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v, want nil", err)
			}

			_, err := LoadMetadata(path)
			if !errors.Is(err, domain.ErrUnsupportedSidecarVersion) {
				t.Fatalf("LoadMetadata(version=%d) error = %v, want %v", badVersion, err, domain.ErrUnsupportedSidecarVersion)
			}
		})
	}
}

func TestSaveMetadataRejectsDuplicateCIIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	sidecar := domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{
			{CIID: "TWR-TIJ-01"},
			{CIID: " TWR-TIJ-01 "},
		},
	}

	err := SaveMetadata(path, sidecar)
	if !errors.Is(err, domain.ErrDuplicateCIID) {
		t.Fatalf("SaveMetadata() error = %v, want %v", err, domain.ErrDuplicateCIID)
	}
}

func TestSaveMetadataRejectsSelfReferentialRelationship(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	sidecar := domain.MetadataSidecar{
		Version: domain.MetadataSidecarVersion,
		Entries: []domain.CIMetadataEntry{
			{
				CIID: "AP-TIJ-SEC1",
				Relationships: []domain.CIRelationship{
					{TargetCIID: "AP-TIJ-SEC1", Kind: "mounted-on"},
				},
			},
		},
	}

	err := SaveMetadata(path, sidecar)
	if !errors.Is(err, domain.ErrSelfReferentialRelationship) {
		t.Fatalf("SaveMetadata() error = %v, want %v", err, domain.ErrSelfReferentialRelationship)
	}
}

