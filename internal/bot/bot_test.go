package bot

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"discord-runner/internal/config"
	"github.com/bwmarrin/discordgo"
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
func TestRunScriptTimesOut(t *testing.T) {
	// A command longer than the timeout must be terminated and reported.
	_, err := runScript(config.Script{Cmd: []string{"sleep", "10"}, Timeout: 1})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %v, want one mentioning timeout", err)
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

func newRESTErr(status int) *discordgo.RESTError {
	return &discordgo.RESTError{
		Response: &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
		},
	}
}

func TestWrapRESTErrNil(t *testing.T) {
	if err := wrapRESTErr("op", nil); err != nil {
		t.Errorf("wrapRESTErr(nil) = %v, want nil", err)
	}
}

func TestWrapRESTErrNonREST(t *testing.T) {
	got := wrapRESTErr("op", fmt.Errorf("boom"))
	if got.Error() != "op: boom" {
		t.Errorf("wrapRESTErr(non-REST) = %q, want %q", got, "op: boom")
	}
}

func TestWrapRESTErrUnauthorized(t *testing.T) {
	got := wrapRESTErr("op", newRESTErr(http.StatusUnauthorized))
	for _, sub := range []string{"unauthorized", "HTTP 401", "bot token", "Original error"} {
		if !strings.Contains(got.Error(), sub) {
			t.Errorf("wrapRESTErr(401) message %q missing %q", got, sub)
		}
	}
}

func TestWrapRESTErrForbidden(t *testing.T) {
	got := wrapRESTErr("op", newRESTErr(http.StatusForbidden))
	for _, sub := range []string{"forbidden", "HTTP 403", "Original error"} {
		if !strings.Contains(got.Error(), sub) {
			t.Errorf("wrapRESTErr(403) message %q missing %q", got, sub)
		}
	}
}

func TestWrapRESTErrOtherStatus(t *testing.T) {
	got := wrapRESTErr("op", newRESTErr(http.StatusInternalServerError))
	msg := got.Error()
	if strings.Contains(msg, "HTTP 401") || strings.Contains(msg, "HTTP 403") {
		t.Errorf("wrapRESTErr(500) should not use auth hint: %q", msg)
	}
	if !strings.HasPrefix(msg, "op:") {
		t.Errorf("wrapRESTErr(500) = %q, want prefix %q", msg, "op:")
	}
}

func TestPrefixBotToken(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"bare", "abc.def.ghi", "Bot abc.def.ghi"},
		{"already prefixed", "Bot abc.def.ghi", "Bot abc.def.ghi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prefixBotToken(tt.in); got != tt.want {
				t.Errorf("prefixBotToken(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestCommandData guards the type used to read ApplicationCommandInteractionData.
// discordgo stores the value type in InteractionCreate.Data; a pointer assertion
// (the bug behind "Unsupported interaction") would reject every command.
func TestCommandData(t *testing.T) {
	name := "deploy"
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
		},
	}
	i.Data = discordgo.ApplicationCommandInteractionData{Name: name}
	got, ok := commandData(i)
	if !ok {
		t.Fatalf("commandData: ok = false, want true for a real application command")
	}
	if got.Name != name {
		t.Errorf("commandData: Name = %q, want %q", got.Name, name)
	}

	if _, ok := commandData(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{}}); ok {
		t.Error("commandData: ok = true, want false for nil data")
	}
}

func TestScriptDescription(t *testing.T) {
	cases := map[string]struct {
		name        string
		description string
		want        string
	}{
		"provided": {name: "ping", description: "Ping the bot", want: "Ping the bot"},
		"empty":    {name: "ping", description: "", want: "Run the \"ping\" script"},
	}
	for name, tc := range cases {
		if got := scriptDescription(tc.name, tc.description); got != tc.want {
			t.Errorf("%s: scriptDescription(%q, %q) = %q, want %q", name, tc.name, tc.description, got, tc.want)
		}
	}
}
