package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: discord-runner <config.json>")
		os.Exit(2)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %v\n", err)
		os.Exit(1)
	}

	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		fmt.Fprintf(os.Stderr, "invalid JSON: %v\n", err)
		os.Exit(1)
	}
	out := &bytes.Buffer{}
	flatten("", root, out)

	if _, err := os.Stdout.Write(out.Bytes()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

// flatten walks a decoded JSON value and emits key=value lines.
// Nested objects are flattened with a "." separator and keys are
// emitted in sorted order for deterministic output; arrays are
// indexed with "[i]".
func flatten(prefix string, v any, out *bytes.Buffer) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := k
			if prefix != "" {
				child = prefix + "." + k
			}
			flatten(child, t[k], out)
		}
	case []any:
		for i, item := range t {
			child := fmt.Sprintf("%s[%d]", prefix, i)
			flatten(child, item, out)
		}
	default:
		fmt.Fprintf(out, "%s=%s\n", prefix, formatValue(v))
	}
}

// formatValue renders a scalar leaf. Strings are printed unquoted
// so output matches the key=value convention; other JSON types use
// their canonical JSON encoding.
func formatValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return "null"
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}
