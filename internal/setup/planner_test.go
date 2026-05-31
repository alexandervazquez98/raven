package setup

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type fakeCommands map[string]string

func (f fakeCommands) LookPath(name string) (string, bool) {
	path, ok := f[name]
	return path, ok
}

func TestPlanIncludesTargetedEcosystemsAndMetadataWithoutMutation(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	env := SetupEnv{
		ProjectDir: projectDir,
		HomeDir:    homeDir,
		GOOS:       "linux",
		Commands: fakeCommands{
			"ollama": "/usr/bin/ollama",
			"gemini": "/usr/bin/gemini",
			"codex":  "/usr/bin/codex",
		},
		FS: OSFileSystem{},
	}
	before := snapshotTree(t, projectDir, homeDir)

	plan, err := Plan(env)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	wantEcosystems := []Ecosystem{EcosystemOllama, EcosystemGeminiCLI, EcosystemCodex, EcosystemAntigravity, EcosystemRavenAgents}
	for _, ecosystem := range wantEcosystems {
		if !hasEcosystem(plan, ecosystem) {
			t.Fatalf("plan missing ecosystem %q; items: %#v", ecosystem, plan.Items)
		}
	}

	for _, item := range plan.Items {
		if item.Ecosystem == "" {
			t.Fatalf("item %q has empty ecosystem", item.ID)
		}
		if item.Action == "" {
			t.Fatalf("item %q has empty action", item.ID)
		}
		if item.Scope == "" {
			t.Fatalf("item %q has empty scope", item.ID)
		}
		if item.ValidationMethod == "" && item.SmokeTestCommand == "" && item.ManualWarning == "" {
			t.Fatalf("item %q lacks validation, smoke test, or manual warning", item.ID)
		}
		if item.IsWritable() && item.TargetPath == "" {
			t.Fatalf("writable item %q has empty target path", item.ID)
		}
	}

	after := snapshotTree(t, projectDir, homeDir)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("planning mutated filesystem\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func TestPlanActionsReflectExistingFiles(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, ".gemini", "settings.json"), `{"existing":true}`)
	writeFile(t, filepath.Join(projectDir, "ollama", "Modelfile.raven"), "# BEGIN RAVEN MANAGED\n")

	plan, err := Plan(SetupEnv{ProjectDir: projectDir, HomeDir: homeDir, GOOS: "linux", Commands: fakeCommands{}, FS: OSFileSystem{}})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	tests := []struct {
		id   string
		want Action
	}{
		{id: "ollama-modelfile", want: ActionSkip},
		{id: "gemini-settings", want: ActionUpdate},
		{id: "codex-agents", want: ActionCreate},
		{id: "antigravity-guidance", want: ActionManual},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			item, ok := findItem(plan, tt.id)
			if !ok {
				t.Fatalf("plan item %q not found", tt.id)
			}
			if item.Action != tt.want {
				t.Fatalf("item %q action = %q, want %q", tt.id, item.Action, tt.want)
			}
		})
	}
}

func snapshotTree(t *testing.T, roots ...string) []string {
	t.Helper()
	var entries []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			entries = append(entries, root+":"+rel)
			return nil
		})
		if err != nil {
			t.Fatalf("snapshot %s: %v", root, err)
		}
	}
	sort.Strings(entries)
	return entries
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func hasEcosystem(plan PlanResult, ecosystem Ecosystem) bool {
	for _, item := range plan.Items {
		if item.Ecosystem == ecosystem {
			return true
		}
	}
	return false
}

func findItem(plan PlanResult, id string) (PlanItem, bool) {
	for _, item := range plan.Items {
		if item.ID == id {
			return item, true
		}
	}
	return PlanItem{}, false
}
