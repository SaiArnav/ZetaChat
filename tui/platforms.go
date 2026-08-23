package tui

import (
	"sort"
	"strings"

	"github.com/SaiArnav/ZetaChat/core"
)

// platformMeta holds the dashboard presentation data for one platform.
type platformMeta struct {
	name    string
	tagline string
	accent  string // hex accent color for the card
	icon    []string
}

var platformRegistry = map[core.Platform]platformMeta{
	core.PlatformAll: {
		name:    "INBOX",
		tagline: "unified · every platform merged",
		accent:  colAccent,
		icon: []string{
			"    /\\     _      _    ",
			"   /  \\   | |    | |   ",
			"  / /\\ \\  | |    | |   ",
			" / ____ \\ | |___ | |___",
			"/_/    \\_\\|_____||_____|",
		},
	},
	core.PlatformTelegram: {
		name:    "TELEGRAM",
		tagline: "MTProto · personal account",
		accent:  "#D9A441", // ember gold
		icon: []string{
			" _____   ____ ",
			"|_   _| / ___|",
			"  | |  | |  _ ",
			"  | |  | |_| |",
			"  |_|   \\____|",
		},
	},
	core.PlatformWhatsApp: {
		name:    "WHATSAPP",
		tagline: "QR linked · personal account",
		accent:  "#93A978", // ember sage
		icon: []string{
			"__        __     /\\    ",
			"\\ \\      / /    /  \\   ",
			" \\ \\ /\\ / /    / /\\ \\  ",
			"  \\ V  V /    / ____ \\ ",
			"   \\_/\\_/    /_/    \\_\\",
		},
	},
}

// platOrder is the deterministic display order on the dashboard.
func platOrder(platforms []core.Platform) []core.Platform {
	order := []core.Platform{core.PlatformTelegram, core.PlatformWhatsApp}
	out := make([]core.Platform, 0, len(order))
	for _, p := range order {
		for _, q := range platforms {
			if q == p {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// dashCards is the full card row on the dashboard: the unified inbox
// first, then one card per configured platform.
func dashCards(platOrder []core.Platform) []core.Platform {
	return append([]core.Platform{core.PlatformAll}, platOrder...)
}

// sortChatsByRecency orders a merged inbox: most recently updated first;
// chats without a timestamp fall back to name order at the end.
func sortChatsByRecency(chats []core.Chat) {
	sort.SliceStable(chats, func(i, j int) bool {
		ti, tj := chats[i].UpdatedAt, chats[j].UpdatedAt
		if !ti.IsZero() && !tj.IsZero() {
			return ti.After(tj)
		}
		if ti.IsZero() != tj.IsZero() {
			return tj.IsZero()
		}
		return chats[i].Name < chats[j].Name
	})
}

// chatTag renders a small colored platform chip for merged-inbox rows.
func chatTag(p core.Platform) string {
	meta, ok := platformRegistry[p]
	if !ok || p == "" || p == core.PlatformAll {
		return ""
	}
	short := map[core.Platform]string{
		core.PlatformTelegram: "TG",
		core.PlatformWhatsApp: "WA",
	}[p]
	if short == "" {
		short = strings.ToUpper(string(p[:2]))
	}
	return lipglossAccent(meta.accent).Render("[" + short + "]")
}

// statusDot renders a colored connection indicator.
func (s *platformState) statusDot() string {
	switch {
	case s.err != nil:
		return errorStyle.Render("✗")
	case s.ready:
		return flashStyle.Render("●")
	case s.pairing:
		return amberStyle.Render("◇")
	default:
		return dimStyle.Render("◌")
	}
}

// statusLabel renders a short textual connection state.
func (s *platformState) statusLabel() string {
	switch {
	case s.err != nil:
		return "OFFLINE"
	case s.ready:
		return "ONLINE"
	case s.pairing:
		return "SCAN TO LINK"
	default:
		return "LINKING…"
	}
}

// aggregateDot summarizes every configured platform for the INBOX card.
func (m Model) aggregateDot() (string, string) {
	online, pairing, offline := 0, 0, 0
	for _, p := range m.platOrder {
		switch st := m.platState[p]; {
		case st == nil:
		case st.err != nil:
			offline++
		case st.ready:
			online++
		case st.pairing:
			pairing++
		}
	}
	switch {
	case online > 0:
		return flashStyle.Render("●"), "MERGED"
	case pairing > 0:
		return amberStyle.Render("◇"), "PARTIAL"
	default:
		return dimStyle.Render("◌"), "OFFLINE"
	}
}
