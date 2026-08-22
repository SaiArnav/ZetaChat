package tui

import (
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
	core.PlatformTelegram: {
		name:    "TELEGRAM",
		tagline: "MTProto · personal account",
		accent:  "#2AABEE",
		icon: []string{
			"   ▄▄▄▄▄▄▄   ",
			"  █████████▄ ",
			"   ████▀███▄ ",
			"    ██▄ ▀███ ",
			"   ▄████▄▀▀  ",
			"  ▀▀▀▀▀▀▀▀   ",
		},
	},
	core.PlatformWhatsApp: {
		name:    "WHATSAPP",
		tagline: "QR linked · personal account",
		accent:  "#25D366",
		icon: []string{
			"  ▄▄▄▄▄▄▄▄▄  ",
			" █ ███████ █ ",
			"█  ███████  █",
			"█  █████  ▄▀",
			" ██ ████ ▄▀  ",
			"  ███████▀   ",
			"   ▀▀▀▀▀▀    ",
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
