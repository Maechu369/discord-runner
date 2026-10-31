# Adding the bot to a server

discord-runner registers each configured script as a slash command. To use them
in a server, create a Discord application and a bot first, then invite the bot to
the server with the slash-command permission enabled.

Most steps are covered by Discord's official documentation; the links below take
you there. This page only shows the one setting you need.

## 1. Create a Discord application and a bot

Create an application, add a bot, then copy the **Application ID** and the **bot
token**.

- Guide: [Getting Started](https://docs.discord.com/developers/quick-start/getting-started)
- Create an app now: [https://discord.com/developers/applications?new_application=true](https://discord.com/developers/applications?new_application=true)

## 2. Invite the bot with the "Use Bot Commands" permission

Slash commands require the **"Use Bot Commands"** permission. Enable it when you
invite the bot — this is the only permission this project needs.

- How the invite flow and permissions work: [OAuth2 and Permissions](https://docs.discord.com/developers/topics/oauth2)

The permission is `USE_APPLICATION_COMMANDS` (`1 << 31`, value
`2147483648`). An invite link that enables it looks like:

```
https://discord.com/oauth2/authorize?client_id=CLIENT_ID&scope=bot%20applications.commands&permissions=2147483648
```

Replace `CLIENT_ID` with your Application ID.

### Easier: the installation page

You can also set the scopes and this permission, then generate the invite link,
on the installation page for your app:

- [https://discord.com/developers/applications/select/<APP_ID>/installation](https://discord.com/developers/applications/select/<APP_ID>/installation)

## Where to put the values

`client_id` and the bot token go in your config file. See the [main
README](../README.md) for the config format.
