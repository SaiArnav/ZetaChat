# Contributing to ZetaChat

Thanks for wanting to contribute! Here's how to get started.

## Setup

```bash
go mod tidy
make build        # or: go build -o zetachat ./cmd/zetachat
```

Create `.env` with your Telegram API credentials (see the README).

## Commands

- `make build` — compile the binary
- `make test` — run tests
- `make vet` — static analysis
- `make fmt` — format the code

## Before you open a PR

1. Run `make fmt`, `make vet` and `make test` — everything must pass.
2. Keep the scope of the change focused and explain it in the PR description.
3. Update the README if the change affects usage.

## Project layout

```
cmd/zetachat/   entry point: TUI + CLI dispatch
tui/            Bubble Tea model, views, theme, keys
adapters/       platform adapters (telegram/ via gotd/td)
core/           platform-neutral types (Message, Chat, Messenger)
config/         .env loading + paths
storage/        SQLite cache
```

## Code style

- Follow `gofmt` output.
- No comments unless they explain *why*, not *what*.
- Keep the `core.Messenger` interface platform-neutral — new platforms are new
  adapters, not changes to core.

## License

By contributing you agree that your contributions are licensed under the
[MIT license](LICENSE).
