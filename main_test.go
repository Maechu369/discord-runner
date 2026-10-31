package main

import (
	"os"
	"strings"
	"testing"
)

// withArgs temporarily replaces os.Args for the duration of fn.
func withArgs(t *testing.T, args []string, fn func()) {
	t.Helper()
	old := os.Args
	os.Args = args
	defer func() { os.Args = old }()
	fn()
}

func TestConfigArg(t *testing.T) {
	t.Run("returns the config path after the subcommand", func(t *testing.T) {
		withArgs(t, []string{"discord-runner", "flatten", "cfg.json"}, func() {
			got, err := configArg("flatten")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "cfg.json" {
				t.Errorf("configArg(flatten) = %q, want cfg.json", got)
			}
		})
	})

	t.Run("returns the config path for the bot subcommand", func(t *testing.T) {
		withArgs(t, []string{"discord-runner", "bot", "c.json"}, func() {
			got, err := configArg("bot")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != "c.json" {
				t.Errorf("configArg(bot) = %q, want c.json", got)
			}
		})
	})

	t.Run("errors when the config path is missing", func(t *testing.T) {
		withArgs(t, []string{"discord-runner", "flatten"}, func() {
			_, err := configArg("flatten")
			if err == nil {
				t.Fatal("expected error for missing config path, got nil")
			}
			if !strings.Contains(err.Error(), "flatten") {
				t.Errorf("error %q should name the subcommand", err)
			}
		})
	})
}
