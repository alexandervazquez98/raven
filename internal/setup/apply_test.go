package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyRequiresOverallApproval(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	plan := PlanResult{Items: []PlanItem{
		writableItem("project", ScopeProjectLocal, filepath.Join(projectDir, "AGENTS.md"), "project content\n"),
		writableItem("global", ScopeUserGlobal, filepath.Join(homeDir, ".codex", "config.toml"), "global content\n"),
	}}

	result := Apply(plan, ApplyApproval{}, SetupEnv{ProjectDir: projectDir, HomeDir: homeDir, FS: OSFileSystem{}})

	if len(result.Failed) != 0 || len(result.Applied) != 0 {
		t.Fatalf("Apply() applied/failed = %#v/%#v, want no writes and no failures", result.Applied, result.Failed)
	}
	if len(result.Skipped) != 2 {
		t.Fatalf("skipped = %d, want 2", len(result.Skipped))
	}
	assertNotExists(t, filepath.Join(projectDir, "AGENTS.md"))
	assertNotExists(t, filepath.Join(homeDir, ".codex", "config.toml"))
}

func TestApplyGlobalConfirmationGate(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	projectPath := filepath.Join(projectDir, "AGENTS.md")
	globalPath := filepath.Join(homeDir, ".codex", "config.toml")
	plan := PlanResult{Items: []PlanItem{
		writableItem("project", ScopeProjectLocal, projectPath, "project content\n"),
		writableItem("global", ScopeUserGlobal, globalPath, "global content\n"),
	}}

	result := Apply(plan, ApplyApproval{Approved: true, UserGlobalApproved: false}, SetupEnv{ProjectDir: projectDir, HomeDir: homeDir, FS: OSFileSystem{}})

	if len(result.Applied) != 1 || result.Applied[0].ItemID != "project" {
		t.Fatalf("applied = %#v, want only project item", result.Applied)
	}
	if len(result.Skipped) != 1 || result.Skipped[0].ItemID != "global" {
		t.Fatalf("skipped = %#v, want only global item", result.Skipped)
	}
	assertFileContent(t, projectPath, "project content\n")
	assertNotExists(t, globalPath)
}

func TestApplyAllowsGlobalWritesAfterConfirmation(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	globalPath := filepath.Join(homeDir, ".codex", "config.toml")
	plan := PlanResult{Items: []PlanItem{
		writableItem("global", ScopeUserGlobal, globalPath, "global content\n"),
	}}

	result := Apply(plan, ApplyApproval{Approved: true, UserGlobalApproved: true}, SetupEnv{ProjectDir: projectDir, HomeDir: homeDir, FS: OSFileSystem{}})

	if len(result.Applied) != 1 || result.Applied[0].TargetPath != globalPath {
		t.Fatalf("applied = %#v, want global write", result.Applied)
	}
	assertFileContent(t, globalPath, "global content\n")
}

func TestApplyNeverWritesManualOrSkipItems(t *testing.T) {
	projectDir := t.TempDir()
	manualPath := filepath.Join(projectDir, "manual.txt")
	skipPath := filepath.Join(projectDir, "skip.txt")
	plan := PlanResult{Items: []PlanItem{
		{ID: "manual", Ecosystem: EcosystemAntigravity, Scope: ScopeManual, Action: ActionManual, TargetPath: manualPath, GeneratedContent: "manual\n", ManualWarning: "Review this guidance manually"},
		{ID: "skip", Ecosystem: EcosystemOllama, Scope: ScopeProjectLocal, Action: ActionSkip, TargetPath: skipPath, GeneratedContent: "skip\n"},
	}}

	result := Apply(plan, ApplyApproval{Approved: true, UserGlobalApproved: true}, SetupEnv{ProjectDir: projectDir, FS: OSFileSystem{}})

	if len(result.Applied) != 0 {
		t.Fatalf("applied = %#v, want none", result.Applied)
	}
	if len(result.Skipped) != 2 {
		t.Fatalf("skipped = %d, want 2", len(result.Skipped))
	}
	if got, want := result.Skipped[0].Reason, "Review this guidance manually"; got != want {
		t.Fatalf("manual reason = %q, want %q", got, want)
	}
	if got, want := result.Skipped[1].Reason, "already configured"; got != want {
		t.Fatalf("skip reason = %q, want %q", got, want)
	}
	assertNotExists(t, manualPath)
	assertNotExists(t, skipPath)
}

func TestApplyUsesManualWarningDefaultForManualItemsWithoutExplicitWarning(t *testing.T) {
	projectDir := t.TempDir()
	manualPath := filepath.Join(projectDir, "manual.txt")
	plan := PlanResult{Items: []PlanItem{
		{ID: "manual", Ecosystem: EcosystemAntigravity, Scope: ScopeProjectLocal, Action: ActionManual, TargetPath: manualPath, GeneratedContent: "manual\n"},
	}}

	result := Apply(plan, ApplyApproval{Approved: true, UserGlobalApproved: true}, SetupEnv{ProjectDir: projectDir, FS: OSFileSystem{}})

	if len(result.Applied) != 0 {
		t.Fatalf("applied = %#v, want none", result.Applied)
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("skipped = %d, want 1", len(result.Skipped))
	}
	if got := result.Skipped[0].Reason; !strings.Contains(got, "review manually") {
		t.Fatalf("manual reason = %q, want review message", got)
	}
	assertNotExists(t, manualPath)
}

func TestApplyUsesManualWarningForManualItems(t *testing.T) {
	projectDir := t.TempDir()
	manualPath := filepath.Join(projectDir, "manual.txt")
	plan := PlanResult{Items: []PlanItem{
		{ID: "manual", Ecosystem: EcosystemAntigravity, Scope: ScopeManual, Action: ActionManual, TargetPath: manualPath, GeneratedContent: "manual\n", ManualWarning: "Review this guidance manually"},
	}}

	result := Apply(plan, ApplyApproval{Approved: true, UserGlobalApproved: true}, SetupEnv{ProjectDir: projectDir, FS: OSFileSystem{}})

	if len(result.Applied) != 0 {
		t.Fatalf("applied = %#v, want none", result.Applied)
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("skipped = %d, want 1", len(result.Skipped))
	}
	if got := result.Skipped[0].Reason; got != "Review this guidance manually" {
		t.Fatalf("manual reason = %q, want %q", got, "Review this guidance manually")
	}
	assertNotExists(t, manualPath)
}

func TestApplyUsesManagedBlocksForExistingFiles(t *testing.T) {
	projectDir := t.TempDir()
	target := filepath.Join(projectDir, "AGENTS.md")
	writeFile(t, target, "operator notes\n")
	plan := PlanResult{Items: []PlanItem{
		writableItem("agents", ScopeProjectLocal, target, "raven instructions\n"),
	}}
	plan.Items[0].ManagedBlockID = "agents"

	result := Apply(plan, ApplyApproval{Approved: true}, SetupEnv{ProjectDir: projectDir, FS: OSFileSystem{}})

	if len(result.Applied) != 1 {
		t.Fatalf("applied = %#v, want one", result.Applied)
	}
	content := readFile(t, target)
	if want := "operator notes"; !strings.Contains(content, want) {
		t.Fatalf("content missing %q\n%s", want, content)
	}
	if want := "BEGIN RAVEN MANAGED: agents"; !strings.Contains(content, want) {
		t.Fatalf("content missing %q\n%s", want, content)
	}
}

func TestApplyKeepsExistingManagedBlockWhenApplyingAdditionalOne(t *testing.T) {
	projectDir := t.TempDir()
	target := filepath.Join(projectDir, "AGENTS.md")
	writeFile(t, target, "operator notes\n")
	plan := PlanResult{Items: []PlanItem{
		writableItem("agents", ScopeProjectLocal, target, "raven instructions\n"),
		writableItem("agent-guidance", ScopeProjectLocal, target, "raven guidance\n"),
	}}
	plan.Items[0].ManagedBlockID = "agents"
	plan.Items[1].ManagedBlockID = "agent-guidance"

	result := Apply(plan, ApplyApproval{Approved: true}, SetupEnv{ProjectDir: projectDir, FS: OSFileSystem{}})

	if len(result.Applied) != 2 {
		t.Fatalf("applied = %#v, want two", result.Applied)
	}
	content := readFile(t, target)
	if !strings.Contains(content, "BEGIN RAVEN MANAGED: agents") {
		t.Fatalf("content missing first managed block\n%s", content)
	}
	if !strings.Contains(content, "BEGIN RAVEN MANAGED: agent-guidance") {
		t.Fatalf("content missing second managed block\n%s", content)
	}
	if !strings.Contains(content, "operator notes") {
		t.Fatalf("operator content not preserved\n%s", content)
	}
}

func writableItem(id string, scope Scope, path, content string) PlanItem {
	return PlanItem{
		ID:               id,
		Ecosystem:        EcosystemRavenAgents,
		Scope:            scope,
		Action:           ActionCreate,
		TargetPath:       path,
		GeneratedContent: content,
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s exists or stat error = %v, want not exist", path, err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	if got := readFile(t, path); got != want {
		t.Fatalf("%s content = %q, want %q", path, got, want)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
