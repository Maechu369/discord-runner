package main

import (
	"fmt"
	"os"

	"discord-runner/internal/bot"
	"discord-runner/internal/config"
)

// version is the build version, injected at compile time via
//
//	-ldflags "-X main.version=<version>"
//
// It falls back to "dev" when no value is supplied (e.g. `go run`).
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "--help", "-h":
		// Show usage. Same text as running without arguments.
		fmt.Fprint(os.Stdout, usage)
		os.Exit(0)
	case "start":
		// Run the Discord bot from the given config.
		if err := bot.RunBot(arg("start", 2)); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	case "config":
		// `config` groups the config helpers: `config init` and
		// `config flatten <config.json>`.
		if len(os.Args) < 3 {
			fmt.Fprint(os.Stderr, configUsage)
			os.Exit(2)
		}
		switch os.Args[2] {
		case "--help", "-h":
			fmt.Fprint(os.Stdout, configUsage)
			os.Exit(0)
		case "init":
			path := "config.json"
			if len(os.Args) >= 4 {
				path = os.Args[3]
			}
			if err := config.Init(path); err != nil {
				fmt.Fprintf(os.Stderr, "config init: %v\n", err)
				os.Exit(1)
			}
		case "flatten":
			if err := config.Flatten(arg("flatten", 3)); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprint(os.Stderr, configUsage)
			os.Exit(2)
		}
	case "list":
		// List the slash commands defined by the config without connecting.
		if err := config.List(arg("list", 2)); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	case "version":
		// Print the build version. No config or Discord connection needed.
		fmt.Println(version)
		os.Exit(0)
	default:
		// Default entrypoint: run the Discord bot from the given config.
		if err := bot.RunBot(arg("start", 2)); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}

// configArg returns os.Args[idx], the config path that follows the
// subcommand(s). name is used only in the usage message when the path is
// absent (idx is out of range).
func configArg(name string, idx int) (string, error) {
	if idx >= len(os.Args) {
		return "", fmt.Errorf("usage: discord-runner %s <config.json>\n", name)
	}
	return os.Args[idx], nil
}

// arg returns the config path for a subcommand, exiting with usage when
// it is missing. idx is the index of the path in os.Args.
func arg(name string, idx int) string {
	p, err := configArg(name, idx)
	if err != nil {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	return p
}

const usage = `discord-runner - a Discord bot that runs local commands

usage:
  discord-runner [start] <config.json>
      Run the Discord bot from the given config.
      When no subcommand is given, "start" is assumed.

  discord-runner config init [path]
      Generate an initial config from the embedded template.
      Default path is config.json.

  discord-runner config flatten <config.json>
      Read a config file and print it as key=value lines.

  discord-runner list <config.json>
      List the slash commands defined in the config without connecting.

  discord-runner version
      Print the build version.
`

const configUsage = `config commands:

  config init [path]
      Generate an initial config from the embedded template.
      Default path is config.json.

  config flatten <config.json>
      Read a config file and print it as key=value lines.

config file:

  token_file  Path to a file containing the bot token. The token is read
              from that file at startup, never from the config itself.
  client_id   Discord client/application ID
  scripts     Map of slash command name to script. Each script has:
                cmd      Command to run: [executable, args...]
                cwd      Working directory (~ expands to your home directory)
                timeout  Maximum runtime in seconds (0 disables the limit)
`
