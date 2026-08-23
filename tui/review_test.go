package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/SaiArnav/ZetaChat/config"
	"github.com/SaiArnav/ZetaChat/core"
)

func TestFilterPlatformKeepsEverythingForInbox(t *testing.T) {
	chats := []core.Chat{
		{ID: "u1", Platform: core.PlatformTelegram},
		{ID: "wa-9", Platform: core.PlatformWhatsApp},
	}
	got := filterPlatform(chats, core.PlatformAll)
	if len(got) != 2 {
		t.Fatalf("inbox filter = %d chats, want 2", len(got))
	}
}

func TestSortChatsByRecency(t *testing.T) {
	now := time.Now()
	chats := []core.Chat{
		{Name: "old", UpdatedAt: now.Add(-2 * time.Hour)},
		{Name: "new", UpdatedAt: now},
		{Name: "mid", UpdatedAt: now.Add(-time.Minute)},
		{Name: "zzz-no-time"},
	}
	sortChatsByRecency(chats)
	want := []string{"new", "mid", "old", "zzz-no-time"}
	for i, name := range want {
		if chats[i].Name != name {
			t.Fatalf("order[%d] = %s, want %s", i, chats[i].Name, name)
		}
	}
}

func TestDashCardsPutsInboxFirst(t *testing.T) {
	cards := dashCards([]core.Platform{
		core.PlatformTelegram, core.PlatformWhatsApp,
	})
	if cards[0] != core.PlatformAll {
		t.Fatalf("first card = %v, want all", cards[0])
	}
	if m := NewModel(&config.Config{}, map[core.Platform]core.LiveMessenger{}, nil); m.dashCardAt() != core.PlatformAll {
		// With no platforms configured, the inbox card is still present.
		t.Fatalf("dashCardAt on empty model = %q, want all", m.dashCardAt())
	}
}

func TestHelpOverlayListsSections(t *testing.T) {
	m := NewModel(&config.Config{}, map[core.Platform]core.LiveMessenger{}, nil)
	m.helpOpen = true
	m.width = 100
	m.height = 40
	out := renderHelpOverlay(m)
	for _, want := range []string{"KEYBINDINGS", "shift+enter", "toggle mouse", "send message"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help overlay missing %q", want)
		}
	}
}

func TestComposerHeightGrowsAndClamps(t *testing.T) {
	m := NewModel(&config.Config{}, map[core.Platform]core.LiveMessenger{}, nil)
	m.width = 100
	m.height = 40
	m.stage = stageReady

	if got := m.composerHeight(); got != minComposerLines {
		t.Fatalf("empty composer height = %d, want %d", got, minComposerLines)
	}

	m.composer.SetValue(strings.Repeat("word ", 400) + "\n" +
		strings.Repeat("word ", 400))
	if got := m.composerHeight(); got > maxComposerLines {
		t.Fatalf("composer height = %d, want clamped to %d", got, maxComposerLines)
	}
}

func TestRenderMessagesUsesCache(t *testing.T) {
	msg := core.Message{ID: "m1", Text: "hi", Timestamp: time.Now()}
	cache := map[string]string{}

	first := renderMessages([]core.Message{msg}, cache, 80)
	if len(cache) == 0 {
		t.Fatal("cache not populated")
	}

	// Poison the cache entry; if the cache is honored, the poisoned
	// value shows up instead of a fresh render.
	for k := range cache {
		cache[k] = "CACHED"
	}
	if got := renderMessages([]core.Message{msg}, cache, 80); got != "CACHED" {
		t.Fatalf("render ignored cache: %q", got)
	}

	// Changing the text invalidates the entry.
	msg.Text = "edited"
	if got := renderMessages([]core.Message{msg}, cache, 80); got == "CACHED" {
		t.Fatal("stale cached render served for edited message")
	}
	_ = first
}
