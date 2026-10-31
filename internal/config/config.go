package config

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// templateJSON is the seed config written by `Init`. Embedding the file
// keeps the template and the binary in sync (single source of truth).
//
//go:embed templates/config.template.json
var templateJSON []byte

// Script defines a single local command exposed as a Discord slash
// command. Cwd is the working directory, resolved against the home
// directory when it starts with "~"; Cmd is [executable, args...].
// Timeout is the maximum runtime in seconds before the command is
// terminated; a non-positive value disables the limit. Description is an
// optional human-readable slash command description; when empty the bot
// falls back to a generated description.
type Script struct {
	Cwd         string   `json:"cwd"`
	Cmd         []string `json:"cmd"`
	Timeout     int      `json:"timeout"`
	Description string   `json:"description"`
}

// Config is the on-disk bot configuration.
type Config struct {
	TokenFile string            `json:"token_file"`
	Token     string            `json:"-"` // populated by LoadConfig from TokenFile; never read from JSON
	ClientID  string            `json:"client_id"`
	GuildID   string            `json:"guild_id"`
	Scripts   map[string]Script `json:"scripts"`
}

// LoadConfig reads and validates the bot configuration.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config %q: %v", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON in %q: %v", path, err)
	}
	if cfg.TokenFile == "" {
		return nil, fmt.Errorf("config %q: missing token_file", path)
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("config %q: missing client_id", path)
	}
	if len(cfg.Scripts) == 0 {
		return nil, fmt.Errorf("config %q: no scripts defined", path)
	}
	for name, s := range cfg.Scripts {
		if s.Timeout < 0 {
			return nil, fmt.Errorf("config %q: script %q timeout must not be negative", path, name)
		}
	}

	token, err := loadToken(cfg.TokenFile)
	if err != nil {
		return nil, err
	}
	cfg.Token = token
	return &cfg, nil
}

// CommandNames returns the configured slash command names in sorted order.
// Sorting makes the output deterministic for tests and CLI previews.
func (c *Config) CommandNames() []string {
	names := make([]string, 0, len(c.Scripts))
	for name := range c.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// loadToken reads the bot token from the file named by p. A leading "~" is
// expanded to the home directory and surrounding whitespace is trimmed so a
// file written with `echo` still yields a clean token.
func loadToken(p string) (string, error) {
	p = expandHome(p)
	data, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("failed to read token file %q: %v", p, err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file %q is empty", p)
	}
	return token, nil
}

// expandHome resolves a leading "~" in p to the user's home directory so
// that "~/.discord-runner/token" style paths work. It returns p unchanged
// when the home directory cannot be determined.
func expandHome(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	return filepath.Join(home, p[1:])
}

// Init writes the embedded template config to path. It refuses to
// overwrite an existing file to avoid clobbering a real config.
func Init(path string) error {
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

// Flatten reads, parses, and prints a JSON config as key=value lines.
func Flatten(path string) error {
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

// ListOutput returns the formatted list of slash commands defined by the config,
// with names in sorted order and the command each one runs. It is the single
// source of truth shared by the "list" CLI command and the built-in /list slash
// command.
func (c *Config) ListOutput() string {
	names := c.CommandNames()
	out := &bytes.Buffer{}
	fmt.Fprintf(out, "%d command(s):\n", len(names))
	for _, name := range names {
		fmt.Fprintf(out, "  /%s  %s\n", name, listContent(c.Scripts[name]))
	}
	return out.String()
}

// listContent returns what the list output shows for a script: its
// description when set, otherwise the joined command so the reader still
// sees what the command runs.
func listContent(s Script) string {
	if s.Description != "" {
		return s.Description
	}
	return strings.Join(s.Cmd, " ")
}

// List reads, validates, and prints the slash commands defined by the config at
// path. It intentionally does not connect to Discord: it reports what the
// config would register, so it works offline and before a token is valid.
func List(path string) error {
	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}

	if _, err := os.Stdout.Write([]byte(cfg.ListOutput())); err != nil {
		return fmt.Errorf("failed to write output: %v", err)
	}
	return nil
}

// flatten walks a decoded JSON value and emits key=value lines.
// Nested objects are flattened with a "." separator and keys are
// emitted in sorted order for deterministic output; arrays are
// indexed with "[i]".
// A "token" key is redacted: its secret is hidden and
// only its length is shown so it can be verified without leaking it.
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
			if k == "token" {
				fmt.Fprintf(out, "%s=%s\n", child, maskToken(fmt.Sprintf("%v", t[k])))
			} else {
				flatten(child, t[k], out)
			}
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

// maskToken hides a token's secret characters while preserving its length so
// callers can still verify the value's presence. An optional "Bot "/"Bearer "
// scheme prefix is kept visible; the remainder is replaced with [REDACTED] and
// the token's total length is reported.
func maskToken(token string) string {
	prefix, rest := tokenScheme(token)
	if rest == "" {
		// Only a scheme prefix was supplied; nothing sensitive to hide.
		return prefix
	}
	return fmt.Sprintf("%s[REDACTED] (%d chars)", prefix, len(token))
}

// tokenScheme splits an optional leading "Bot "/"Bearer " scheme prefix from
// the rest of a token. It returns an empty prefix when the token has none.
func tokenScheme(token string) (prefix, rest string) {
	for _, p := range []string{"Bot ", "Bearer "} {
		if strings.HasPrefix(token, p) {
			return p, token[len(p):]
		}
	}
	return "", token
}
