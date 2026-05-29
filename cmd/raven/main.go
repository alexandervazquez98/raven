package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"raven/internal/app"
	"raven/internal/cli"
	"raven/internal/storage"
	"raven/internal/tui"
	"raven/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println(version.String())
			return
		}
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve config directory: %v\n", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		if err := cli.Run(os.Args[1:], configDir, os.Stdout, os.Stderr); err != nil {
			os.Exit(1)
		}
		return
	}

	components, err := storage.LoadComponents(app.ComponentsPath(configDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "load components: %v\n", err)
		os.Exit(1)
	}

	program := tea.NewProgram(tui.New(version.String(), components), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
