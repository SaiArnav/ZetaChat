# ZetaChat — Features & TODO

A living checklist. Update as things get built. Grouped by MVP phase, matching SPEC.md.

---

## Feature List (What ZetaChat Does)

### Unified Messaging
- [ ] Single inbox showing chats from every connected platform, tagged (WA/TG/DC)
- [ ] Open any chat and read full message history regardless of platform
- [ ] Send a message to any chat from one composer
- [ ] Unified search across all connected platforms
- [ ] Unified unread view (`zeta chats --unread`)
- [ ] Cross-platform notifications (`[TG] @arnav mentioned you`)
- [ ] View/download attachments shared across platforms

### Multi-Platform Support
- [ ] Telegram adapter (first)
- [ ] Discord adapter (second)
- [ ] WhatsApp adapter (investigate WhatsMeow, build later — policy/complexity risk)
- [ ] Signal adapter (stretch)
- [ ] Matrix adapter (stretch)
- [ ] Slack adapter (stretch)

### Platform System
- [ ] Every adapter implements the same `Messenger` interface — core/TUI never special-cases a platform
- [ ] Plugin install system (`zetachat plugin install telegram`)
- [ ] Multiple accounts per platform

### Local-First
- [ ] SQLite cache for chats/messages (fast startup, partial offline read)
- [ ] Config file for enabled plugins, data dir, etc.

### CLI / Scripting
- [ ] `zeta chats`, `zeta chats --unread`
- [ ] `zeta open <platform>:<chat>`
- [ ] `zeta search "<query>"`
- [ ] `zeta send <platform>:<chat> "<text>"`
- [ ] JSON output mode for all commands (machine-readable)

### AI-Agent Integration
- [ ] Agent tools: `messaging.search`, `messaging.get_chat`, `messaging.get_messages`, `messaging.list_unread`
- [ ] Agent tool: `messaging.send_message` — **must require explicit human confirmation**, never auto-fires
- [ ] Draft-reply / summarize helpers

---

## TODO — In Build Order

### MVP 0.1 — Telegram Loop Works End-to-End
- [ ] Rename module path (`Vishvas77` → real username) across repo
- [ ] `go mod tidy` — pull in Bubble Tea, Lip Gloss
- [ ] Add `gotd/td` (or chosen MTProto lib) dependency
- [ ] Implement `Adapter.Login()` — phone/code or QR auth, persist session to `sessionPath`
- [ ] Implement `Adapter.Chats()` — fetch dialogs, map to `[]core.Chat`
- [ ] Implement `Adapter.Messages(chatID)` — fetch history, map to `[]core.Message`
- [ ] Implement `Adapter.SendMessage(chatID, text)`
- [ ] Implement `Adapter.Search(query)`
- [ ] Wire `[A] Add account` keypress in `tui/app.go` to call `Login()`
- [ ] Build sidebar view listing chats after login
- [ ] Build chat view: open a chat, show messages, type + send
- [ ] Manual end-to-end test: login → see chats → open one → send a message → confirm it lands on Telegram

### MVP 0.2 — Cache, Polish, Notifications
- [ ] Implement `storage.Open()` with real SQLite driver (`modernc.org/sqlite`)
- [ ] Create schema/migrations: `chats`, `messages`, `users` tables
- [ ] Implement `SaveChats` / `SaveMessages` / `CachedChats` / `CachedMessages`
- [ ] On startup, render cached data instantly, then refresh from adapter in background
- [ ] Add keyboard shortcuts (navigate chats, jump to search, quit, etc.)
- [ ] Add basic desktop/terminal notification on new message
- [ ] Implement `zeta chats --unread`

### MVP 0.3 — Discord + Plugin System
- [ ] Build Discord adapter using `discordgo`, implementing `Messenger`
- [ ] Implement unified search across 2+ adapters (fan-out + merge results)
- [ ] Design plugin manifest format + `zetachat plugin install <name>`
- [ ] Support multiple accounts per platform (config + adapter instance keyed by account)

### MVP 0.4 — WhatsApp + Others
- [ ] Investigate `whatsmeow` — auth flow, rate limits, ToS risk
- [ ] Build WhatsApp adapter if viable
- [ ] Evaluate Signal / Matrix / Slack adapters

### MVP 0.5 — AI Agent Layer
- [ ] Add `--json` output flag to all CLI commands
- [ ] Define agent tool schemas: `messaging.search`, `messaging.get_chat`, `messaging.get_messages`, `messaging.list_unread`, `messaging.send_message`
- [ ] Build confirmation gate for `send_message` (CLI prompt or TUI modal — no silent sends, ever)
- [ ] Add draft/summarize helper commands

---

## Immediate Next Action
Implement `Adapter.Login()` in `adapters/telegram/telegram.go` using `gotd/td`, and wire `[A]` in
`tui/app.go` to call it. That's the one thing standing between the current shell and a real, usable
first version.

