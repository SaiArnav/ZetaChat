# ZetaChat — Project Spec

A terminal-first, plugin-based universal messaging client. One TUI. Multiple platforms
(Telegram, WhatsApp, Discord, later Signal/Matrix/Slack) unified behind a common API,
with optional AI-agent access for search, drafting, and (confirmed) sending.

## 1. Goals
- One inbox for all messaging platforms, no context-switching between apps.
- A stable internal API (`Messenger` interface) so adding a platform never touches core/UI code.
- Local-first: SQLite cache, fast startup, works semi-offline for history.
- Safe automation: AI agents can read/search/draft freely, but sending always needs human confirmation.

## 2. Non-Goals (for MVP)
- No cloud sync / hosted backend — local binary only.
- No mobile app.
- No WhatsApp support until Telegram + Discord are stable (policy/technical complexity).

## 3. Core Interfaces

```go
// core/messenger.go
type Messenger interface {
    Login() error
    Chats() ([]Chat, error)
    Messages(chatID string) ([]Message, error)
    SendMessage(chatID string, text string) error
    Search(query string) ([]Message, error)
    Logout() error
}
```

```go
// core/message.go
type Platform string

const (
    Telegram Platform = "telegram"
    WhatsApp Platform = "whatsapp"
    Discord  Platform = "discord"
)

type Message struct {
    ID          string
    Platform    Platform
    ChatID      string
    Sender      User
    Text        string
    Timestamp   time.Time
    Attachments []Attachment
}
```

## 4. Architecture

```
ZetaChat
   │
Universal API
   │
 ┌─┴────────────┐
 │              │
TUI/CLI      AI Agent
 │              │
 └──────┬───────┘
        │
Message Abstraction
        │
 ┌──────┼───────┬─────────┐
 ▼      ▼       ▼         ▼
TG     WA      DC      (future)
```

## 5. Feature List

| Feature | MVP Phase | Notes |
|---|---|---|
| Telegram auth + chats + messages | 0.1 | Foundation |
| Send/receive messages (Telegram) | 0.1 | |
| Basic search (Telegram only) | 0.1 | |
| SQLite local cache | 0.2 | |
| Notifications | 0.2 | `[TG] @arnav mentioned you` style |
| Keyboard shortcuts, polished TUI | 0.2 | |
| Unified search across platforms | 0.2–0.3 | Depends on 2+ adapters existing |
| Discord adapter | 0.3 | |
| Plugin/adapter install system | 0.3 | `zetachat plugin install <name>` |
| Multi-account support | 0.3 | |
| WhatsApp adapter (WhatsMeow-based) | 0.4 | Investigate before building |
| Signal / Matrix / Slack adapters | 0.4 | Stretch |
| JSON CLI output mode | 0.5 | For scripting + AI agents |
| AI-agent tools (search/get/send) | 0.5 | Send requires explicit confirm |
| Draft/summarize via AI | 0.5 | |

## 6. CLI Surface (target)

```
zeta chats                     # list all chats, all platforms
zeta chats --unread
zeta open telegram:@arnav
zeta search "hackathon"
zeta send telegram:@arnav "Hey"
zetachat plugin install telegram
zetachat plugin install discord
```

## 7. Tech Stack
- **Language:** Go
- **TUI:** Bubble Tea + Lip Gloss
- **Storage:** SQLite (`mattn/go-sqlite3` or `modernc.org/sqlite` for CGO-free builds)
- **Telegram:** MTProto library (e.g. `gotd/td`) or Bot API (`go-telegram-bot-api`) — decide based on whether you need full user-account access (MTProto) or bot-only (Bot API). For a personal unified inbox, MTProto is the right call.
- **Discord:** `discordgo`
- **WhatsApp (later):** `whatsmeow`

## 8. Repo Layout

```
zetachat/
├── core/
│   ├── message.go
│   ├── chat.go
│   ├── user.go
│   └── messenger.go
├── adapters/
│   ├── telegram/
│   ├── whatsapp/
│   └── discord/
├── tui/
│   ├── app.go
│   ├── inbox.go
│   ├── chat.go
│   ├── sidebar.go
│   └── composer.go
├── storage/
│   └── sqlite.go
├── config/
│   └── config.go
├── cmd/
│   └── zetachat/
│       └── main.go
├── go.mod
└── SPEC.md
```

## 9. Safety Rule (non-negotiable)
No adapter or AI-agent tool may call `SendMessage` without an explicit user confirmation step
surfaced in the TUI or CLI (`y/n` prompt, or `--confirm` flag for scripted use). This applies even
once agent tools are wired in — drafting and searching are unrestricted, sending is not.

## 10. Immediate Next Steps
1. `go mod init github.com/<you>/zetachat`
2. Get a barebones Bubble Tea shell running (`[A] Add account`, `[Q] Quit`).
3. Implement `core/` interfaces + structs (no logic yet, just types).
4. Build the Telegram adapter against the `Messenger` interface using `gotd/td`.
5. Wire adapter into TUI: list chats → open chat → send message.
6. Only once that loop works end-to-end, start on SQLite caching.

