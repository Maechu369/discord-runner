package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// templateJSON is the seed config written by `init`. Embedding the file
// keeps the template and the binary in sync (single source of truth).
//
//go:embed templates/config.template.json
var templateJSON []byte

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		path := "config.json"
		if len(os.Args) >= 3 {
			path = os.Args[2]
		}
		if err := cmdInit(path); err != nil {
			fmt.Fprintf(os.Stderr, "init: %v\n", err)
			os.Exit(1)
		}
	default:
		if err := runFlatten(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}

const usage = `discord-runner - read a JSON config and print key=value

usage:
  discord-runner <config.json>   read a JSON config and print key=value
  discord-runner init [path]     generate an initial config (default: config.json)
`

// cmdInit writes the embedded template config to path. It refuses to
// overwrite an existing file to avoid clobbering a real config.
func cmdInit(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("refusing to overwrite existing file: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.WriteFile(path, templateJSON, 0o644); err != nil {
		return err
	}
	fmt.Printf("created %s\n", path)
	return nil
}

// runFlatten reads, parses, and prints a JSON config as key=value lines.
func runFlatten(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config %q: %v", path, err)
	}

	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("invalid JSON in %q: %v", path, err)
	}

	out := &bytes.Buffer{}
	flatten("", root, out)

	if _, err := os.Stdout.Write(out.Bytes()); err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}
	return nil
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
