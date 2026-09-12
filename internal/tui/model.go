package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"raven/internal/domain"
)

const (
	recentMemoriesLimit = 5
	searchResultLimit   = 20
)

type section int

const (
	sectionMenu section = iota
	sectionInstall
	sectionInventory
	sectionRecentMemories
	sectionSearch
)

type menuItem struct {
	Label   string
	Section section
	Exit    bool
}

var menuItems = []menuItem{
	{Label: "Install", Section: sectionInstall},
	{Label: "Inventory", Section: sectionInventory},
	{Label: "Recent Memories", Section: sectionRecentMemories},
	{Label: "Search", Section: sectionSearch},
	{Label: "Exit", Exit: true},
}

type Model struct {
	Version    string
	Width      int
	Height     int
	Components []domain.Component
	Events     []domain.Event

	activeSection    section
	selectedMenuItem int
	searchQuery      string
}

func New(version string, components ...[]domain.Component) Model {
	model := Model{Version: version}
	if len(components) > 0 {
		model.Components = append([]domain.Component(nil), components[0]...)
	}
	return model
}

func NewWithEvents(version string, components []domain.Component, events []domain.Event) Model {
	model := New(version, components)
	model.Events = append([]domain.Event(nil), events...)
	return model
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case tea.KeyMsg:
		if m.activeSection == sectionMenu {
			return m.updateMenu(msg)
		}
		return m.updateSection(msg)
	}
	return m, nil
}

func (m Model) View() string {
	switch m.activeSection {
	case sectionInstall:
		return m.renderInstallView()
	case sectionInventory:
		return m.renderInventoryView()
	case sectionRecentMemories:
		return m.renderRecentMemoriesView()
	case sectionSearch:
		return m.renderSearchView()
	default:
		return m.renderMenuView()
	}
}

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.selectedMenuItem > 0 {
			m.selectedMenuItem--
		}
	case "down", "j":
		if m.selectedMenuItem < len(menuItems)-1 {
			m.selectedMenuItem++
		}
	case "enter":
		selected := menuItems[m.selectedMenuItem]
		if selected.Exit {
			return m, tea.Quit
		}
		m.activeSection = selected.Section
	}
	return m, nil
}

func (m Model) updateSection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.activeSection == sectionSearch {
		return m.updateSearch(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		m.activeSection = sectionMenu
	}

	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.activeSection = sectionMenu
	case "backspace":
		m.searchQuery = removeLastRune(m.searchQuery)
	default:
		if msg.Type == tea.KeySpace {
			m.searchQuery += " "
		} else if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
		}
	}
	return m, nil
}

func (m Model) renderMenuView() string {
	var builder strings.Builder
	builder.WriteString("Raven " + m.Version + "\n\n")
	builder.WriteString("Main menu\n\n")
	for i, item := range menuItems {
		prefix := "  "
		if i == m.selectedMenuItem {
			prefix = "> "
		}
		builder.WriteString(prefix + item.Label + "\n")
	}
	builder.WriteString("\nUse up/down or j/k to move. Press enter to open. Press q or Ctrl-C to quit.\n")
	return builder.String()
}

func (m Model) renderInstallView() string {
	var builder strings.Builder
	builder.WriteString("Raven " + m.Version + "\n\n")
	builder.WriteString("Install\n\n")
	builder.WriteString("Raven setup is not yet enabled from the main TUI.\n")
	builder.WriteString("Run the setup command in your shell first:\n")
	builder.WriteString("\n  raven setup\n\n")
	builder.WriteString("Setup configures project-local AI integration guidance, including:\n")
	builder.WriteString("- model and tool references in AGENTS.md\n")
	builder.WriteString("- local validation commands\n")
	builder.WriteString("- runtime guidance for supported AI tools\n\n")
	builder.WriteString("Press Esc or Backspace to return to the menu, q/Ctrl-C to quit.\n")
	return builder.String()
}

func (m Model) renderInventoryView() string {
	var builder strings.Builder
	builder.WriteString("Raven " + m.Version + "\n\n")
	builder.WriteString("Inventory\n\n")
	if len(m.Components) == 0 {
		builder.WriteString("No components yet.\n\n")
	} else {
		for _, component := range m.Components {
			builder.WriteString(fmt.Sprintf("- %s — %s\n", component.CIID, component.DisplayName()))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("Press Esc or Backspace to return to the menu, q/Ctrl-C to quit.\n")
	return builder.String()
}

func (m Model) renderRecentMemoriesView() string {
	var builder strings.Builder
	builder.WriteString("Raven " + m.Version + "\n\n")
	builder.WriteString("Recent Memories\n\n")
	recent := latestMemories(m.Events, recentMemoriesLimit)
	if len(recent) == 0 {
		builder.WriteString("No local memories yet.\n\n")
	} else {
		builder.WriteString(fmt.Sprintf("Showing %d most recent local memories.\n\n", len(recent)))
		for i, event := range recent {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, formatMemoryLine(event)))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("Press Esc or Backspace to return to the menu, q/Ctrl-C to quit.\n")
	return builder.String()
}

func (m Model) renderSearchView() string {
	var builder strings.Builder
	builder.WriteString("Raven " + m.Version + "\n\n")
	builder.WriteString("Search\n\n")
	if m.searchQuery == "" {
		builder.WriteString("Type a query to search local memories by CI ID, summary, details, source, type, severity, or status.\n")
	} else {
		builder.WriteString("Search query: " + m.searchQuery + "\n")
	}
	if strings.TrimSpace(m.searchQuery) == "" {
		builder.WriteString("\nNo query yet.\n")
		builder.WriteString("\nPress Esc to return to the menu, Backspace to edit the query, Ctrl-C to quit.\n")
		return builder.String()
	}
	matches := searchMemories(m.Events, m.searchQuery)
	if len(matches) == 0 {
		builder.WriteString("\nNo matches.\n")
	} else {
		totalMatches := len(matches)
		if len(matches) > searchResultLimit {
			matches = matches[:searchResultLimit]
		}
		builder.WriteString(fmt.Sprintf("\nMatches: %d", totalMatches))
		if totalMatches > len(matches) {
			builder.WriteString(fmt.Sprintf(" (showing first %d)", len(matches)))
		}
		builder.WriteString("\n\n")
		for i, event := range matches {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, formatMemoryLine(event)))
		}
	}
	builder.WriteString("\nPress Esc to return to the menu, Backspace to edit the query, Ctrl-C to quit.\n")
	return builder.String()
}

func latestMemories(events []domain.Event, limit int) []domain.Event {
	sorted := append([]domain.Event(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		timeI, timeJ := memoryOrderTime(sorted[i]), memoryOrderTime(sorted[j])
		return timeI.After(timeJ)
	})
	if limit <= 0 || len(sorted) <= limit {
		return sorted
	}
	return sorted[:limit]
}

func searchMemories(events []domain.Event, query string) []domain.Event {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	sorted := latestMemories(events, 0)
	matches := make([]domain.Event, 0, len(sorted))
	for _, event := range sorted {
		if memoryMatches(event, query) {
			matches = append(matches, event)
		}
	}
	return matches
}

func memoryOrderTime(event domain.Event) time.Time {
	if !event.ObservedAt.IsZero() {
		return event.ObservedAt
	}
	return event.IngestedAt
}

func memoryMatches(event domain.Event, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return false
	}
	fields := []string{
		event.CIID,
		event.Summary,
		event.Details,
		event.Source,
		event.Type,
		event.Severity,
		event.Status,
	}
	for _, value := range fields {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func formatMemoryLine(event domain.Event) string {
	status := event.Status
	if status == "" {
		status = "unknown"
	}
	return fmt.Sprintf("%s [%s | %s/%s | %s] %s", event.CIID, event.Source, event.Type, event.Severity, status, event.Summary)
}

func removeLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
}
