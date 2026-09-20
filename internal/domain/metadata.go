package domain

import (
	"errors"
	"strings"
)

// MetadataSidecarVersion is the current schema version of metadata.json.
// Bump when the on-disk shape changes in a non-backward-compatible way.
const MetadataSidecarVersion = 1

var (
	ErrInvalidTypedValue             = errors.New("typed value must have exactly one of string, number, bool, enum")
	ErrMissingAttributeKey           = errors.New("attribute key is required")
	ErrMissingRelationshipTargetCIID = errors.New("relationship target_ci_id is required")
	ErrMissingRelationshipKind       = errors.New("relationship kind is required")
	ErrSelfReferentialRelationship   = errors.New("relationship cannot target its own ci_id")
	ErrUnsupportedSidecarVersion     = errors.New("unsupported sidecar version")
)

// TypedValue is a schema-validated scalar: exactly one field must be non-nil.
// Using a sum type instead of `any` keeps attribute values validated at write
// time, catching typos like `azimuth_deg` vs `azimuth` across operators.
type TypedValue struct {
	String *string  `json:"string,omitempty"`
	Number *float64 `json:"number,omitempty"`
	Bool   *bool    `json:"bool,omitempty"`
	Enum   *string  `json:"enum,omitempty"`
}

// StringValue constructs a TypedValue holding a string.
func StringValue(s string) TypedValue { return TypedValue{String: &s} }

// NumberValue constructs a TypedValue holding a float64.
func NumberValue(n float64) TypedValue { return TypedValue{Number: &n} }

// BoolValue constructs a TypedValue holding a bool.
func BoolValue(b bool) TypedValue { return TypedValue{Bool: &b} }

// EnumValue constructs a TypedValue holding a string-enum value.
func EnumValue(e string) TypedValue { return TypedValue{Enum: &e} }

// Raw returns the underlying scalar value of v as `any`. Returns nil if v has
// zero or multiple fields set (caller should Validate() first).
func (v TypedValue) Raw() any {
	switch {
	case v.String != nil:
		return *v.String
	case v.Number != nil:
		return *v.Number
	case v.Bool != nil:
		return *v.Bool
	case v.Enum != nil:
		return *v.Enum
	default:
		return nil
	}
}

// Validate ensures exactly one field is set.
func (v TypedValue) Validate() error {
	count := 0
	if v.String != nil {
		count++
	}
	if v.Number != nil {
		count++
	}
	if v.Bool != nil {
		count++
	}
	if v.Enum != nil {
		count++
	}
	if count != 1 {
		return ErrInvalidTypedValue
	}
	return nil
}

// CIRelationship links a CI to another CI by a named kind.
// Kind is intentionally a free-form string to preserve domain neutrality:
// operators pick their own vocabulary (telecom uses `located-at`/`hosts`,
// datacenters use `racked-in`/`powered-by`).
type CIRelationship struct {
	TargetCIID string `json:"target_ci_id"`
	Kind       string `json:"kind"`
}

// Validate ensures both fields are non-empty after trimming whitespace.
func (r CIRelationship) Validate() error {
	if strings.TrimSpace(r.TargetCIID) == "" {
		return ErrMissingRelationshipTargetCIID
	}
	if strings.TrimSpace(r.Kind) == "" {
		return ErrMissingRelationshipKind
	}
	return nil
}

// CIMetadataEntry holds optional attributes and relationships for one CI.
// Opt-in: a CI without metadata simply has no entry in the sidecar.
type CIMetadataEntry struct {
	CIID          string                `json:"ci_id"`
	Attributes    map[string]TypedValue `json:"attributes,omitempty"`
	Relationships []CIRelationship      `json:"relationships,omitempty"`
}

// Validate enforces non-empty ci_id, non-empty attribute keys, valid TypedValue
// per attribute, no self-referential relationships, and valid CIRelationship
// per relationship.
func (e CIMetadataEntry) Validate() error {
	if strings.TrimSpace(e.CIID) == "" {
		return ErrMissingCIID
	}
	for key, val := range e.Attributes {
		if strings.TrimSpace(key) == "" {
			return ErrMissingAttributeKey
		}
		if err := val.Validate(); err != nil {
			return err
		}
	}
	ciID := strings.TrimSpace(e.CIID)
	for _, rel := range e.Relationships {
		if strings.TrimSpace(rel.TargetCIID) == ciID {
			return ErrSelfReferentialRelationship
		}
		if err := rel.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// MetadataSidecar is the on-disk shape of metadata.json.
type MetadataSidecar struct {
	Version int               `json:"version"`
	Entries []CIMetadataEntry `json:"entries"`
}

// Validate enforces the schema version, then validates every entry, then
// ensures ci_id uniqueness across all entries.
func (s MetadataSidecar) Validate() error {
	if s.Version != MetadataSidecarVersion {
		return ErrUnsupportedSidecarVersion
	}
	seen := make(map[string]struct{}, len(s.Entries))
	for _, entry := range s.Entries {
		if err := entry.Validate(); err != nil {
			return err
		}
		ciID := strings.TrimSpace(entry.CIID)
		if _, exists := seen[ciID]; exists {
			return ErrDuplicateCIID
		}
		seen[ciID] = struct{}{}
	}
	return nil
}
