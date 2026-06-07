package setup

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type fakeCommands map[string]string

func (f fakeCommands) LookPath(name string) (string, bool) {
	path, ok := f[name]
	return path, ok
}

type alwaysMissingFS struct{}

func (alwaysMissingFS) ReadFile(string) ([]byte, error)  { return nil, fs.ErrNotExist }
func (alwaysMissingFS) Stat(string) (fs.FileInfo, error) { return nil, fs.ErrNotExist }

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

	wantEcosystems := []Ecosystem{
		EcosystemOllama,
		EcosystemGeminiCLI,
		EcosystemCodex,
		EcosystemAntigravity,
		EcosystemRavenAgents,
	}
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
	writeFile(t, filepath.Join(projectDir, "ollama", "Modelfile.raven"), "# "+RavenManagedMarker+"\n")

	plan, err := Plan(SetupEnv{
		ProjectDir: projectDir,
		HomeDir:    homeDir,
		GOOS:       "linux",
		Commands:   fakeCommands{},
		FS:         OSFileSystem{},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	tests := []struct {
		id   string
		want Action
	}{
		{id: "ollama-modelfile", want: ActionSkip},
		{id: "gemini-settings", want: ActionManual},
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

func TestPlanIncludesGeneratedContentAndManagedIdentifiers(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	plan, err := Plan(SetupEnv{ProjectDir: projectDir, HomeDir: homeDir, GOOS: "linux", Commands: fakeCommands{}, FS: OSFileSystem{}})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	tests := []struct {
		id             string
		wantGenerated  bool
		wantManagedID  string
		wantValidation string
		wantScope      Scope
		wantAction     Action
		wantManualWarn bool
	}{
		{id: "gemini-settings", wantGenerated: true, wantValidation: "json-parse", wantScope: ScopeProjectLocal},
		{id: "codex-agents", wantGenerated: true, wantManagedID: "codex-agents", wantValidation: "managed-block-present", wantScope: ScopeProjectLocal},
		{id: "ollama-modelfile", wantGenerated: true, wantValidation: "managed-file-present", wantScope: ScopeProjectLocal},
		{id: "raven-agent-contract", wantGenerated: true, wantValidation: "managed-file-present", wantScope: ScopeProjectLocal},
		{id: "raven-incident-skill", wantGenerated: true, wantManagedID: "raven-incident-skill", wantValidation: "managed-block-present", wantScope: ScopeProjectLocal},
		{id: "codex-global-guidance", wantScope: ScopeUserGlobal, wantAction: ActionManual, wantManualWarn: true},
		{id: "antigravity-guidance", wantScope: ScopeManual, wantAction: ActionManual, wantManualWarn: true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			item, ok := findItem(plan, tt.id)
			if !ok {
				t.Fatalf("plan item %q not found", tt.id)
			}
			if tt.wantGenerated && strings.TrimSpace(item.GeneratedContent) == "" {
				t.Fatalf("item %q has no generated content", tt.id)
			}
			if item.ManagedBlockID != tt.wantManagedID {
				t.Fatalf("managed block id = %q, want %q", item.ManagedBlockID, tt.wantManagedID)
			}
			if tt.wantValidation != "" && item.ValidationMethod != tt.wantValidation {
				t.Fatalf("validation method = %q, want %q", item.ValidationMethod, tt.wantValidation)
			}
			if item.Scope != tt.wantScope {
				t.Fatalf("scope = %q, want %q", item.Scope, tt.wantScope)
			}
			if tt.wantAction != "" && item.Action != tt.wantAction {
				t.Fatalf("action = %q, want %q", item.Action, tt.wantAction)
			}
			if tt.wantManualWarn && item.ManualWarning == "" {
				t.Fatal("manual warning is empty")
			}
		})
	}
}

func TestPlanUsesInjectedPlatformPaths(t *testing.T) {
	tests := []struct {
		name           string
		goos           string
		projectDir     string
		homeDir        string
		wantGeminiPath string
		wantCodexPath  string
	}{
		{name: "linux", goos: "linux", projectDir: "/work/raven", homeDir: "/home/operator", wantGeminiPath: "/work/raven/.gemini/settings.json", wantCodexPath: "/home/operator/.codex/config.toml"},
		{name: "darwin", goos: "darwin", projectDir: "/Users/operator/raven", homeDir: "/Users/operator", wantGeminiPath: "/Users/operator/raven/.gemini/settings.json", wantCodexPath: "/Users/operator/.codex/config.toml"},
		{name: "windows", goos: "windows", projectDir: `C:\repo\raven`, homeDir: `C:\Users\operator`, wantGeminiPath: `C:\repo\raven\.gemini\settings.json`, wantCodexPath: `C:\Users\operator\.codex\config.toml`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := Plan(SetupEnv{ProjectDir: tt.projectDir, HomeDir: tt.homeDir, GOOS: tt.goos, Commands: fakeCommands{}, FS: alwaysMissingFS{}})
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			gemini, ok := findItem(plan, "gemini-settings")
			if !ok {
				t.Fatal("gemini-settings not found")
			}
			if gemini.TargetPath != tt.wantGeminiPath {
				t.Fatalf("gemini path = %q, want %q", gemini.TargetPath, tt.wantGeminiPath)
			}
			codex, ok := findItem(plan, "codex-global-guidance")
			if !ok {
				t.Fatal("codex-global-guidance not found")
			}
			if codex.TargetPath != tt.wantCodexPath {
				t.Fatalf("codex path = %q, want %q", codex.TargetPath, tt.wantCodexPath)
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
