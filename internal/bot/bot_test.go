package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"discord-runner/internal/config"
)

func TestRunScriptEcho(t *testing.T) {
	out, err := runScript(config.Script{Cmd: []string{"echo", "hello world"}})
	if err != nil {
		t.Fatalf("runScript: %v", err)
	}
	if got := strings.TrimSpace(out); got != "hello world" {
		t.Errorf("out = %q, want %q", got, "hello world")
	}
}

func TestRunScriptEmptyStdout(t *testing.T) {
	out, err := runScript(config.Script{Cmd: []string{"true"}})
	if err != nil {
		t.Fatalf("runScript(true): %v", err)
	}
	if out != "" {
		t.Errorf("out = %q, want empty", out)
	}
}

func TestRunScriptErrorIncludesStatusAndStderr(t *testing.T) {
	out, err := runScript(config.Script{Cmd: []string{"sh", "-c", "echo boom >&2; exit 3"}})
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

	out, err := runScript(config.Script{Cwd: dir, Cmd: []string{"cat", "marker.txt"}})
	if err != nil {
		t.Fatalf("runScript: %v", err)
	}
	if got := strings.TrimSpace(out); got != "here" {
		t.Errorf("out = %q, want %q", got, "here")
	}
}

func TestRunScriptEmptyCmd(t *testing.T) {
	if _, err := runScript(config.Script{Cmd: nil}); err == nil {
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
