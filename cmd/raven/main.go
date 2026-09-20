package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

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
	runModeSetupHelp
)

func selectRunMode(args []string) runMode {
	if len(args) == 0 {
		return runModeDashboard
	}

	switch args[0] {
	case "version", "--version", "-v":
		return runModeVersion
	case "setup":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h" || args[1] == "help") {
			return runModeSetupHelp
		}
		return runModeSetup
	default:
		return runModeCLI
	}
}

func main() {
	rawArgs := os.Args[1:]
	flagValue, args := parseGlobalFlags(rawArgs)
	configDir, err := resolveDataDir(flagValue)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve config directory: %v\n", err)
		os.Exit(1)
	}
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
	case runModeSetupHelp:
		fmt.Fprint(os.Stdout, setupUsage())
		return
	}

	if mode == runModeCLI {
		if err := cli.Run(args, configDir, os.Stdout, os.Stderr); err != nil {
			os.Exit(1)
		}
		return
	}

	programModel, err := buildDashboardModel(configDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	program := tea.NewProgram(programModel, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// parseGlobalFlags extracts --data-dir from args and returns the residual.
// Supports both "--data-dir <value>" and "--data-dir=<value>" forms.
// If --data-dir is the last token with no value, returns ("", residual).
func parseGlobalFlags(args []string) (dataDir string, residual []string) {
	residual = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--data-dir":
			if i+1 < len(args) {
				dataDir = args[i+1]
				i++ // consume the value
			}
			// If --data-dir is the last token with no value, fall through
			// and let resolveDataDir treat it as empty.
			continue
		case strings.HasPrefix(a, "--data-dir="):
			dataDir = strings.TrimPrefix(a, "--data-dir=")
			continue
		}
		residual = append(residual, a)
	}
	return dataDir, residual
}

// resolveDataDir applies precedence: flag (if non-empty after trim) > env (if non-empty after trim) > os.UserConfigDir() default.
func resolveDataDir(flagValue string) (string, error) {
	flagValue = strings.TrimSpace(flagValue)
	if flagValue != "" {
		return flagValue, nil
	}
	envValue := strings.TrimSpace(os.Getenv("RAVEN_DATA_DIR"))
	if envValue != "" {
		return envValue, nil
	}
	return os.UserConfigDir()
}

func buildDashboardModel(configDir string) (tui.Model, error) {
	components, err := storage.LoadComponents(app.ComponentsPath(configDir))
	if err != nil {
		return tui.Model{}, fmt.Errorf("load components: %w", err)
	}
	events, err := storage.LoadEvents(app.EventsPath(configDir))
	if err != nil {
		return tui.Model{}, fmt.Errorf("load events: %w", err)
	}

	return tui.NewWithEvents(version.String(), components, events), nil
}

func setupUsage() string {
	return `Usage: raven setup [--help]

Launch the Raven AI integration setup wizard.

The setup wizard detects supported AI tooling, shows a reviewable plan, asks before writing files, requires separate approval for user-global writes, and validates generated artifacts where practical.
`
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
