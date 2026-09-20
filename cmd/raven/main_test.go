package main

import (
	"os"
	"strings"
	"testing"
)

func TestSelectRunMode(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want runMode
	}{
		{name: "no args opens dashboard", args: nil, want: runModeDashboard},
		{name: "setup selects setup flow", args: []string{"setup"}, want: runModeSetup},
		{name: "setup help selects setup help", args: []string{"setup", "--help"}, want: runModeSetupHelp},
		{name: "setup short help selects setup help", args: []string{"setup", "-h"}, want: runModeSetupHelp},
		{name: "setup help subcommand selects setup help", args: []string{"setup", "help"}, want: runModeSetupHelp},
		{name: "version subcommand stays version", args: []string{"version"}, want: runModeVersion},
		{name: "long version flag stays version", args: []string{"--version"}, want: runModeVersion},
		{name: "short version flag stays version", args: []string{"-v"}, want: runModeVersion},
		{name: "normal CLI command stays CLI", args: []string{"ci", "list"}, want: runModeCLI},
		{name: "unknown command stays CLI", args: []string{"unknown"}, want: runModeCLI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectRunMode(tt.args); got != tt.want {
				t.Fatalf("selectRunMode(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildDashboardModelLoadsMissingEventsAsEmpty(t *testing.T) {
	model, err := buildDashboardModel(t.TempDir())
	if err != nil {
		t.Fatalf("buildDashboardModel() error = %v, want nil", err)
	}
	view := model.View()
	for _, want := range []string{"Main menu", "Install", "Recent Memories", "Search", "Exit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dashboard view = %q, want %q", view, want)
		}
	}
}

func TestParseGlobalFlags(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		wantFlag         string
		wantResidualLen  int
		wantResidualTail []string
	}{
		{name: "empty args", args: nil, wantFlag: "", wantResidualLen: 0},
		{name: "no flag", args: []string{"ci", "list"}, wantFlag: "", wantResidualLen: 2},
		{name: "space form at start", args: []string{"--data-dir", "/tmp/foo", "ci", "list"}, wantFlag: "/tmp/foo", wantResidualLen: 2, wantResidualTail: []string{"ci", "list"}},
		{name: "equals form at start", args: []string{"--data-dir=/tmp/foo", "ci", "list"}, wantFlag: "/tmp/foo", wantResidualLen: 2, wantResidualTail: []string{"ci", "list"}},
		{name: "flag in middle", args: []string{"ci", "--data-dir", "/tmp/foo", "list"}, wantFlag: "/tmp/foo", wantResidualLen: 2, wantResidualTail: []string{"ci", "list"}},
		{name: "flag at end with value", args: []string{"ci", "list", "--data-dir", "/tmp/foo"}, wantFlag: "/tmp/foo", wantResidualLen: 2, wantResidualTail: []string{"ci", "list"}},
		{name: "flag at end without value", args: []string{"ci", "list", "--data-dir"}, wantFlag: "", wantResidualLen: 2, wantResidualTail: []string{"ci", "list"}},
		{name: "only flag and value", args: []string{"--data-dir", "/tmp/foo"}, wantFlag: "/tmp/foo", wantResidualLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFlag, gotResidual := parseGlobalFlags(tt.args)
			if gotFlag != tt.wantFlag {
				t.Errorf("flag = %q, want %q", gotFlag, tt.wantFlag)
			}
			if len(gotResidual) != tt.wantResidualLen {
				t.Errorf("residual len = %d, want %d (got %v)", len(gotResidual), tt.wantResidualLen, gotResidual)
			}
			if tt.wantResidualTail != nil {
				if len(gotResidual) >= len(tt.wantResidualTail) {
					tail := gotResidual[len(gotResidual)-len(tt.wantResidualTail):]
					for i := range tt.wantResidualTail {
						if tail[i] != tt.wantResidualTail[i] {
							t.Errorf("residual tail[%d] = %q, want %q", i, tail[i], tt.wantResidualTail[i])
						}
					}
				}
			}
		})
	}
}

func TestResolveDataDirPrecedence(t *testing.T) {
	// Use t.Setenv to control RAVEN_DATA_DIR cleanly across subtests.
	tests := []struct {
		name     string
		flag     string
		envValue string
		want     string // exact expected, or "USERCONFIGDIR" sentinel for os.UserConfigDir()
	}{
		{name: "neither set falls back to UserConfigDir", flag: "", envValue: "", want: "USERCONFIGDIR"},
		{name: "flag only", flag: "/flag/path", envValue: "", want: "/flag/path"},
		{name: "env only", flag: "", envValue: "/env/path", want: "/env/path"},
		{name: "flag wins over env", flag: "/flag/path", envValue: "/env/path", want: "/flag/path"},
		{name: "whitespace flag falls back to env", flag: "   ", envValue: "/env/path", want: "/env/path"},
		{name: "whitespace-only env falls back to UserConfigDir", flag: "", envValue: "   ", want: "USERCONFIGDIR"},
		{name: "flag with surrounding whitespace trimmed", flag: "  /flag/path  ", envValue: "", want: "/flag/path"},
		{name: "empty flag string treated as unset", flag: "", envValue: "/env/path", want: "/env/path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RAVEN_DATA_DIR", tt.envValue)
			got, err := resolveDataDir(tt.flag)
			if err != nil {
				t.Fatalf("resolveDataDir(%q) error = %v", tt.flag, err)
			}
			want := tt.want
			if want == "USERCONFIGDIR" {
				defaultDir, derr := os.UserConfigDir()
				if derr != nil {
					t.Fatalf("os.UserConfigDir() error = %v", derr)
				}
				want = defaultDir
			}
			if got != want {
				t.Errorf("resolveDataDir(%q) with RAVEN_DATA_DIR=%q = %q, want %q", tt.flag, tt.envValue, got, want)
			}
		})
	}
}
