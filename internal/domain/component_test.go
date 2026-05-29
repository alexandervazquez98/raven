package domain

import (
	"errors"
	"testing"
)

func TestComponentValidate(t *testing.T) {
	tests := []struct {
		name string
		in   Component
		want error
	}{
		{
			name: "valid component",
			in:   Component{CIID: "cpu-1", Category: CategoryCPU, Manufacturer: "AMD", Model: "Ryzen 7 7800X3D"},
		},
		{
			name: "missing ci id",
			in:   Component{Category: CategoryCPU, Model: "Ryzen 7 7800X3D"},
			want: ErrMissingCIID,
		},
		{
			name: "missing category",
			in:   Component{CIID: "cpu-1", Model: "Ryzen 7 7800X3D"},
			want: ErrMissingCategory,
		},
		{
			name: "flexible cmdb category",
			in:   Component{CIID: "ups-1", Category: ComponentCategory("power"), Model: "Smart-UPS 1500"},
		},
		{
			name: "missing model",
			in:   Component{CIID: "cpu-1", Category: CategoryCPU},
			want: ErrMissingModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestComponentDisplayName(t *testing.T) {
	tests := []struct {
		name string
		in   Component
		want string
	}{
		{name: "manufacturer and model", in: Component{Manufacturer: "Kingston", Model: "KC3000 2TB"}, want: "Kingston KC3000 2TB"},
		{name: "model only", in: Component{Model: "RM850x"}, want: "RM850x"},
		{name: "trims whitespace", in: Component{Manufacturer: "  Corsair ", Model: " RM850x "}, want: "Corsair RM850x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.DisplayName(); got != tt.want {
				t.Fatalf("DisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}
