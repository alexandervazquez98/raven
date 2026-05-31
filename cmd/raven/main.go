package main

import (
	"fmt"
	"os"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"raven/internal/app"
	"raven/internal/cli"
	"raven/internal/setup"
	"raven/internal/setuptui"
	"raven/internal/storage"
	"raven/internal/tui"
	"raven/internal/version"
)

type runMode int

const (
	runModeDashboard runMode = iota
	runModeVersion
	runModeCLI
	runModeSetup
)

func selectRunMode(args []string) runMode {
	if len(args) == 0 {
		return runModeDashboard
	}

	switch args[0] {
	case "version", "--version", "-v":
		return runModeVersion
	case "setup":
		return runModeSetup
	default:
		return runModeCLI
	}
}

func main() {
	args := os.Args[1:]
	mode := selectRunMode(args)

	switch mode {
	case runModeVersion:
		fmt.Println(version.String())
		return
	case runModeSetup:
		if err := runSetup(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve config directory: %v\n", err)
		os.Exit(1)
	}
	if mode == runModeCLI {
		if err := cli.Run(args, configDir, os.Stdout, os.Stderr); err != nil {
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

func runSetup() error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve project directory: %w", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	env := setup.SetupEnv{
		ProjectDir: projectDir,
		HomeDir:    homeDir,
		GOOS:       runtime.GOOS,
		Commands:   setup.ExecCommandDetector{},
		FS:         setup.OSFileSystem{},
	}
	program := tea.NewProgram(setuptui.NewForEnv(env), tea.WithAltScreen())
	_, err = program.Run()
	return err
}
