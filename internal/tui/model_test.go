package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"raven/internal/domain"
)

func TestModelTracksWindowSize(t *testing.T) {
	updated, cmd := New("test").Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Fatalf("Update() cmd = %v, want nil", cmd)
	}

	model := updated.(Model)
	if model.Width != 120 || model.Height != 40 {
		t.Fatalf("size = %dx%d, want 120x40", model.Width, model.Height)
	}
}

func TestModelViewShowsMenu(t *testing.T) {
	view := New("test-version").View()
	for _, want := range []string{"Raven test-version", "Main menu", "Install", "Inventory", "Recent Memories", "Search", "Exit", "up/down or j/k"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() = %q, want %q", view, want)
		}
	}
}

func TestMenuNavigationAndEnter(t *testing.T) {
	model := New("test")

	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	if model.selectedMenuItem != 1 {
		t.Fatalf("selectedMenuItem = %d, want 1", model.selectedMenuItem)
	}

	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if model.selectedMenuItem != 0 {
		t.Fatalf("selectedMenuItem = %d, want 0", model.selectedMenuItem)
	}

	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.activeSection != sectionSearch {
		t.Fatalf("activeSection = %v, want sectionSearch", model.activeSection)
	}
	if !strings.Contains(model.View(), "Search") {
		t.Fatalf("View() = %q, want search section", model.View())
	}
}

func TestExitSelectionIssuesQuit(t *testing.T) {
	model := New("test")
	model.selectedMenuItem = len(menuItems) - 1

	_, cmd := updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on Exit did not return command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("enter on Exit produced %T, want QuitMsg", cmd())
	}
}

func TestSearchSupportsMultiWordQueriesAndClearEmptyState(t *testing.T) {
	model := NewWithEvents("test", nil, []domain.Event{testMemory("evt-1", "alert", "disk failure", time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))})
	model.selectedMenuItem = 3
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if view := model.View(); strings.Contains(view, "evt-1") || !strings.Contains(view, "No query yet") {
		t.Fatalf("empty search View() = %q, want no query state without results", view)
	}

	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("disk")})
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeySpace})
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("failure")})
	if !strings.Contains(model.View(), "evt-1") {
		t.Fatalf("View() = %q, want evt-1 in multi-word search results", model.View())
	}

	for range len("disk failure") {
		model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	if model.searchQuery != "" || !strings.Contains(model.View(), "No query yet") {
		t.Fatalf("search after clearing = query %q view %q, want empty no-query state", model.searchQuery, model.View())
	}
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyBackspace})
	if model.activeSection != sectionSearch {
		t.Fatalf("activeSection = %v, want sectionSearch after empty Backspace", model.activeSection)
	}
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.activeSection != sectionMenu {
		t.Fatalf("activeSection = %v, want sectionMenu after Esc", model.activeSection)
	}
}

func TestSearchCanTypeQWithoutQuitting(t *testing.T) {
	model := NewWithEvents("test", nil, []domain.Event{testMemory("evt-q", "query router", "needs q search", time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))})
	model.selectedMenuItem = 3
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	updated, cmd := updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Fatalf("typing q returned cmd %v, want nil", cmd)
	}
	view := updated.View()
	if updated.searchQuery != "q" || !strings.Contains(view, "evt-q") {
		t.Fatalf("search after q = query %q view %q, want q query and result", updated.searchQuery, view)
	}
	if strings.Contains(view, "q/Ctrl-C to quit") {
		t.Fatalf("View() = %q, must not advertise q as quit in search mode", view)
	}
}

func TestSearchShowsTotalAndDisplayedLimit(t *testing.T) {
	events := make([]domain.Event, 0, searchResultLimit+2)
	for i := 0; i < searchResultLimit+2; i++ {
		events = append(events, testMemoryWithTime(i+1, time.Date(2026, 7, 1, 12, i, 0, 0, time.UTC)))
	}
	model := NewWithEvents("test", nil, events)
	model.selectedMenuItem = 3
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("event")})

	view := model.View()
	if !strings.Contains(view, "Matches: 22 (showing first 20)") {
		t.Fatalf("View() = %q, want total and displayed count", view)
	}
}

func TestInventoryViewPreservesComponentList(t *testing.T) {
	component := domain.Component{CIID: "FW-MAIN-001", Category: "network", Model: "Firewall"}
	model := NewWithEvents("test", []domain.Component{component}, nil)
	model.selectedMenuItem = 1
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	view := model.View()
	for _, want := range []string{"Inventory", "FW-MAIN-001"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() = %q, want %q", view, want)
		}
	}
}

func TestRecentMemoriesSortsByLatestAndLimitsToFive(t *testing.T) {
	events := make([]domain.Event, 0, 6)
	for i := 1; i <= 6; i++ {
		events = append(events, testMemoryWithTime(i, time.Date(2026, 7, i, 12, 0, 0, 0, time.UTC)))
	}

	model := NewWithEvents("test", nil, events)
	model.selectedMenuItem = 2
	model, _ = updateAsModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	view := model.View()

	if !strings.Contains(view, "evt-6") || !strings.Contains(view, "evt-2") {
		t.Fatalf("View() = %q, want evt-6 and evt-2", view)
	}
	if strings.Contains(view, "evt-1") {
		t.Fatal("oldest memory evt-1 should be outside 5-item recent limit")
	}
}

func TestNewEventOrderingPrefersObservedTimeOverIngested(t *testing.T) {
	events := []domain.Event{
		testMemoryWithObservedAndIngested("evt-old", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)),
		testMemoryWithObservedAndIngested("evt-new", time.Date(2026, 1, 2, 1, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)),
	}
	latest := latestMemories(events, 2)
	if latest[0].CIID != "evt-new" {
		t.Fatalf("latest first = %q, want evt-new", latest[0].CIID)
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

func testMemory(ciID, summary, details string, observed time.Time) domain.Event {
	return domain.Event{
		ID:         ciID + "-id",
		CIID:       ciID,
		Type:       "diagnostic",
		Severity:   "warning",
		Status:     "open",
		Summary:    summary,
		Details:    details,
		Source:     "test",
		DedupKey:   ciID + "-dedup",
		ObservedAt: observed,
		IngestedAt: observed,
	}
}

func testMemoryWithTime(index int, observed time.Time) domain.Event {
	return testMemory(
		fmt.Sprintf("evt-%d", index),
		fmt.Sprintf("event %d summary", index),
		fmt.Sprintf("event %d details", index),
		observed,
	)
}

func testMemoryWithObservedAndIngested(ciID string, observed, ingested time.Time) domain.Event {
	return domain.Event{
		ID:         ciID + "-id",
		CIID:       ciID,
		Type:       "diagnostic",
		Severity:   "warning",
		Status:     "open",
		Summary:    ciID + " summary",
		Details:    ciID + " details",
		Source:     "test",
		DedupKey:   ciID + "-dedup",
		ObservedAt: observed,
		IngestedAt: ingested,
	}
}
