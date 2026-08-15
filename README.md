# ZetaChat

Terminal-first, plugin-based universal messaging client. See `SPEC.md` for full design.

## Setup

1. Rename the module path (currently `github.com/Vishvas77/zetachat`) in every
   `.go` file and in `go.mod` to your actual GitHub username/repo.
   Quick way once you've decided:
   ```bash
   grep -rl "Vishvas77" . | xargs sed -i 's/Vishvas77/your-actual-username/g'
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```
   This will pull in Bubble Tea, Lip Gloss, and (once you add the import) the
   Telegram MTProto client and SQLite driver.

3. Run the shell UI (currently just the "no accounts connected" screen):
   ```bash
   go run ./cmd/zetachat
   ```

## Current State

- ✅ Core types (`Message`, `Chat`, `User`, `Messenger` interface)
- ✅ Bubble Tea shell (`[A] Add account`, `[Q] Quit`)
- ✅ Telegram adapter skeleton (methods stubbed, return "not implemented")
- ⬜ Telegram auth flow (MTProto)
- ⬜ Wiring adapter → TUI (list chats, open chat, send)
- ⬜ SQLite caching
- ⬜ Discord adapter
- ⬜ Plugin install system
- ⬜ AI-agent tool layer

## Next Step

Implement `adapters/telegram/telegram.go`'s `Login()` and `Chats()` using
`gotd/td`, then wire a real "Add account" flow into `tui/app.go` so pressing
`[A]` actually authenticates and populates the sidebar.
