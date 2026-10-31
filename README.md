# discord-runner

🌐 [日本語](docs/README.ja.md)

A Discord bot that runs **local commands** from slash commands.
Each script defined in the config file registers its own slash command, letting you run commands like `ls` or `git status` from Discord and receive the output as a message.

## Features

- Per-slash-command working directory, arguments, and timeout
- `~` expands to the home directory
- Returns command output as a fenced code block (truncated automatically beyond 2000 characters)
- Kills the command on timeout; shows exit status and stderr on failure
- `config init` generates a config from a template; `config flatten` prints the config as `key=value` lines

## Requirements

- Go 1.26 or later
- A Discord application (bot token and client ID)
- Nix (recommended for the dev shell, optional)

## Installation

```sh
go build -o discord-runner .
```

Or build from the Nix dev shell.

```sh
nix develop
make build
```

## Usage

```sh
# Start the bot from a config (subcommand optional)
./discord-runner start config.json
./discord-runner config.json          # same as above

# Generate a config (fails if the file already exists)
./discord-runner config init
./discord-runner config init my.json

# Print a config as key=value lines
./discord-runner config flatten config.json
```

`config flatten` never prints the token itself: it shows the `Bot ` prefix and the character count, but masks the secret.

`--help` / `-h` prints the same usage.

## Config file

It uses JSON. It requires a `token_file` pointing at a file that holds the bot
token, a `client_id`, and at least one script.
You can generate one with `config init`; the template embedded in this binary is also available in this repo.

The `token_file` points at a separate file that holds your bot token, so the
token itself never lives in the config. Put the raw token from your Discord app
in that file — the `Bot ` prefix is optional and added for you at startup.
Leading and trailing whitespace, including a trailing newline from `echo`, is
trimmed, so writing just the token is enough:

```sh
echo "MII3.YourTokenHere.example" > token.txt
```

The token is read from that file when the bot starts and is never written back.

```json
{
  "token_file": ">>Path to a file containing your token<<",
  "client_id": ">>Your Client ID<<",
  "scripts": {
    "script1": {
      "cwd": "~",
      "cmd": ["echo", "If you defined like this, you can call this command with `/script1`."],
      "timeout": 60
    }
  }
}
```

| Key | Meaning |
| --- | --- |
| `token_file` | Path to a file containing the bot token ([get it from the official docs](https://docs.discord.com/developers/quick-start/getting-started)) |
| `client_id` | Discord application client ID ([how to find it](https://support-dev.discord.com/hc/en-us/articles/360028717192-Where-can-I-find-my-Application-Team-Server-ID)) |
| `scripts` | Map of slash command name to script |

Each script:

| Field | Meaning |
| --- | --- |
| `cmd` | Command to run: `[executable, args...]` |
| `cwd` | Working directory (`~` expands to your home directory) |
| `timeout` | Max runtime in seconds (0 or less disables it) |

Both values come from your Discord app: the **Application ID** is on the *General Information* page and the bot **Token** on the *Bot* page. The [Getting Started guide](https://docs.discord.com/developers/quick-start/getting-started) walks you through creating an app and copying them.

## Development

Use the Makefile.

| Target | Description |
| --- | --- |
| `make build` | Build the binary |
| `make test` | Run tests |
| `make race` | Run tests with the race detector |
| `make check` | `fmtcheck` + `vet` + `test` (default `all`) |
| `make run` | Start the bot from `config.json` |
| `make init` | Generate the config file |
| `make flatten` | Print the config as `key=value` |
| `make fmt` | Format Go sources with gofmt |
| `make clean` | Remove the binary |

Go 1.26 or later is required. The dev shell is provided via the Nix flake and enables `gofmt`, `nixfmt`, and `git-secrets` pre-commit hooks.

```sh
nix develop   # dev shell with hooks
```

## Project structure

```
.
├── main.go                          # CLI entrypoint
├── internal/
│   ├── config/                      # Config load, init, and flatten
│   └── bot/                         # Discord bot behavior
└── Makefile
```
