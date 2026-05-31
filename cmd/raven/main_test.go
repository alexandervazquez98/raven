package main

import "testing"

func TestSelectRunMode(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want runMode
	}{
		{name: "no args opens dashboard", args: nil, want: runModeDashboard},
		{name: "setup selects setup flow", args: []string{"setup"}, want: runModeSetup},
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
