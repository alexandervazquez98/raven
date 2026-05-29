package domain

import (
	"errors"
	"strings"
)

type ComponentCategory string

const (
	CategoryCPU         ComponentCategory = "cpu"
	CategoryMotherboard ComponentCategory = "motherboard"
	CategoryMemory      ComponentCategory = "memory"
	CategoryStorage     ComponentCategory = "storage"
	CategoryGPU         ComponentCategory = "gpu"
	CategoryPSU         ComponentCategory = "psu"
	CategoryCase        ComponentCategory = "case"
	CategoryOther       ComponentCategory = "other"
)

type Component struct {
	CIID         string            `json:"ci_id"`
	Category     ComponentCategory `json:"category"`
	Manufacturer string            `json:"manufacturer,omitempty"`
	Model        string            `json:"model"`
	SerialNumber string            `json:"serial_number,omitempty"`
	Notes        string            `json:"notes,omitempty"`
}

var (
	ErrMissingCategory = errors.New("component category is required")
	ErrMissingModel    = errors.New("component model is required")
)

func (c Component) Validate() error {
	if strings.TrimSpace(c.CIID) == "" {
		return ErrMissingCIID
	}
	if strings.TrimSpace(string(c.Category)) == "" {
		return ErrMissingCategory
	}
	if strings.TrimSpace(c.Model) == "" {
		return ErrMissingModel
	}
	return nil
}

func (c Component) DisplayName() string {
	maker := strings.TrimSpace(c.Manufacturer)
	model := strings.TrimSpace(c.Model)
	if maker == "" {
		return model
	}
	return maker + " " + model
}
