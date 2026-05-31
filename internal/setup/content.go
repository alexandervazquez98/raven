package setup

import (
	"fmt"
	"strings"
)

func UpsertManagedBlock(existing, blockID, generated string) (string, error) {
	begin := managedBlockBegin(blockID)
	end := managedBlockEnd(blockID)
	beginCount := strings.Count(existing, begin)
	endCount := strings.Count(existing, end)

	if beginCount != endCount {
		return "", fmt.Errorf("managed block %q has mismatched markers", blockID)
	}
	if beginCount > 1 {
		return "", fmt.Errorf("managed block %q appears more than once", blockID)
	}

	block := renderManagedBlock(blockID, generated)
	if beginCount == 0 {
		return appendBlock(existing, block), nil
	}

	beginAt := strings.Index(existing, begin)
	endAt := strings.Index(existing, end)
	if beginAt < 0 || endAt < 0 || endAt < beginAt {
		return "", fmt.Errorf("managed block %q has malformed markers", blockID)
	}
	endAt += len(end)
	return existing[:beginAt] + block + existing[endAt:], nil
}

func renderManagedBlock(blockID, generated string) string {
	return managedBlockBegin(blockID) + "\n" + strings.TrimRight(generated, "\n") + "\n" + managedBlockEnd(blockID)
}

func managedBlockBegin(blockID string) string {
	return fmt.Sprintf("<!-- BEGIN RAVEN MANAGED: %s -->", blockID)
}

func managedBlockEnd(blockID string) string {
	return fmt.Sprintf("<!-- END RAVEN MANAGED: %s -->", blockID)
}

func appendBlock(existing, block string) string {
	if strings.TrimSpace(existing) == "" {
		return block + "\n"
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + block + "\n"
}
