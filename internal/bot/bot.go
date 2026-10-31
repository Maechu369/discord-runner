package bot

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"

	"discord-runner/internal/config"
)

// RunBot connects to Discord, registers a slash command for each script,
// and serves interactions until interrupted by a signal.
func RunBot(configPath string) error {
	cfg, err := config.LoadConfig(configPath)
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
		return wrapRESTErr("failed to register slash commands", err)
	}

	if err := session.Open(); err != nil {
		return wrapRESTErr("failed to connect to Discord", err)
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
func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) {
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
// appended so the failure is visible in Discord. When s.Timeout is
// positive, the command is terminated once that many seconds elapse.
func runScript(s config.Script) (string, error) {
	if len(s.Cmd) == 0 {
		return "", fmt.Errorf("script cmd is empty")
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if s.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(s.Timeout)*time.Second)
		defer cancel()
	}

	c := exec.CommandContext(ctx, s.Cmd[0], s.Cmd[1:]...)
	if dir := expandHome(s.Cwd); dir != "" {
		c.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()

	out := strings.TrimSpace(stdout.String())
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			out = fmt.Sprintf("timed out after %ds", s.Timeout)
			return out, fmt.Errorf("script %q timed out after %ds", s.Cmd[0], s.Timeout)
		}
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
