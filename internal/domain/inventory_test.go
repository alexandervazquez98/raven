package domain

import (
	"errors"
	"testing"
)

func TestInventoryAddAndList(t *testing.T) {
	inventory := NewInventory()
	component := Component{
		CIID:         "cpu-1",
		Category:     CategoryCPU,
		Manufacturer: "AMD",
		Model:        "Ryzen 7 7800X3D",
	}

	if err := inventory.Add(component); err != nil {
		t.Fatalf("Add() error = %v, want nil", err)
	}

	got := inventory.List()
	if len(got) != 1 {
		t.Fatalf("List() length = %d, want 1", len(got))
	}
	if got[0] != component {
		t.Fatalf("List()[0] = %#v, want %#v", got[0], component)
	}
}

func TestInventoryAddValidation(t *testing.T) {
	tests := []struct {
		name string
		in   Component
		want error
	}{
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
			name: "missing model",
			in:   Component{CIID: "cpu-1", Category: CategoryCPU},
			want: ErrMissingModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inventory := NewInventory()
			err := inventory.Add(tt.in)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Add() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestInventoryAddRejectsDuplicateCIID(t *testing.T) {
	inventory := NewInventory()
	first := Component{CIID: "gpu-1", Category: CategoryGPU, Manufacturer: "NVIDIA", Model: "RTX 4080"}
	duplicate := Component{CIID: "gpu-1", Category: CategoryGPU, Manufacturer: "NVIDIA", Model: "RTX 4090"}

	if err := inventory.Add(first); err != nil {
		t.Fatalf("Add(first) error = %v, want nil", err)
	}

	err := inventory.Add(duplicate)
	if !errors.Is(err, ErrDuplicateCIID) {
		t.Fatalf("Add(duplicate) error = %v, want %v", err, ErrDuplicateCIID)
	}

	got := inventory.List()
	if len(got) != 1 || got[0] != first {
		t.Fatalf("List() after duplicate = %#v, want only %#v", got, first)
	}
}

func TestInventoryNormalizesCIIDs(t *testing.T) {
	inventory := NewInventory()
	component := Component{CIID: " cpu-1 ", Category: CategoryCPU, Manufacturer: "AMD", Model: "Ryzen 7 7800X3D"}

	if err := inventory.Add(component); err != nil {
		t.Fatalf("Add() error = %v, want nil", err)
	}

	got, err := inventory.Get(" cpu-1 ")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.CIID != "cpu-1" {
		t.Fatalf("Get().CIID = %q, want %q", got.CIID, "cpu-1")
	}

	if err := inventory.Remove(" cpu-1 "); err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	if got := inventory.List(); len(got) != 0 {
		t.Fatalf("List() length after normalized remove = %d, want 0", len(got))
	}
}

func TestInventoryAddIsZeroValueSafe(t *testing.T) {
	var inventory Inventory
	component := Component{CIID: "case-1", Category: CategoryCase, Manufacturer: "Fractal", Model: "North"}

	if err := inventory.Add(component); err != nil {
		t.Fatalf("Add() error = %v, want nil", err)
	}

	got, err := inventory.Get("case-1")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != component {
		t.Fatalf("Get() = %#v, want %#v", got, component)
	}
}

func TestInventoryGet(t *testing.T) {
	inventory := NewInventory()
	component := Component{CIID: "memory-1", Category: CategoryMemory, Manufacturer: "G.Skill", Model: "Trident Z5"}
	if err := inventory.Add(component); err != nil {
		t.Fatalf("Add() error = %v, want nil", err)
	}

	got, err := inventory.Get("memory-1")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got != component {
		t.Fatalf("Get() = %#v, want %#v", got, component)
	}
}

func TestInventoryGetMissing(t *testing.T) {
	inventory := NewInventory()

	_, err := inventory.Get("missing")
	if !errors.Is(err, ErrComponentNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, ErrComponentNotFound)
	}
}

func TestInventoryRemove(t *testing.T) {
	inventory := NewInventory()
	component := Component{CIID: "storage-1", Category: CategoryStorage, Manufacturer: "Samsung", Model: "990 Pro"}
	if err := inventory.Add(component); err != nil {
		t.Fatalf("Add() error = %v, want nil", err)
	}

	if err := inventory.Remove("storage-1"); err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}

	if got := inventory.List(); len(got) != 0 {
		t.Fatalf("List() length after remove = %d, want 0", len(got))
	}
	_, err := inventory.Get("storage-1")
	if !errors.Is(err, ErrComponentNotFound) {
		t.Fatalf("Get() after remove error = %v, want %v", err, ErrComponentNotFound)
	}
}

func TestInventoryRemoveMissing(t *testing.T) {
	inventory := NewInventory()

	err := inventory.Remove("missing")
	if !errors.Is(err, ErrComponentNotFound) {
		t.Fatalf("Remove() error = %v, want %v", err, ErrComponentNotFound)
	}
}
