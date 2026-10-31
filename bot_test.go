package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"token": "test-token",
		"client_id": "test-client",
		"scripts": {
			"script1": {"cwd": "~", "cmd": ["echo", "hi"]}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Token != "test-token" || cfg.ClientID != "test-client" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	s, ok := cfg.Scripts["script1"]
	if !ok {
		t.Fatal("script1 missing from config")
	}
	if s.Cwd != "~" || len(s.Cmd) != 2 || s.Cmd[0] != "echo" || s.Cmd[1] != "hi" {
		t.Errorf("unexpected script: %+v", s)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"missing file":   "",
		"invalid JSON":   `not json`,
		"missing token":  `{"client_id": "c", "scripts": {}}`,
		"missing client": `{"token": "t", "scripts": {}}`,
		"no scripts":     `{"token": "t", "client_id": "c"}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			var path string
			if content != "" {
				path = filepath.Join(dir, name+".json")
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := loadConfig(path); err == nil {
				t.Fatalf("expected error for %s, got nil", name)
			}
		})
	}
}

func TestRunScriptEcho(t *testing.T) {
	out, err := runScript(Script{Cmd: []string{"echo", "hello world"}})
	if err != nil {
		t.Fatalf("runScript: %v", err)
	}
	if got := strings.TrimSpace(out); got != "hello world" {
		t.Errorf("out = %q, want %q", got, "hello world")
	}
}

func TestRunScriptEmptyStdout(t *testing.T) {
	out, err := runScript(Script{Cmd: []string{"true"}})
	if err != nil {
		t.Fatalf("runScript(true): %v", err)
	}
	if out != "" {
		t.Errorf("out = %q, want empty", out)
	}
}

func TestRunScriptErrorIncludesStatusAndStderr(t *testing.T) {
	out, err := runScript(Script{Cmd: []string{"sh", "-c", "echo boom >&2; exit 3"}})
	if err == nil {
		t.Fatal("expected error for non-zero exit")
	}
	if !strings.Contains(out, "exit status 3") {
		t.Errorf("out missing exit status: %q", out)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("out missing stderr: %q", out)
	}
}

func TestRunScriptRunsInCwd(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker.txt")
	if err := os.WriteFile(marker, []byte("here"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runScript(Script{Cwd: dir, Cmd: []string{"cat", "marker.txt"}})
	if err != nil {
		t.Fatalf("runScript: %v", err)
	}
	if got := strings.TrimSpace(out); got != "here" {
		t.Errorf("out = %q, want %q", got, "here")
	}
}

func TestRunScriptEmptyCmd(t *testing.T) {
	if _, err := runScript(Script{Cmd: nil}); err == nil {
		t.Fatal("expected error for empty cmd")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	cases := map[string]string{
		"":             "",
		"~":            home,
		"~/foo":        filepath.Join(home, "foo"),
		"/abs/path":    "/abs/path",
		"relative/dir": "relative/dir",
	}
	for in, want := range cases {
		if got := expandHome(in); got != want {
			t.Errorf("expandHome(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatOutputWrapsInFence(t *testing.T) {
	got := formatOutput("hello\n")
	want := "```\nhello\n```"
	if got != want {
		t.Errorf("formatOutput = %q, want %q", got, want)
	}
}

func TestFormatOutputEmpty(t *testing.T) {
	got := formatOutput("   \n  \t")
	want := "```\n(no output)\n```"
	if got != want {
		t.Errorf("formatOutput = %q, want %q", got, want)
	}
}

func TestFormatOutputTruncates(t *testing.T) {
	got := formatOutput(strings.Repeat("a", maxMessageLen*2))
	if n := utf8.RuneCountInString(got); n > maxMessageLen {
		t.Errorf("truncated output has %d runes, exceeds limit %d", n, maxMessageLen)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected truncation ellipsis, got tail %q", got[len(got)-2:])
	}
}
