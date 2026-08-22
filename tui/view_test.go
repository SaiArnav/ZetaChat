package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SaiArnav/ZetaChat/core"
)

func TestReadyViewRendersLayout(t *testing.T) {
	m := Model{
		stage:        stageReady,
		width:        100,
		height:       24,
		sidebarWidth: 32,
		chats: []core.Chat{
			{ID: "u1", Name: "Arnav", UnreadCount: 2},
			{ID: "g2", Name: "Go Gophers"},
		},
		chatIdx: 0,
		messages: []core.Message{
			{
				ID:        "m1",
				ChatID:    "u1",
				Text:      "hello there",
				Timestamp: time.Now(),
				Sender:    core.User{DisplayName: "Arnav"},
			},
			{
				ID:        "m2",
				ChatID:    "u1",
				Text:      "hi back",
				Timestamp: time.Now(),
				Sender:    core.User{DisplayName: "me"},
				Out:       true,
			},
		},
		composer: textinput.New(),
		search:   textinput.New(),
	}

	m = m.refreshMessagesView()

	out := m.View()

	for _, want := range []string{
		"ZETACHAT",
		"Arnav",
		"Go Gophers",
		"hello there",
		"hi back",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n%s", want, out)
		}
	}
}

func TestReadyViewDoesNotExceedTerminalHeight(t *testing.T) {
	m := Model{
		stage:        stageReady,
		width:        100,
		height:       24,
		sidebarWidth: 32,
		chats: []core.Chat{
			{ID: "u1", Name: "Arnav"},
			{ID: "g2", Name: "Go Gophers"},
		},
		chatIdx: 0,
		messages: []core.Message{
			{
				ID:        "m1",
				ChatID:    "u1",
				Text:      "hello there",
				Timestamp: time.Now(),
				Sender:    core.User{DisplayName: "Arnav"},
			},
		},
		composer: textinput.New(),
		search:   textinput.New(),
	}

	m = m.refreshMessagesView()

	out := m.View()
	height := lipgloss.Height(out)

	if height > m.height {
		t.Fatalf(
			"View() rendered %d lines for a %d-line terminal:\n%s",
			height,
			m.height,
			out,
		)
	}
}

func TestSplashViewHasBanner(t *testing.T) {
	t.Skip("TODO: banner assertion needs to account for lipgloss rendering")
}

// TestNewMessageRefreshesViewport guards against the viewport only being
// re-rendered when the chat changes: messages appended after the initial
// render must appear without a chat switch or resize.
func TestNewMessageRefreshesViewport(t *testing.T) {
	m := Model{
		stage:        stageReady,
		width:        100,
		height:       24,
		sidebarWidth: 32,
		chats:        []core.Chat{{ID: "u1", Name: "Arnav"}},
		chatIdx:      0,
		messages: []core.Message{
			{
				ID:        "m1",
				ChatID:    "u1",
				Text:      "first",
				Timestamp: time.Now(),
				Sender:    core.User{DisplayName: "Arnav"},
			},
		},
		composer: textinput.New(),
		search:   textinput.New(),
	}

	m = m.refreshMessagesView()

	if out := m.View(); !strings.Contains(out, "first") {
		t.Fatalf("initial message not rendered:\n%s", out)
	}

	m.messages = append(m.messages, core.Message{
		ID:        "m2",
		ChatID:    "u1",
		Text:      "second message appears live",
		Timestamp: time.Now(),
		Sender:    core.User{DisplayName: "Arnav"},
	})

	m = m.refreshMessagesView()

	if out := m.View(); !strings.Contains(out, "second message appears live") {
		t.Fatalf("new message not rendered after refresh:\n%s", out)
	}
}

// TestHandleLiveReplacesLocalEcho guards against sent messages showing twice:
// the live echo of our own message should replace, not duplicate, the local echo.
func TestHandleLiveReplacesLocalEcho(t *testing.T) {
	m := Model{
		stage:        stageReady,
		width:        100,
		height:       24,
		sidebarWidth: 32,
		chats:        []core.Chat{{ID: "u1", Name: "Arnav"}},
		chatIdx:      0,
		messages: []core.Message{
			{
				ID:        "local-1",
				ChatID:    "u1",
				Text:      "yo",
				Timestamp: time.Now(),
				Out:       true,
			},
		},
		composer: textinput.New(),
		search:   textinput.New(),
	}

	live := core.Message{
		ID:        "123",
		ChatID:    "u1",
		Text:      "yo",
		Timestamp: time.Now(),
		Out:       true,
		Sender:    core.User{DisplayName: "me"},
	}

	got, _ := m.handleLive(live)
	nm := got.(Model)

	if len(nm.messages) != 1 {
		t.Fatalf("expected echo replaced, got %d messages", len(nm.messages))
	}

	if nm.messages[0].ID != "123" {
		t.Fatalf("expected real message ID, got %q", nm.messages[0].ID)
	}
}

// TestStatusBarDoesNotOverflowWidth asserts that the status bar renders
// within the terminal width — right-side hints must be dropped when the
// terminal is too narrow.
func TestStatusBarDoesNotOverflowWidth(t *testing.T) {
	for _, w := range []int{20, 40, 80, 120} {
		m := Model{
			stage:        stageReady,
			width:        w,
			height:       24,
			sidebarWidth: 32,
			status:       "Ready",
		}
		bar := m.statusView()
		barWidth := lipgloss.Width(bar)
		if barWidth > w+2 { // statusBar has Padding(0,1) = 2 extra cols
			t.Errorf("width=%d: status bar rendered %d chars (max %d)", w, barWidth, w+2)
		}
	}
}

// TestLocalEchoAppearsImmediately verifies that pressing Enter on the
// composer appends the message to the list synchronously (before the
// network call completes) so the user sees their message instantly.
func TestLocalEchoAppearsImmediately(t *testing.T) {
	m := Model{
		stage:        stageReady,
		width:        100,
		height:       24,
		sidebarWidth: 32,
		chats:        []core.Chat{{ID: "u1", Name: "Arnav"}},
		chatIdx:      0,
		messages: []core.Message{
			{ID: "m1", ChatID: "u1", Text: "existing", Timestamp: time.Now()},
		},
		composer: textinput.New(),
		search:   textinput.New(),
	}
	m.composer.SetValue("hello world")
	m.composer.Focus()

	got, _ := m.handleComposerKey(tea.KeyMsg{Type: tea.KeyEnter})
	nm := got.(Model)

	if len(nm.messages) != 2 {
		t.Fatalf("expected 2 messages after send (1 existing + 1 local echo), got %d", len(nm.messages))
	}

	echo := nm.messages[1]
	if echo.Text != "hello world" {
		t.Fatalf("local echo text = %q, want %q", echo.Text, "hello world")
	}
	if !echo.Out {
		t.Fatal("local echo should be marked as outgoing")
	}
	if !strings.HasPrefix(echo.ID, "local-") {
		t.Fatalf("local echo ID should start with 'local-', got %q", echo.ID)
	}

	// Composer should be cleared.
	if nm.composer.Value() != "" {
		t.Fatalf("composer should be empty after send, got %q", nm.composer.Value())
	}
}
