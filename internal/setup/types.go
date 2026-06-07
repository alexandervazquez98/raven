package setup

type Ecosystem string

const (
	EcosystemOllama      Ecosystem = "ollama"
	EcosystemGeminiCLI   Ecosystem = "gemini-cli"
	EcosystemCodex       Ecosystem = "codex"
	EcosystemAntigravity Ecosystem = "antigravity"
	EcosystemRavenAgents Ecosystem = "raven-agents"
)

type Scope string

const (
	ScopeProjectLocal Scope = "project-local"
	ScopeUserGlobal   Scope = "user-global"
	ScopeManual       Scope = "manual"
)

type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionSkip   Action = "skip"
	ActionManual Action = "manual"
)

const RavenManagedMarker = "BEGIN RAVEN MANAGED"

type PlanResult struct {
	Items []PlanItem
}

type PlanItem struct {
	ID               string
	Ecosystem        Ecosystem
	TargetPath       string
	Scope            Scope
	Action           Action
	ValidationMethod string
	SmokeTestCommand string
	ManualWarning    string
	ManagedBlockID   string
	GeneratedContent string
}

func (item PlanItem) RollbackID() string {
	if item.ManagedBlockID != "" {
		return item.ManagedBlockID
	}
	return item.ID
}

func (item PlanItem) IsWritable() bool {
	return (item.Action == ActionCreate || item.Action == ActionUpdate) && (item.Scope == ScopeProjectLocal || item.Scope == ScopeUserGlobal)
}
