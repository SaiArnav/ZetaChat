package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// modeName derives the current interaction mode chip shown in the bar.
func (m Model) modeName() string {
	if m.helpOpen {
		return "HELP"
	}
	switch m.stage {
	case stageBooting:
		return "BOOT"
	case stageLoading:
		return "LOAD"
	case stageAuth:
		return "AUTH"
	case stageQR:
		return "PAIR"
	case stageError:
		return "ERROR"
	case stageDashboard:
		return "DASH"
	case stageReady:
		if m.searchMode {
			return "SEARCH"
		}
		switch m.focus {
		case PaneComposer:
			if m.confirming {
				return "CONFIRM"
			}
			return "EDIT"
		case PaneMessages:
			return "VIEW"
		default:
			return "CHATS"
		}
	}
	return "…"
}

// statusView renders the bottom bar: mode chip + live status on the
// left, contextual key hints (from keyMap) on the right. It re-derives
// everything from current state on every render, so every focus/mode
// transition is reflected immediately.
func (m Model) statusView() string {
	chip := modeChip.Render(m.modeName())

	left := chip + " " + dimStyle.Render(strings.TrimPrefix(m.status, "\a"))

	// The mouse-off notice is a luxury; drop it on narrow terminals.
	if !m.mouseOn && lipgloss.Width(left)+14 < m.width-2 {
		left = chip + dimStyle.Render(" · mouse off") +
			" " + dimStyle.Render(strings.TrimPrefix(m.status, "\a"))
	}

	var right string
	if !m.helpOpen {
		right = m.help.ShortHelpView(shortHelpFor(m))
	}

	width := m.width
	if width < 1 {
		width = 80
	}

	contentWidth := width - 2 // statusBar padding

	pad := contentWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		// Not enough room: keep status, drop hints.
		right = ""
		pad = contentWidth - lipgloss.Width(left)
	}
	if pad < 1 {
		pad = 1
	}

	line := left + strings.Repeat(" ", pad) + right

	// Never overflow: hard-clip whatever remains.
	return statusBar.Render(truncateRight(line, contentWidth))
}
