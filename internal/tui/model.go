package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"raven/internal/domain"
)

type Model struct {
	Version    string
	Width      int
	Height     int
	Components []domain.Component
}

func New(version string, components ...[]domain.Component) Model {
	model := Model{Version: version}
	if len(components) > 0 {
		model.Components = append([]domain.Component(nil), components[0]...)
	}
	return model
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var builder strings.Builder
	builder.WriteString("Raven " + m.Version + "\n\n")
	builder.WriteString("Hardware support workspace\n\n")
	builder.WriteString("Components\n")
	if len(m.Components) == 0 {
		builder.WriteString("No components yet.\n")
	} else {
		for _, component := range m.Components {
			builder.WriteString("- " + component.DisplayName() + "\n")
		}
	}
	builder.WriteString("\nPress q to quit.\n")
	return builder.String()
}
