package main

import (
	"fmt"
	"os"

	"discord-runner/internal/bot"
	"discord-runner/internal/config"
)

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
		if err := config.Init(path); err != nil {
			fmt.Fprintf(os.Stderr, "init: %v\n", err)
			os.Exit(1)
		}
	case "flatten":
		if err := config.Flatten(arg("flatten")); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	case "bot":
		if err := bot.RunBot(arg("bot")); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	default:
		// Default entrypoint: run the Discord bot from the given config.
		if err := bot.RunBot(arg("bot")); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}

// configArg returns the config path that follows the subcommand in
// os.Args (os.Args[2]). name is used only in the usage message when the
// path is absent.
func configArg(name string) (string, error) {
	if len(os.Args) < 3 {
		return "", fmt.Errorf("usage: discord-runner %s <config.json>\n", name)
	}
	return os.Args[2], nil
}

// arg returns the config path for a subcommand, exiting with usage when
// it is missing.
func arg(name string) string {
	p, err := configArg(name)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(2)
	}
	return p
}

const usage = `discord-runner - a Discord bot that runs local commands

usage:
  discord-runner [bot] <config.json>   run the Discord bot (default: bot)
  discord-runner flatten <config.json>   print a JSON config as key=value
  discord-runner init [path]           generate an initial config (default: config.json)
`
