package setuptui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"raven/internal/setup"
)

type State string

const (
	StateDetecting         State = "detecting"
	StatePlanReview        State = "plan-review"
	StateGlobalApproval    State = "global-approval"
	StateApplyProgress     State = "apply-progress"
	StateValidationSummary State = "validation-summary"
	StateFailed            State = "failed"
)

type Services struct {
	Plan     func() (setup.PlanResult, error)
	Apply    func(setup.PlanResult, setup.ApplyApproval) setup.ApplySummary
	Validate func([]setup.PlanItem) []setup.ValidationResult
}

type DetectionCompleteMsg struct{ Plan setup.PlanResult }
type DetectionFailedMsg struct{ Err error }
type PlanApprovedMsg struct{}
type GlobalWritesConfirmedMsg struct{ Approved bool }
type ApplyCompleteMsg struct{ Summary setup.ApplySummary }
type ValidationCompleteMsg struct{ Results []setup.ValidationResult }

type Model struct {
	state      State
	services   Services
	plan       setup.PlanResult
	apply      setup.ApplySummary
	validation []setup.ValidationResult
	err        error
}

func New(services Services) Model { return Model{state: StateDetecting, services: services} }

func NewForEnv(env setup.SetupEnv) Model {
	return New(Services{
		Plan: func() (setup.PlanResult, error) { return setup.Plan(env) },
		Apply: func(plan setup.PlanResult, approval setup.ApplyApproval) setup.ApplySummary {
			return setup.Apply(plan, approval, env)
		},
		Validate: func(items []setup.PlanItem) []setup.ValidationResult { return setup.Validate(items, env) },
	})
}

func (m Model) State() State                     { return m.state }
func (m Model) Plan() setup.PlanResult           { return m.plan }
func (m Model) PlanItemCount() int               { return len(m.plan.Items) }
func (m Model) ApplySummary() setup.ApplySummary { return m.apply }
func (m Model) ValidationResults() []setup.ValidationResult {
	return append([]setup.ValidationResult(nil), m.validation...)
}
func (m Model) Err() error { return m.err }

func (m Model) Init() tea.Cmd { return m.detectCmd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DetectionCompleteMsg:
		m.plan, m.err, m.state = msg.Plan, nil, StatePlanReview
	case DetectionFailedMsg:
		m.err, m.state = msg.Err, StateFailed
	case PlanApprovedMsg:
		if m.hasUserGlobalWrites() {
			m.state = StateGlobalApproval
			return m, nil
		}
		m.state = StateApplyProgress
		return m, m.applyCmd(setup.ApplyApproval{Approved: true})
	case GlobalWritesConfirmedMsg:
		m.state = StateApplyProgress
		return m, m.applyCmd(setup.ApplyApproval{Approved: true, UserGlobalApproved: msg.Approved})
	case ApplyCompleteMsg:
		m.apply, m.state = msg.Summary, StateApplyProgress
		return m, m.validateCmd()
	case ValidationCompleteMsg:
		m.validation, m.state = append([]setup.ValidationResult(nil), msg.Results...), StateValidationSummary
	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "enter":
		if m.state == StatePlanReview {
			return m.Update(PlanApprovedMsg{})
		}
		if m.state == StateGlobalApproval {
			return m.Update(GlobalWritesConfirmedMsg{Approved: false})
		}
	case "y":
		if m.state == StateGlobalApproval {
			return m.Update(GlobalWritesConfirmedMsg{Approved: true})
		}
	case "n":
		if m.state == StateGlobalApproval {
			return m.Update(GlobalWritesConfirmedMsg{Approved: false})
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("Raven setup\n\n")
	switch m.state {
	case StateDetecting:
		b.WriteString("Detecting AI integrations...\n")
	case StatePlanReview:
		fmt.Fprintf(&b, "Plan Review: %d item(s)\n", len(m.plan.Items))
		for _, item := range m.plan.Items {
			fmt.Fprintf(&b, "- %s [%s/%s]\n", item.ID, item.Scope, item.Action)
		}
	case StateGlobalApproval:
		b.WriteString("User-global writes require separate approval. Press y to approve, n/enter to skip.\n")
	case StateApplyProgress:
		b.WriteString("Applying setup plan...\n")
	case StateValidationSummary:
		writeSummary(&b, m.apply, m.validation)
	case StateFailed:
		b.WriteString("Setup failed.\n")
		if m.err != nil {
			fmt.Fprintf(&b, "%v\n", m.err)
		}
	}
	b.WriteString("\nPress q to quit.\n")
	return b.String()
}

func (m Model) detectCmd() tea.Cmd {
	return func() tea.Msg {
		if m.services.Plan == nil {
			return DetectionCompleteMsg{}
		}
		plan, err := m.services.Plan()
		if err != nil {
			return DetectionFailedMsg{Err: err}
		}
		return DetectionCompleteMsg{Plan: plan}
	}
}

func (m Model) applyCmd(approval setup.ApplyApproval) tea.Cmd {
	return func() tea.Msg {
		if m.services.Apply == nil {
			return ApplyCompleteMsg{}
		}
		return ApplyCompleteMsg{Summary: m.services.Apply(m.plan, approval)}
	}
}

func (m Model) validateCmd() tea.Cmd {
	return func() tea.Msg {
		if m.services.Validate == nil {
			return ValidationCompleteMsg{}
		}
		return ValidationCompleteMsg{Results: m.services.Validate(m.plan.Items)}
	}
}

func (m Model) hasUserGlobalWrites() bool {
	for _, item := range m.plan.Items {
		if item.Scope == setup.ScopeUserGlobal && item.IsWritable() {
			return true
		}
	}
	return false
}

func writeSummary(b *strings.Builder, summary setup.ApplySummary, validation []setup.ValidationResult) {
	for _, section := range []struct {
		name  string
		items []setup.ApplyItemResult
	}{
		{"Applied", summary.Applied}, {"Skipped", summary.Skipped}, {"Failed", summary.Failed},
	} {
		b.WriteString(section.name + "\n")
		writeApplyItems(b, section.items)
	}
	b.WriteString("Manual\n")
	count := 0
	for _, result := range validation {
		if result.Status == setup.ValidationManual {
			count++
			fmt.Fprintf(b, "- %s: %s\n", result.ItemID, result.SmokeTestCommand)
		}
	}
	if count == 0 {
		b.WriteString("- none\n")
	}
}

func writeApplyItems(b *strings.Builder, items []setup.ApplyItemResult) {
	if len(items) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, item := range items {
		fmt.Fprintf(b, "- %s", item.ItemID)
		if item.TargetPath != "" {
			fmt.Fprintf(b, " -> %s", item.TargetPath)
		}
		if item.Reason != "" {
			fmt.Fprintf(b, " (%s)", item.Reason)
		}
		b.WriteString("\n")
	}
}
