package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/SaiArnav/ZetaChat/core"
)

func TestReadyViewRendersLayout(t *testing.T) {
	m := Model{
		stage:  stageReady,
		width:  100,
		height: 24,
		chats: []core.Chat{
			{ID: "u1", Name: "Arnav", UnreadCount: 2},
			{ID: "g2", Name: "Go Gophers"},
		},
		chatIdx:  0,
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
	for _, want := range []string{"ZETACHAT", "Arnav", "Go Gophers", "hello there", "hi back"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n%s", want, out)
		}
	}
}

func TestSplashViewHasBanner(t *testing.T) {
	m := Model{
		stage:  stageBooting,
		width:  80,
		height: 24,
	}
	out := m.View()
	if !strings.Contains(out, "████") {
		t.Fatalf("splash missing banner:\n%s", out)
	}
}
