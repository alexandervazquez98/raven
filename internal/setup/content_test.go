package setup

import (
	"strings"
	"testing"
)

func TestUpsertManagedBlockInsertsAndDoesNotDuplicate(t *testing.T) {
	existing := "# Operator notes\nkeep me\n"
	generated := "raven: enabled\n"

	first, err := UpsertManagedBlock(existing, "agent-contract", generated)
	if err != nil {
		t.Fatalf("UpsertManagedBlock() error = %v", err)
	}
	second, err := UpsertManagedBlock(first, "agent-contract", generated)
	if err != nil {
		t.Fatalf("second UpsertManagedBlock() error = %v", err)
	}

	if second != first {
		t.Fatalf("rerun changed content\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if count := strings.Count(second, "BEGIN RAVEN MANAGED: agent-contract"); count != 1 {
		t.Fatalf("managed block count = %d, want 1\n%s", count, second)
	}
	if !strings.Contains(second, existing) {
		t.Fatalf("operator content was not preserved\n%s", second)
	}
}

func TestUpsertManagedBlockReplacesOnlyManagedBlock(t *testing.T) {
	existing := strings.Join([]string{
		"before",
		"<!-- BEGIN RAVEN MANAGED: agent-contract -->",
		"old raven content",
		"<!-- END RAVEN MANAGED: agent-contract -->",
		"after",
		"",
	}, "\n")

	updated, err := UpsertManagedBlock(existing, "agent-contract", "new raven content\n")
	if err != nil {
		t.Fatalf("UpsertManagedBlock() error = %v", err)
	}

	if !strings.Contains(updated, "before") || !strings.Contains(updated, "after") {
		t.Fatalf("user content not preserved\n%s", updated)
	}
	if strings.Contains(updated, "old raven content") {
		t.Fatalf("old managed content was preserved\n%s", updated)
	}
	if !strings.Contains(updated, "new raven content") {
		t.Fatalf("new managed content missing\n%s", updated)
	}
}

func TestUpsertManagedBlockRejectsDuplicateOrMalformedMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "duplicate block",
			content: strings.Join([]string{
				"<!-- BEGIN RAVEN MANAGED: agent-contract -->",
				"one",
				"<!-- END RAVEN MANAGED: agent-contract -->",
				"<!-- BEGIN RAVEN MANAGED: agent-contract -->",
				"two",
				"<!-- END RAVEN MANAGED: agent-contract -->",
			}, "\n"),
		},
		{
			name: "begin without end",
			content: strings.Join([]string{
				"operator",
				"<!-- BEGIN RAVEN MANAGED: agent-contract -->",
				"missing end",
			}, "\n"),
		},
		{
			name: "end without begin",
			content: strings.Join([]string{
				"operator",
				"<!-- END RAVEN MANAGED: agent-contract -->",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := UpsertManagedBlock(tt.content, "agent-contract", "new\n"); err == nil {
				t.Fatal("UpsertManagedBlock() error = nil, want marker error")
			}
		})
	}
}
