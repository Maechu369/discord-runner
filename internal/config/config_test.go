package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// parseLines runs flatten and returns the parsed key=value pairs.
func parseLines(t *testing.T, v any) map[string]string {
	t.Helper()
	out := &bytes.Buffer{}
	flatten("", v, out)
	got := map[string]string{}
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		k, val, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("malformed output line: %q", line)
		}
		got[k] = val
	}
	return got
}

func TestFlattenBasics(t *testing.T) {
	input := map[string]any{
		"token":    "abc123",
		"bot":      map[string]any{"name": "Runner", "presence": true},
		"channels": []any{"111", "222", "333"},
		"limits": map[string]any{
			"maxRetries": 5,
			"timeout":    1.5,
			"metadata":   nil,
		},
	}

	want := map[string]string{
		"token":             "[REDACTED] (6 chars)",
		"bot.name":          "Runner",
		"bot.presence":      "true",
		"channels[0]":       "111",
		"channels[1]":       "222",
		"channels[2]":       "333",
		"limits.maxRetries": "5",
		"limits.timeout":    "1.5",
		"limits.metadata":   "null",
	}

	got := parseLines(t, input)
	for k, v := range want {
		if got[k] != v {
			t.Errorf("key %q = %q, want %q", k, got[k], v)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("unexpected key %q", k)
		}
	}
}

func TestFlattenTokenMasked(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"bare", "abcdef012345.ghijklmnop.stuvwxyz0123456789_abc", maskToken("abcdef012345.ghijklmnop.stuvwxyz0123456789_abc")},
		{"bot prefix", "Bot abcdef012345.ghijklmnop.stuvwxyz0123456789_abc", maskToken("Bot abcdef012345.ghijklmnop.stuvwxyz0123456789_abc")},
		{"bearer prefix", "Bearer xyz.Bearer-prefix-token-value-123456", maskToken("Bearer xyz.Bearer-prefix-token-value-123456")},
		{"empty", "", maskToken("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			flatten("", map[string]any{"token": tc.in}, out)
			got := out.String()
			if !strings.Contains(got, tc.want) {
				t.Errorf("flatten(%q) = %q, want to contain %q", tc.in, got, tc.want)
			}
			if tc.in != "" && strings.Contains(got, tc.in) {
				t.Errorf("flatten leaked raw token in %q", got)
			}
		})
	}
}

func TestFlattenEmptyAndNested(t *testing.T) {
	// Empty object and empty array must not emit any lines.
	if got := parseLines(t, map[string]any{}); len(got) != 0 {
		t.Errorf("empty object = %v, want empty", got)
	}
	if got := parseLines(t, []any{}); len(got) != 0 {
		t.Errorf("empty array = %v, want empty", got)
	}

	// Deeply nested keys chain with dots.
	deep := map[string]any{
		"a": map[string]any{"b": map[string]any{"c": 1}},
	}
	got := parseLines(t, deep)
	if got["a.b.c"] != "1" {
		t.Errorf("deep nesting a.b.c = %q, want 1", got["a.b.c"])
	}
}

func TestFlattenProducesSortedLines(t *testing.T) {
	// Output lines must be in deterministic sorted order.
	input := map[string]any{"z": 1, "a": 2, "m": 3}
	out := &bytes.Buffer{}
	flatten("", input, out)
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if !slices.IsSortedFunc(lines, strings.Compare) {
		t.Errorf("output not sorted: %v", lines)
	}
	if strings.Join(lines, ",") != "a=2,m=3,z=1" {
		t.Errorf("unexpected output: %v", lines)
	}
}

func TestFormatValue(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"string", "hello", "hello"},
		{"string with spaces", "a b", "a b"},
		{"int", 42, "42"},
		{"negative", -7, "-7"},
		{"float", 1.5, "1.5"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "null"},
		{"json number", json.Number("9"), "9"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatValue(c.in); got != c.want {
				t.Errorf("formatValue(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(`{
		"token_file": %q,
		"client_id": "test-client",
		"scripts": {
			"script1": {"cwd": "~", "cmd": ["echo", "hi"]}
		}
	}`, tokenPath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
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
	emptyToken := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(emptyToken, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"missing file":         "",
		"invalid JSON":         `not json`,
		"missing token_file":   `{"client_id": "c", "scripts": {}}`,
		"missing client":       `{"token_file": "token.txt", "scripts": {}}`,
		"no scripts":           `{"token_file": "token.txt", "client_id": "c"}`,
		"negative timeout":     `{"token_file": "token.txt", "client_id": "c", "scripts": {"s": {"cmd": ["echo"], "timeout": -1}}}`,
		"token file not found": `{"token_file": "missing.txt", "client_id": "c", "scripts": {"s": {"cmd": ["echo"]}}}`,
		"empty token file":     fmt.Sprintf(`{"token_file": %q, "client_id": "c", "scripts": {"s": {"cmd": ["echo"]}}}`, emptyToken),
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
			if _, err := LoadConfig(path); err == nil {
				t.Fatalf("expected error for %s, got nil", name)
			}
		})
	}
}

func TestLoadConfigGuildID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(`{
		"token_file": %q,
		"client_id": "test-client",
		"guild_id": "123456789",
		"scripts": {
			"script1": {"cwd": "~", "cmd": ["echo", "hi"]}
		}
	}`, tokenPath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.GuildID != "123456789" {
		t.Errorf("guild_id = %q, want %q", cfg.GuildID, "123456789")
	}
}

func TestLoadConfigDescription(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(`{
		"token_file": %q,
		"client_id": "test-client",
		"scripts": {
			"script1": {"cwd": "~", "cmd": ["echo", "hi"], "description": "Runs a test"}
		}
	}`, tokenPath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got := cfg.Scripts["script1"].Description; got != "Runs a test" {
		t.Errorf("script1 description = %q, want %q", got, "Runs a test")
	}
}

func TestCmdInitWritesTemplate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := Init(path); err != nil {
		t.Fatalf("Init: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if string(got) != string(templateJSON) {
		t.Errorf("generated content =\n%q\nwant template =\n%q", got, templateJSON)
	}

	// The generated file must be valid, parseable JSON.
	var root any
	if err := json.Unmarshal(got, &root); err != nil {
		t.Errorf("generated config is not valid JSON: %v", err)
	}
}

func TestCmdInitRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := Init(path); err == nil {
		t.Fatal("expected error overwriting existing file, got nil")
	}

	got, _ := os.ReadFile(path)
	if string(got) != "original" {
		t.Errorf("existing file was clobbered: %q", got)
	}
}

func TestCommandNames(t *testing.T) {
	cfg := &Config{Scripts: map[string]Script{
		"zeta":  {},
		"alpha": {},
		"mid":   {},
	}}
	want := []string{"alpha", "mid", "zeta"}
	if got := cfg.CommandNames(); !slices.Equal(got, want) {
		t.Errorf("CommandNames = %v, want %v", got, want)
	}
}

func TestCommandNamesEmpty(t *testing.T) {
	if got := (&Config{}).CommandNames(); len(got) != 0 {
		t.Errorf("empty config CommandNames = %v, want none", got)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("test-token"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(`{
		"token_file": %q,
		"client_id": "test-client",
		"scripts": {
			"zeta": {"cmd": ["echo", "z"]},
			"alpha": {"cmd": ["echo", "a"]}
		}
	}`, tokenPath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := captureStdout(t, func() {
		if err := List(path); err != nil {
			t.Fatalf("List: %v", err)
		}
	})
	want := "2 command(s):\n  /alpha  echo a\n  /zeta  echo z\n"
	if got != want {
		t.Errorf("List output =\n%q\nwant\n%q", got, want)
	}
}

func TestListErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")
	if err := List(path); err == nil {
		t.Fatal("expected error listing a missing config, got nil")
	}
}

// TestListOutput covers the renderer shared by the built-in /list slash command.
// It validates sorting and formatting independently of the CLI stdout path.
func TestListOutput(t *testing.T) {
	cfg := &Config{Scripts: map[string]Script{
		"zeta":  {Cmd: []string{"echo", "z"}},
		"alpha": {Cmd: []string{"echo", "a"}},
	}}
	want := "2 command(s):\n  /alpha  echo a\n  /zeta  echo z\n"
	if got := cfg.ListOutput(); got != want {
		t.Errorf("ListOutput =\n%q\nwant\n%q", got, want)
	}
}

// captureStdout redirects os.Stdout during fn and returns everything written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy stdout: %v", err)
	}
	r.Close()
	return buf.String()
}
