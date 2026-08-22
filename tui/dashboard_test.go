package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SaiArnav/ZetaChat/config"
	"github.com/SaiArnav/ZetaChat/core"
)

// fakeLive is a minimal core.LiveMessenger stub for tests.
type fakeLive struct {
	plat    core.Platform
	readyCh chan struct{}
	err     error
	self    string
	updates chan core.Message
}

func newFakeLive(p core.Platform) *fakeLive {
	return &fakeLive{
		plat:    p,
		readyCh: make(chan struct{}),
		updates: make(chan core.Message, 1),
	}
}

func (f *fakeLive) Login() error                            { return nil }
func (f *fakeLive) Chats() ([]core.Chat, error)             { return nil, nil }
func (f *fakeLive) Messages(string) ([]core.Message, error) { return nil, nil }
func (f *fakeLive) SendMessage(string, string) error        { return nil }
func (f *fakeLive) Search(string) ([]core.Message, error)   { return nil, nil }
func (f *fakeLive) Logout() error                           { return nil }
func (f *fakeLive) PlatformName() core.Platform             { return f.plat }
func (f *fakeLive) Connect(ctx context.Context)             { close(f.readyCh) }
func (f *fakeLive) Ready() <-chan struct{}                  { return f.readyCh }
func (f *fakeLive) AuthErr() error                          { return f.err }
func (f *fakeLive) SelfName() string                        { return f.self }
func (f *fakeLive) Updates() <-chan core.Message            { return f.updates }
func (f *fakeLive) Close() error                            { return nil }

func TestDashboardRendersPlatformCards(t *testing.T) {
	tg := newFakeLive(core.PlatformTelegram)
	wa := newFakeLive(core.PlatformWhatsApp)

	m := NewModel(&config.Config{}, map[core.Platform]core.LiveMessenger{
		core.PlatformTelegram: tg,
		core.PlatformWhatsApp: wa,
	}, nil)

	view := m.dashboardView()
	for _, want := range []string{"TELEGRAM", "WHATSAPP", "ZETACHAT"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dashboard missing %q\n%s", want, view)
		}
	}
	if got := len(m.platOrder); got != 2 {
		t.Fatalf("platOrder = %d, want 2", got)
	}
	// Telegram is always first in the display order.
	if m.platOrder[0] != core.PlatformTelegram {
		t.Fatalf("first platform = %v, want telegram", m.platOrder[0])
	}
}

func TestSelectPlatformEntersLoading(t *testing.T) {
	tg := newFakeLive(core.PlatformTelegram)
	close(tg.readyCh)

	m := NewModel(&config.Config{}, map[core.Platform]core.LiveMessenger{
		core.PlatformTelegram: tg,
	}, nil)
	st := m.platState[core.PlatformTelegram]
	st.ready = true

	updated, cmd := m.selectPlatform(core.PlatformTelegram)
	got := updated.(Model)

	if got.stage != stageLoading {
		t.Fatalf("stage = %v, want stageLoading", got.stage)
	}
	if cmd == nil {
		t.Fatal("selectPlatform returned no command")
	}
}

func TestSelectPlatformReportsError(t *testing.T) {
	tg := newFakeLive(core.PlatformTelegram)
	close(tg.readyCh)

	m := NewModel(&config.Config{}, map[core.Platform]core.LiveMessenger{
		core.PlatformTelegram: tg,
	}, nil)
	m.stage = stageDashboard
	m.platState[core.PlatformTelegram].ready = true
	m.platState[core.PlatformTelegram].err = errors.New("boom")

	updated, _ := m.selectPlatform(core.PlatformTelegram)
	got := updated.(Model)

	if got.stage != stageDashboard {
		t.Fatalf("stage = %v, want stageDashboard on error", got.stage)
	}
}

func TestFilterPlatformKeepsLegacyRowsOnTelegram(t *testing.T) {
	chats := []core.Chat{
		{ID: "u1", Platform: core.PlatformTelegram},
		{ID: "wa-9", Platform: core.PlatformWhatsApp},
		{ID: "u2", Platform: ""},
	}

	tgOnly := filterPlatform(chats, core.PlatformTelegram)
	if len(tgOnly) != 2 {
		t.Fatalf("telegram filter = %d chats, want 2", len(tgOnly))
	}

	waOnly := filterPlatform(chats, core.PlatformWhatsApp)
	if len(waOnly) != 1 || waOnly[0].ID != "wa-9" {
		t.Fatalf("whatsapp filter = %+v, want only wa-9", waOnly)
	}
}

func TestPlatOrderIsDeterministic(t *testing.T) {
	order := platOrder([]core.Platform{
		core.PlatformWhatsApp, core.PlatformTelegram,
	})
	if len(order) != 2 || order[0] != core.PlatformTelegram || order[1] != core.PlatformWhatsApp {
		t.Fatalf("platOrder = %v, want [telegram whatsapp]", order)
	}
}
