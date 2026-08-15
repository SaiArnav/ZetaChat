# ZetaChat

Terminal-first messaging client — Telegram from the terminal.

[![CI](https://github.com/SaiArnav/ZetaChat/actions/workflows/ci.yml/badge.svg)](https://github.com/SaiArnav/ZetaChat/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/github/license/SaiArnav/ZetaChat)](LICENSE)
[![Stars](https://img.shields.io/github/stars/SaiArnav/ZetaChat)](https://github.com/SaiArnav/ZetaChat/stargazers)
[![Forks](https://img.shields.io/github/forks/SaiArnav/ZetaChat)](https://github.com/SaiArnav/ZetaChat/forks)
[![Issues](https://img.shields.io/github/issues/SaiArnav/ZetaChat)](https://github.com/SaiArnav/ZetaChat/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr-closed/SaiArnav/ZetaChat)](https://github.com/SaiArnav/ZetaChat/pulls)
[![Top language](https://img.shields.io/github/languages/top/SaiArnav/ZetaChat)](https://github.com/SaiArnav/ZetaChat)
[![Commit activity](https://img.shields.io/github/commit-activity/m/SaiArnav/ZetaChat)](https://github.com/SaiArnav/ZetaChat/commits)
[![Go Report Card](https://goreportcard.com/badge/github.com/SaiArnav/ZetaChat)](https://goreportcard.com/report/github.com/SaiArnav/ZetaChat)

A neon-styled Bubble Tea TUI over the MTProto protocol. Browse chats, read and
send messages, and search Telegram globally — all from your terminal.

## Features

- **TUI** — sidebar chat list with unread badges, scrollable message history,
  composer, `/` global search, live incoming messages, neon lime/cyan theme.
- **CLI** — `chats`, `open`, `search`, `send`, `whoami` with `--json` output
  for scripting.
- **Caching** — SQLite store keeps chats and messages locally, so the TUI
  paints instantly and can show history offline.
- **Interactive auth** — phone → code → 2FA password → terms, prompted
  in-app (TUI) or on stdin (CLI).
- **Plugin-ready** — the `core.Messenger` interface means new platforms
  (Discord, etc.) drop in as new adapters.

## Setup

1. Create `.env` with your Telegram API credentials from
   [my.telegram.org](https://my.telegram.org/apps):

   ```
   TELEGRAM_API_ID=123456
   TELEGRAM_API_HASH=your_hash_here
   ```

   (Legacy `APP_ID` / `APP_HASH` keys are also accepted.)

2. Build:

   ```bash
   go build -o zetachat ./cmd/zetachat
   ```

3. Run:

   ```bash
   ./zetachat            # TUI
   ./zetachat chats      # CLI
   ```

The session file is stored at `~/.zetachat/sessions/telegram.json` and the
cache at `~/.zetachat/cache.db`.

## Usage

### TUI keys

| Key              | Action                          |
| ---------------- | ------------------------------- |
| `j`/`k`, `↑`/`↓` | navigate chats                  |
| `enter`          | open chat                       |
| `tab`            | focus composer / back to list   |
| `enter` (composer)| send message                   |
| `/`              | search Telegram                 |
| `r`              | refresh chats                   |
| `q` / `ctrl+c`   | quit                            |

### CLI

```bash
zetachat chats                # list chats
zetachat chats --json         # machine-readable
zetachat open @username       # recent messages
zetachat search "golang"      # global search
zetachat send @arnav "hey!"   # sends after a y/N confirm
zetachat send u123 "hi" --yes # skip the confirm for scripts
zetachat whoami               # logged-in user
```

Chat IDs: `u<id>` private chats, `g<id>` basic groups, `s<id>`
supergroups/channels, or `@username`.

## Architecture

```
cmd/zetachat/   entry point: TUI + CLI dispatch
tui/            Bubble Tea model, views, theme, keys
adapters/       platform adapters (telegram/ via gotd/td)
core/           platform-neutral types (Message, Chat, Messenger)
config/         .env loading + paths
storage/        SQLite cache
```

## License

[MIT](LICENSE)
