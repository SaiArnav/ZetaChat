package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SaiArnav/ZetaChat/adapters/telegram"
	"github.com/SaiArnav/ZetaChat/config"
	"github.com/SaiArnav/ZetaChat/storage"
	"github.com/SaiArnav/ZetaChat/tui"
)

func main() {
	if len(os.Args) > 1 {
		if err := runCLI(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "zetachat:", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "zetachat:", err)
		os.Exit(1)
	}
	store, err := storage.Open(filepath.Join(cfg.DataDir, "cache.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "zetachat:", err)
		os.Exit(1)
	}
	defer store.Close()

	adapter := telegram.New(cfg.AppID, cfg.AppHash, cfg.SessionPath, nil)
	model := tui.NewModel(cfg, adapter, store)
	program := tea.NewProgram(model, tea.WithAltScreen())

	// The adapter asks for auth inputs through the TUI prompt.
	adapter.SetPrompter(&tui.UIPrompter{Program: program})

	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "zetachat:", err)
		os.Exit(1)
	}
}

// runCLI executes a subcommand: chats | open | search | send | whoami.
func runCLI(args []string) error {
	cmd := args[0]
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		usage()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := telegram.New(cfg.AppID, cfg.AppHash, cfg.SessionPath, nil)
	adapter.Connect(ctx)
	if err := adapter.Login(); err != nil {
		return fmt.Errorf("login: %w", err)
	}
	defer adapter.Close()

	switch cmd {
	case "chats":
		return cliChats(adapter, args[1:])
	case "open":
		return cliOpen(adapter, args[1:])
	case "search":
		return cliSearch(adapter, args[1:])
	case "send":
		return cliSend(ctx, adapter, args[1:])
	case "whoami":
		fmt.Println(adapter.SelfName())
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func usage() {
	fmt.Println(`ZetaChat — Telegram from the terminal.

  zetachat                     launch the TUI
  zetachat chats               list chats          (--json)
  zetachat open <chat>         show recent messages (--json)
  zetachat search <query>      search globally     (--json)
  zetachat send <chat> "<text>" send a message     (--yes to skip confirm)
  zetachat whoami              print the logged-in user

chat: "u123" | "g123" | "s123" | "@username"`)
}

// flagsJSON reports whether --json was passed and returns remaining args.
func flagsJSON(args []string) (rest []string, jsonOut bool) {
	for _, a := range args {
		switch a {
		case "--json", "-j":
			jsonOut = true
		default:
			rest = append(rest, a)
		}
	}
	return rest, jsonOut
}

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func cliChats(adapter *telegram.Adapter, args []string) error {
	rest, jsonOut := flagsJSON(args)
	_ = rest
	chats, err := adapter.Chats()
	if err != nil {
		return err
	}
	if jsonOut {
		printJSON(chats)
		return nil
	}
	for _, c := range chats {
		unread := ""
		if c.UnreadCount > 0 {
			unread = fmt.Sprintf("  [%d unread]", c.UnreadCount)
		}
		fmt.Printf("%-8s %s%s\n", c.ID, c.Name, unread)
	}
	return nil
}

func cliOpen(adapter *telegram.Adapter, args []string) error {
	rest, jsonOut := flagsJSON(args)
	if len(rest) < 1 {
		return errors.New("open requires a chat, e.g. zetachat open @username")
	}
	chatID := resolveChatID(rest[0])
	msgs, err := adapter.Messages(chatID)
	if err != nil {
		return err
	}
	if jsonOut {
		printJSON(msgs)
		return nil
	}
	if len(msgs) == 0 {
		fmt.Println("(no messages)")
		return nil
	}
	for _, m := range msgs {
		mark := "→"
		if m.Out {
			mark = "❯"
		}
		fmt.Printf("%s %-20s %s\n", mark, m.Sender.DisplayName, m.Text)
	}
	return nil
}

func cliSearch(adapter *telegram.Adapter, args []string) error {
	rest, jsonOut := flagsJSON(args)
	if len(rest) < 1 {
		return errors.New("search requires a query, e.g. zetachat search \"hello\"")
	}
	msgs, err := adapter.Search(strings.Join(rest, " "))
	if err != nil {
		return err
	}
	if jsonOut {
		printJSON(msgs)
		return nil
	}
	if len(msgs) == 0 {
		fmt.Println("no results")
		return nil
	}
	for _, m := range msgs {
		fmt.Printf("%-20s %s\n", m.Sender.DisplayName, m.Text)
	}
	return nil
}

func cliSend(ctx context.Context, adapter *telegram.Adapter, args []string) error {
	rest, _ := flagsJSON(args)

	yes := false
	kept := rest[:0]
	for _, a := range rest {
		if a == "--yes" {
			yes = true
		} else {
			kept = append(kept, a)
		}
	}
	rest = kept

	if len(rest) < 2 {
		return errors.New("send requires a chat and a message, e.g. zetachat send @arnav \"hello\"")
	}
	chatRef := rest[0]
	text := strings.Join(rest[1:], " ")

	// Never send silently: require an explicit confirmation.
	if !yes && isTTY() {
		fmt.Printf("Send to %q: %q [y/N] ", chatRef, text)
		var ans string
		if _, err := fmt.Scanln(&ans); err != nil || !strings.EqualFold(ans, "y") {
			return errors.New("send aborted")
		}
	} else if !yes {
		return errors.New("not a terminal; pass --yes to confirm the message")
	}

	chatID := resolveChatID(chatRef)
	if err := adapter.SendMessage(chatID, text); err != nil {
		return err
	}
	fmt.Println("sent ✓")
	return nil
}

// resolveChatID normalizes a chat reference to a core chatID.
func resolveChatID(ref string) string {
	ref = strings.TrimSpace(ref)
	if idRe.MatchString(ref) {
		return ref
	}
	return "@" + strings.TrimPrefix(ref, "@")
}

// idRe matches core chatIDs: "u123", "g123", "s123".
var idRe = regexp.MustCompile(`^[ugs][0-9]+$`)

// isTTY reports whether stdin is an interactive terminal.
func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
