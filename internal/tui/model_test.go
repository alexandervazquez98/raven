package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"raven/internal/domain"
)

func TestModelTracksWindowSize(t *testing.T) {
	updated, cmd := New("test").Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Fatalf("Update() cmd = %v, want nil", cmd)
	}

	model := updated.(Model)
	if model.Width != 120 || model.Height != 40 {
		t.Fatalf("size = %dx%d, want 120x40", model.Width, model.Height)
	}
}

func TestModelViewShowsVersion(t *testing.T) {
	view := New("test-version").View()
	if !strings.Contains(view, "Raven test-version") {
		t.Fatalf("View() = %q, want version", view)
	}
}

func TestModelViewShowsEmptyInventory(t *testing.T) {
	view := New("test-version").View()
	if !strings.Contains(view, "No components yet.") {
		t.Fatalf("View() = %q, want empty inventory message", view)
	}
}

func TestModelViewShowsComponents(t *testing.T) {
	components := []domain.Component{
		{CIID: "cpu-1", Category: domain.CategoryCPU, Manufacturer: "AMD", Model: "Ryzen 7 7800X3D"},
		{CIID: "storage-1", Category: domain.CategoryStorage, Manufacturer: "Samsung", Model: "990 Pro"},
	}

	view := New("test-version", components).View()
	for _, want := range []string{"Components", "AMD Ryzen 7 7800X3D", "Samsung 990 Pro"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() = %q, want %q", view, want)
		}
	}
}
