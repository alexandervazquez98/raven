package app

import (
	"path/filepath"
	"testing"
)

func TestComponentsPathUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()

	got := ComponentsPath(configDir)
	want := filepath.Join(configDir, "raven", "components.json")
	if got != want {
		t.Fatalf("ComponentsPath() = %q, want %q", got, want)
	}
}

func TestEventsPathUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()

	got := EventsPath(configDir)
	want := filepath.Join(configDir, "raven", "events.json")
	if got != want {
		t.Fatalf("EventsPath() = %q, want %q", got, want)
	}
}

func TestAliasesPathUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()

	got := AliasesPath(configDir)
	want := filepath.Join(configDir, "raven", "aliases.json")
	if got != want {
		t.Fatalf("AliasesPath() = %q, want %q", got, want)
	}
}
