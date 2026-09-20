package domain

import (
	"errors"
	"testing"
)

func ptrString(s string) *string {
	return &s
}

func ptrFloat(f float64) *float64 {
	return &f
}

func ptrBool(b bool) *bool {
	return &b
}

func TestTypedValueValidate(t *testing.T) {
	s := "hello"
	n := 3.14
	b := true
	e := "active"

	tests := []struct {
		name    string
		in      TypedValue
		wantErr error
	}{
		{name: "valid string", in: TypedValue{String: &s}},
		{name: "valid number", in: TypedValue{Number: &n}},
		{name: "valid bool", in: TypedValue{Bool: &b}},
		{name: "valid enum", in: TypedValue{Enum: &e}},
		{name: "valid zero number", in: TypedValue{Number: ptrFloat(0)}},
		{name: "valid false bool", in: TypedValue{Bool: ptrBool(false)}},
		{name: "valid empty string", in: TypedValue{String: ptrString("")}},
		{name: "no fields set", in: TypedValue{}, wantErr: ErrInvalidTypedValue},
		{name: "two fields set", in: TypedValue{String: &s, Number: &n}, wantErr: ErrInvalidTypedValue},
		{name: "three fields set", in: TypedValue{String: &s, Number: &n, Bool: &b}, wantErr: ErrInvalidTypedValue},
		{name: "all four fields set", in: TypedValue{String: &s, Number: &n, Bool: &b, Enum: &e}, wantErr: ErrInvalidTypedValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCIRelationshipValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      CIRelationship
		wantErr error
	}{
		{name: "valid", in: CIRelationship{TargetCIID: "TWR-01", Kind: "located-at"}},
		{name: "missing target_ci_id", in: CIRelationship{Kind: "located-at"}, wantErr: ErrMissingRelationshipTargetCIID},
		{name: "whitespace target_ci_id", in: CIRelationship{TargetCIID: "   ", Kind: "located-at"}, wantErr: ErrMissingRelationshipTargetCIID},
		{name: "missing kind", in: CIRelationship{TargetCIID: "TWR-01"}, wantErr: ErrMissingRelationshipKind},
		{name: "whitespace kind", in: CIRelationship{TargetCIID: "TWR-01", Kind: "  "}, wantErr: ErrMissingRelationshipKind},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCIMetadataEntryValidate(t *testing.T) {
	attr := TypedValue{String: ptrString("north")}
	rel := CIRelationship{TargetCIID: "TWR-01", Kind: "located-at"}

	tests := []struct {
		name    string
		in      CIMetadataEntry
		wantErr error
	}{
		{
			name: "valid with attributes only",
			in: CIMetadataEntry{
				CIID:       "RADIO-TIJ-01",
				Attributes: map[string]TypedValue{"azimuth": attr},
			},
		},
		{
			name: "valid with relationships only",
			in: CIMetadataEntry{
				CIID:          "RADIO-TIJ-01",
				Relationships: []CIRelationship{rel},
			},
		},
		{
			name: "valid with both",
			in: CIMetadataEntry{
				CIID:          "RADIO-TIJ-01",
				Attributes:    map[string]TypedValue{"azimuth": attr},
				Relationships: []CIRelationship{rel},
			},
		},
		{
			name:    "valid with neither",
			in:      CIMetadataEntry{CIID: "RADIO-TIJ-01"},
		},
		{
			name:    "missing ci_id",
			in:      CIMetadataEntry{Attributes: map[string]TypedValue{"azimuth": attr}},
			wantErr: ErrMissingCIID,
		},
		{
			name:    "whitespace ci_id",
			in:      CIMetadataEntry{CIID: "  "},
			wantErr: ErrMissingCIID,
		},
		{
			name: "empty attribute key",
			in: CIMetadataEntry{
				CIID:       "RADIO-TIJ-01",
				Attributes: map[string]TypedValue{"": attr},
			},
			wantErr: ErrMissingAttributeKey,
		},
		{
			name: "whitespace attribute key",
			in: CIMetadataEntry{
				CIID:       "RADIO-TIJ-01",
				Attributes: map[string]TypedValue{"   ": attr},
			},
			wantErr: ErrMissingAttributeKey,
		},
		{
			name: "invalid attribute TypedValue (empty)",
			in: CIMetadataEntry{
				CIID:       "RADIO-TIJ-01",
				Attributes: map[string]TypedValue{"azimuth": {}},
			},
			wantErr: ErrInvalidTypedValue,
		},
		{
			name: "invalid attribute TypedValue (two fields)",
			in: CIMetadataEntry{
				CIID:       "RADIO-TIJ-01",
				Attributes: map[string]TypedValue{"azimuth": {String: ptrString("x"), Number: ptrFloat(1)}},
			},
			wantErr: ErrInvalidTypedValue,
		},
		{
			name: "invalid relationship (missing target)",
			in: CIMetadataEntry{
				CIID:          "RADIO-TIJ-01",
				Relationships: []CIRelationship{{Kind: "located-at"}},
			},
			wantErr: ErrMissingRelationshipTargetCIID,
		},
		{
			name: "invalid relationship (missing kind)",
			in: CIMetadataEntry{
				CIID:          "RADIO-TIJ-01",
				Relationships: []CIRelationship{{TargetCIID: "TWR-01"}},
			},
			wantErr: ErrMissingRelationshipKind,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataSidecarValidate(t *testing.T) {
	entry := CIMetadataEntry{CIID: "RADIO-TIJ-01"}

	tests := []struct {
		name    string
		in      MetadataSidecar
		wantErr error
	}{
		{name: "valid empty entries", in: MetadataSidecar{Version: MetadataSidecarVersion}},
		{name: "valid with entries", in: MetadataSidecar{Version: MetadataSidecarVersion, Entries: []CIMetadataEntry{entry}}},
		{name: "version zero", in: MetadataSidecar{Version: 0, Entries: []CIMetadataEntry{entry}}, wantErr: ErrUnsupportedSidecarVersion},
		{name: "version higher", in: MetadataSidecar{Version: 2, Entries: []CIMetadataEntry{entry}}, wantErr: ErrUnsupportedSidecarVersion},
		{name: "version negative", in: MetadataSidecar{Version: -1, Entries: []CIMetadataEntry{entry}}, wantErr: ErrUnsupportedSidecarVersion},
		{
			name: "valid version but invalid entry (missing ci_id)",
			in: MetadataSidecar{
				Version: MetadataSidecarVersion,
				Entries: []CIMetadataEntry{{}},
			},
			wantErr: ErrMissingCIID,
		},
		{
			name: "valid version but invalid entry (bad attribute)",
			in: MetadataSidecar{
				Version: MetadataSidecarVersion,
				Entries: []CIMetadataEntry{
					{CIID: "RADIO-01", Attributes: map[string]TypedValue{"": {String: ptrString("x")}}},
				},
			},
			wantErr: ErrMissingAttributeKey,
		},
		{
			name: "valid version but invalid entry (bad relationship)",
			in: MetadataSidecar{
				Version: MetadataSidecarVersion,
				Entries: []CIMetadataEntry{
					{CIID: "RADIO-01", Relationships: []CIRelationship{{}}},
				},
			},
			wantErr: ErrMissingRelationshipTargetCIID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
