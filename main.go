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
