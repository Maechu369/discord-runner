package main

import (
	"bytes"
	"encoding/json"
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
		"token":   "abc123",
		"bot":     map[string]any{"name": "Runner", "presence": true},
		"channels": []any{"111", "222", "333"},
		"limits": map[string]any{
			"maxRetries": 5,
			"timeout":    1.5,
			"metadata":   nil,
		},
	}

	want := map[string]string{
		"token":          "abc123",
		"bot.name":       "Runner",
		"bot.presence":   "true",
		"channels[0]":    "111",
		"channels[1]":    "222",
		"channels[2]":    "333",
		"limits.maxRetries": "5",
		"limits.timeout":   "1.5",
		"limits.metadata":  "null",
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
