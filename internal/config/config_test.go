package config

import (
	"bytes"
	"encoding/json"
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
		"token":             "abc123",
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
	cases := map[string]string{
		"missing file":     "",
		"invalid JSON":     `not json`,
		"missing token":    `{"client_id": "c", "scripts": {}}`,
		"missing client":   `{"token": "t", "scripts": {}}`,
		"no scripts":       `{"token": "t", "client_id": "c"}`,
		"negative timeout": `{"token": "t", "client_id": "c", "scripts": {"s": {"cmd": ["echo"], "timeout": -1}}}`,
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
