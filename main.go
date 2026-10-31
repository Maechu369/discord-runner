package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
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
	case "flatten":
		if err := runFlatten(arg("flatten")); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	case "bot":
		if err := runBot(arg("bot")); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	default:
		// Default entrypoint: run the Discord bot from the given config.
		if err := runBot(arg("bot")); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}

// arg returns the positional argument for a subcommand, exiting with
// usage when it is missing.
func arg(name string) string {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: discord-runner %s <config.json>\n", name)
		os.Exit(2)
	}
	return os.Args[1]
}

const usage = `discord-runner - a Discord bot that runs local commands

usage:
  discord-runner [bot] <config.json>   run the Discord bot (default: bot)
  discord-runner flatten <config.json>   print a JSON config as key=value
  discord-runner init [path]           generate an initial config (default: config.json)
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

// Script defines a single local command exposed as a Discord slash
// command. Cwd is the working directory, resolved against the home
// directory when it starts with "~"; Cmd is [executable, args...].
type Script struct {
	Cwd string   `json:"cwd"`
	Cmd []string `json:"cmd"`
}

// Config is the on-disk bot configuration.
type Config struct {
	Token    string            `json:"token"`
	ClientID string            `json:"client_id"`
	Scripts  map[string]Script `json:"scripts"`
}

// loadConfig reads and validates the bot configuration.
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config %q: %v", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON in %q: %v", path, err)
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("config %q: missing token", path)
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("config %q: missing client_id", path)
	}
	if len(cfg.Scripts) == 0 {
		return nil, fmt.Errorf("config %q: no scripts defined", path)
	}
	return &cfg, nil
}

// runBot connects to Discord, registers a slash command for each script,
// and serves interactions until interrupted by a signal.
func runBot(configPath string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	session, err := discordgo.New(cfg.Token)
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	session.AddHandler(func(_ *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			handleSlashCommand(session, i, cfg)
		}
	})

	cmds := make([]*discordgo.ApplicationCommand, 0, len(cfg.Scripts))
	for name := range cfg.Scripts {
		cmds = append(cmds, &discordgo.ApplicationCommand{
			Name:        name,
			Description: fmt.Sprintf("Run the %q script", name),
			Type:        discordgo.ChatApplicationCommand,
		})
	}
	if _, err := session.ApplicationCommandBulkOverwrite(cfg.ClientID, "", cmds); err != nil {
		return fmt.Errorf("failed to register slash commands: %v", err)
	}

	if err := session.Open(); err != nil {
		return fmt.Errorf("failed to connect to Discord: %v", err)
	}

	fmt.Printf("discord-runner is online. %d slash command(s) registered:\n", len(cmds))
	for name := range cfg.Scripts {
		fmt.Printf("  /%s\n", name)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	fmt.Println("\ndiscord-runner shutting down.")
	return nil
}

// handleSlashCommand runs the named script and returns its output to the
// invoking user. It defers the response first so long-running commands
// do not hit Discord's 3-second reply deadline, then edits the deferred
// message with the captured output.
func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *Config) {
	data, ok := i.Data.(*discordgo.ApplicationCommandInteractionData)
	if !ok {
		respondImmediate(s, i, "Unsupported interaction")
		return
	}

	script, ok := cfg.Scripts[data.Name]
	if !ok {
		respondImmediate(s, i, fmt.Sprintf("Unknown script: `%s`", data.Name))
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("defer %q: %v", data.Name, err)
		return
	}

	out, err := runScript(script)
	if err != nil {
		log.Printf("run %q: %v", data.Name, err)
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: new(formatOutput(out)),
	}); err != nil {
		log.Printf("edit %q: %v", data.Name, err)
	}
}

// respondImmediate sends a synchronous reply for short messages that do
// not need to wait for a command to finish.
func respondImmediate(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content},
	}); err != nil {
		log.Printf("respond: %v", err)
	}
}

// runScript executes s.Cmd with its working directory, returning the
// command's stdout. On a non-zero exit, the exit status and stderr are
// appended so the failure is visible in Discord.
func runScript(s Script) (string, error) {
	if len(s.Cmd) == 0 {
		return "", fmt.Errorf("script cmd is empty")
	}

	c := exec.Command(s.Cmd[0], s.Cmd[1:]...)
	if dir := expandHome(s.Cwd); dir != "" {
		c.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()

	out := strings.TrimSpace(stdout.String())
	if err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			if out != "" {
				out += "\n"
			}
			out += fmt.Sprintf("exit status %d\n%s", c.ProcessState.ExitCode(), errStr)
		} else {
			out = fmt.Sprintf("exit status %d", c.ProcessState.ExitCode())
		}
	}
	return out, err
}

// expandHome resolves a leading "~" in p to the user's home directory.
// An empty result signals the default working directory.
func expandHome(p string) string {
	if p == "" {
		return ""
	}
	home := homeDir()
	if p == "~" || strings.HasPrefix(p, "~"+string(os.PathSeparator)) {
		if p == "~" {
			return home
		}
		return filepath.Join(home, p[2:])
	}
	return p
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

const maxMessageLen = 2000

// formatOutput wraps command output in a fenced code block for Discord.
// Output longer than Discord's 2000-character message limit is truncated.
func formatOutput(out string) string {
	out = strings.TrimSpace(out)
	if out == "" {
		out = "(no output)"
	}
	content := "```\n" + out + "\n```"
	if n := utf8.RuneCountInString(content); n > maxMessageLen {
		// Trim to maxMessageLen-1 runes, then append an ellipsis rune so the
		// total stays at the 2000-character Discord limit without splitting a
		// multi-byte character.
		runes := []rune(content)
		content = string(runes[:maxMessageLen-1]) + "…"
	}
	return content
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
