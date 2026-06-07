package setuptui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"raven/internal/setup"
)

func TestModelInitialStateIsDetecting(t *testing.T) {
	model := New(Services{})

	if model.State() != StateDetecting {
		t.Fatalf("State() = %q, want %q", model.State(), StateDetecting)
	}
}

func TestDetectionTransitions(t *testing.T) {
	plan := setup.PlanResult{Items: []setup.PlanItem{{ID: "agents", Ecosystem: setup.EcosystemRavenAgents, Scope: setup.ScopeProjectLocal, Action: setup.ActionCreate}}}

	t.Run("success enters plan review and exposes plan items", func(t *testing.T) {
		model := New(Services{})
		updated, _ := model.Update(DetectionCompleteMsg{Plan: plan})
		model = updated.(Model)

		if model.State() != StatePlanReview {
			t.Fatalf("State() = %q, want %q", model.State(), StatePlanReview)
		}
		if model.PlanItemCount() != 1 {
			t.Fatalf("PlanItemCount() = %d, want 1", model.PlanItemCount())
		}
		if got := model.Plan().Items[0].ID; got != "agents" {
			t.Fatalf("plan item ID = %q, want agents", got)
		}
	})

	t.Run("failure enters failed state", func(t *testing.T) {
		model := New(Services{})
		updated, _ := model.Update(DetectionFailedMsg{Err: errors.New("boom")})
		model = updated.(Model)

		if model.State() != StateFailed {
			t.Fatalf("State() = %q, want %q", model.State(), StateFailed)
		}
		if model.Err() == nil || model.Err().Error() != "boom" {
			t.Fatalf("Err() = %v, want boom", model.Err())
		}
	})
}

func TestPlanApprovalTransitions(t *testing.T) {
	t.Run("plan with global items enters global approval", func(t *testing.T) {
		model := New(Services{})
		model, _ = updateAsModel(t, model, DetectionCompleteMsg{Plan: setup.PlanResult{Items: []setup.PlanItem{
			{ID: "global", Ecosystem: setup.EcosystemCodex, Scope: setup.ScopeUserGlobal, Action: setup.ActionCreate, TargetPath: "home/config", GeneratedContent: "global\n"},
		}}})

		model, cmd := updateAsModel(t, model, PlanApprovedMsg{})

		if model.State() != StateGlobalApproval {
			t.Fatalf("State() = %q, want %q", model.State(), StateGlobalApproval)
		}
		if cmd != nil {
			t.Fatalf("cmd = %#v, want nil until global approval", cmd)
		}
	})

	t.Run("plan without global items enters apply progress", func(t *testing.T) {
		called := false
		services := Services{Apply: func(plan setup.PlanResult, approval setup.ApplyApproval) setup.ApplySummary {
			called = true
			if !approval.Approved || approval.UserGlobalApproved {
				t.Fatalf("approval = %#v, want approved project-only", approval)
			}
			return setup.ApplySummary{}
		}}
		model := New(services)
		model, _ = updateAsModel(t, model, DetectionCompleteMsg{Plan: setup.PlanResult{Items: []setup.PlanItem{
			{ID: "project", Ecosystem: setup.EcosystemRavenAgents, Scope: setup.ScopeProjectLocal, Action: setup.ActionCreate, TargetPath: "project/AGENTS.md", GeneratedContent: "project\n"},
		}}})

		model, cmd := updateAsModel(t, model, PlanApprovedMsg{})

		if model.State() != StateApplyProgress {
			t.Fatalf("State() = %q, want %q", model.State(), StateApplyProgress)
		}
		if cmd == nil {
			t.Fatal("cmd = nil, want apply command")
		}
		_ = cmd()
		if !called {
			t.Fatal("apply service was not called")
		}
	})
}

func TestGlobalApprovalDeclineStillAppliesWithGlobalWritesSkipped(t *testing.T) {
	var gotApproval setup.ApplyApproval
	services := Services{Apply: func(plan setup.PlanResult, approval setup.ApplyApproval) setup.ApplySummary {
		gotApproval = approval
		return setup.ApplySummary{Skipped: []setup.ApplyItemResult{{ItemID: "global", Reason: "user-global writes were not approved"}}}
	}}
	model := New(services)
	model, _ = updateAsModel(t, model, DetectionCompleteMsg{Plan: setup.PlanResult{Items: []setup.PlanItem{
		{ID: "project", Ecosystem: setup.EcosystemRavenAgents, Scope: setup.ScopeProjectLocal, Action: setup.ActionCreate, TargetPath: "project/AGENTS.md", GeneratedContent: "project\n"},
		{ID: "global", Ecosystem: setup.EcosystemCodex, Scope: setup.ScopeUserGlobal, Action: setup.ActionCreate, TargetPath: "home/config", GeneratedContent: "global\n"},
	}}})
	model, _ = updateAsModel(t, model, PlanApprovedMsg{})

	model, cmd := updateAsModel(t, model, GlobalWritesConfirmedMsg{Approved: false})

	if model.State() != StateApplyProgress {
		t.Fatalf("State() = %q, want %q", model.State(), StateApplyProgress)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want apply command")
	}
	msg := cmd()
	if _, ok := msg.(ApplyCompleteMsg); !ok {
		t.Fatalf("cmd message = %T, want ApplyCompleteMsg", msg)
	}
	if !gotApproval.Approved || gotApproval.UserGlobalApproved {
		t.Fatalf("approval = %#v, want overall approved and global declined", gotApproval)
	}
}

func TestApplyAndValidationTransitionsToSummary(t *testing.T) {
	services := Services{Validate: func(items []setup.PlanItem) []setup.ValidationResult {
		return []setup.ValidationResult{{ItemID: "manual", Status: setup.ValidationManual, SmokeTestCommand: "ollama show raven-support"}}
	}}
	model := New(services)
	model, _ = updateAsModel(t, model, DetectionCompleteMsg{Plan: setup.PlanResult{Items: []setup.PlanItem{{ID: "manual", Ecosystem: setup.EcosystemOllama, Scope: setup.ScopeManual, Action: setup.ActionManual}}}})

	model, cmd := updateAsModel(t, model, ApplyCompleteMsg{Summary: setup.ApplySummary{
		Applied: []setup.ApplyItemResult{{ItemID: "applied", TargetPath: "AGENTS.md"}},
		Skipped: []setup.ApplyItemResult{{ItemID: "skipped", Reason: "manual"}},
		Failed:  []setup.ApplyItemResult{{ItemID: "failed", Reason: "bad"}},
	}})

	if model.State() != StateApplyProgress {
		t.Fatalf("State() = %q, want %q until validation completes", model.State(), StateApplyProgress)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want validation command")
	}
	validationMsg := cmd()
	model, _ = updateAsModel(t, model, validationMsg)

	if model.State() != StateValidationSummary {
		t.Fatalf("State() = %q, want %q", model.State(), StateValidationSummary)
	}
	view := model.View()
	for _, want := range []string{"Applied", "Skipped", "Failed", "Manual", "applied", "skipped", "failed", "ollama show raven-support"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func updateAsModel(t *testing.T, model Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := model.Update(msg)
	result, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	return result, cmd
}
