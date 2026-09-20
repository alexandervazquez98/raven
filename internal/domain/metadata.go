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
// per attribute, and valid CIRelationship per relationship.
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
	for _, rel := range e.Relationships {
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

// Validate enforces the schema version and validates every entry.
func (s MetadataSidecar) Validate() error {
	if s.Version != MetadataSidecarVersion {
		return ErrUnsupportedSidecarVersion
	}
	for _, entry := range s.Entries {
		if err := entry.Validate(); err != nil {
			return err
		}
	}
	return nil
}
