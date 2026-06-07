package setup

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

type ApplyApproval struct {
	Approved           bool
	UserGlobalApproved bool
}

type ApplySummary struct {
	Applied []ApplyItemResult
	Skipped []ApplyItemResult
	Failed  []ApplyItemResult
}

type ApplyItemResult struct {
	ItemID     string
	TargetPath string
	Reason     string
	RollbackID string
}

func Apply(plan PlanResult, approval ApplyApproval, env SetupEnv) ApplySummary {
	env = withDefaults(env)
	summary := ApplySummary{}
	for _, item := range plan.Items {
		result := ApplyItemResult{ItemID: item.ID, TargetPath: item.TargetPath, RollbackID: item.RollbackID()}
		if !item.IsWritable() {
			result.Reason = "item is manual or skipped"
			summary.Skipped = append(summary.Skipped, result)
			continue
		}
		if !approval.Approved {
			result.Reason = "setup plan was not approved"
			summary.Skipped = append(summary.Skipped, result)
			continue
		}
		if item.Scope == ScopeUserGlobal && !approval.UserGlobalApproved {
			result.Reason = "user-global writes were not approved"
			summary.Skipped = append(summary.Skipped, result)
			continue
		}
		if err := rejectSecrets(item.GeneratedContent); err != nil {
			result.Reason = err.Error()
			summary.Failed = append(summary.Failed, result)
			continue
		}
		if err := applyItem(item, env); err != nil {
			result.Reason = err.Error()
			summary.Failed = append(summary.Failed, result)
			continue
		}
		result.Reason = "applied"
		summary.Applied = append(summary.Applied, result)
	}
	return summary
}

func applyItem(item PlanItem, env SetupEnv) error {
	if item.TargetPath == "" {
		return fmt.Errorf("item %q has no target path", item.ID)
	}
	files, ok := env.FS.(WritableFileSystem)
	if !ok {
		return fmt.Errorf("setup filesystem does not support writes")
	}
	content := item.GeneratedContent
	if item.ManagedBlockID != "" {
		existing, err := env.FS.ReadFile(item.TargetPath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		updated, err := UpsertManagedBlock(string(existing), item.ManagedBlockID, item.GeneratedContent)
		if err != nil {
			return err
		}
		content = updated
	}
	if err := files.MkdirAll(filepath.Dir(item.TargetPath), 0o755); err != nil {
		return err
	}
	return files.WriteFile(item.TargetPath, []byte(content), 0o644)
}
