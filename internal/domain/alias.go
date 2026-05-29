package domain

import (
	"errors"
	"sort"
	"strings"
)

type AliasType string

const (
	AliasTypeCIID     AliasType = "ci_id"
	AliasTypeIP       AliasType = "ip"
	AliasTypeHostname AliasType = "hostname"
	AliasTypeSerial   AliasType = "serial"
	AliasTypeMAC      AliasType = "mac"
)

type Alias struct {
	CIID   string    `json:"ci_id"`
	Source string    `json:"source"`
	Type   AliasType `json:"type"`
	Value  string    `json:"value"`
}

type AliasKey struct {
	Source string
	Type   AliasType
	Value  string
}

var (
	ErrMissingAliasSource      = errors.New("alias source is required")
	ErrMissingAliasType        = errors.New("alias type is required")
	ErrMissingAliasValue       = errors.New("alias value is required")
	ErrUnsupportedAliasType    = errors.New("unsupported alias type")
	ErrDuplicateAliasKey       = errors.New("alias already exists")
	ErrConflictingAliasMapping = errors.New("alias conflicts with existing ci id")
	ErrAliasNotFound           = errors.New("alias not found")
)

func (a Alias) Normalize() Alias {
	a.CIID = strings.TrimSpace(a.CIID)
	a.Source = strings.ToLower(strings.TrimSpace(a.Source))
	a.Type = AliasType(strings.ToLower(strings.TrimSpace(string(a.Type))))
	a.Value = strings.TrimSpace(a.Value)
	return a
}

func (a Alias) Validate() error {
	a = a.Normalize()
	if a.CIID == "" {
		return ErrMissingCIID
	}
	if a.Source == "" {
		return ErrMissingAliasSource
	}
	if a.Type == "" {
		return ErrMissingAliasType
	}
	if !IsSupportedAliasType(a.Type) {
		return ErrUnsupportedAliasType
	}
	if a.Value == "" {
		return ErrMissingAliasValue
	}
	return nil
}

func (a Alias) Key() AliasKey {
	a = a.Normalize()
	return AliasKey{Source: a.Source, Type: a.Type, Value: a.Value}
}

func IsSupportedAliasType(aliasType AliasType) bool {
	switch AliasType(strings.ToLower(strings.TrimSpace(string(aliasType)))) {
	case AliasTypeCIID, AliasTypeIP, AliasTypeHostname, AliasTypeSerial, AliasTypeMAC:
		return true
	default:
		return false
	}
}

type AliasRegistry struct {
	aliases map[AliasKey]Alias
}

func NewAliasRegistry() *AliasRegistry {
	return &AliasRegistry{aliases: make(map[AliasKey]Alias)}
}

func (r *AliasRegistry) Add(alias Alias) error {
	alias = alias.Normalize()
	if err := alias.Validate(); err != nil {
		return err
	}
	r.ensureInitialized()
	key := alias.Key()
	if existing, exists := r.aliases[key]; exists {
		if existing.CIID == alias.CIID {
			return ErrDuplicateAliasKey
		}
		return ErrConflictingAliasMapping
	}
	r.aliases[key] = alias
	return nil
}

func (r *AliasRegistry) Resolve(key AliasKey) (string, error) {
	r.ensureInitialized()
	key = key.Normalize()
	if err := key.Validate(); err != nil {
		return "", err
	}
	alias, exists := r.aliases[key]
	if !exists {
		return "", ErrAliasNotFound
	}
	return alias.CIID, nil
}

func (r *AliasRegistry) List() []Alias {
	r.ensureInitialized()
	aliases := make([]Alias, 0, len(r.aliases))
	for _, alias := range r.aliases {
		aliases = append(aliases, alias)
	}
	SortAliases(aliases)
	return aliases
}

func (r *AliasRegistry) ensureInitialized() {
	if r.aliases == nil {
		r.aliases = make(map[AliasKey]Alias)
	}
}

func (k AliasKey) Normalize() AliasKey {
	return AliasKey{
		Source: strings.ToLower(strings.TrimSpace(k.Source)),
		Type:   AliasType(strings.ToLower(strings.TrimSpace(string(k.Type)))),
		Value:  strings.TrimSpace(k.Value),
	}
}

func (k AliasKey) Validate() error {
	k = k.Normalize()
	if k.Source == "" {
		return ErrMissingAliasSource
	}
	if k.Type == "" {
		return ErrMissingAliasType
	}
	if !IsSupportedAliasType(k.Type) {
		return ErrUnsupportedAliasType
	}
	if k.Value == "" {
		return ErrMissingAliasValue
	}
	return nil
}

func SortAliases(aliases []Alias) {
	for i := range aliases {
		aliases[i] = aliases[i].Normalize()
	}
	sort.Slice(aliases, func(i, j int) bool {
		if aliases[i].Source != aliases[j].Source {
			return aliases[i].Source < aliases[j].Source
		}
		if aliases[i].Type != aliases[j].Type {
			return aliases[i].Type < aliases[j].Type
		}
		if aliases[i].Value != aliases[j].Value {
			return aliases[i].Value < aliases[j].Value
		}
		return aliases[i].CIID < aliases[j].CIID
	})
}
