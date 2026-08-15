package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// statusView renders the bottom bar: live status on the left, keys on the right.
func (m Model) statusView() string {
	left := dimStyle.Render(m.status)
	if left == "" {
		left = dimStyle.Render(" ")
	}

	var right string
	switch m.stage {
	case stageReady:
		if m.searchMode {
			right = helpKey.Render("enter") + helpDim.Render(" search ") +
				helpKey.Render("j/k") + helpDim.Render(" pick ") +
				helpKey.Render("esc") + helpDim.Render(" close")
		} else if m.focusComposer {
			right = helpKey.Render("enter") + helpDim.Render(" send ") +
				helpKey.Render("esc") + helpDim.Render(" sidebar")
		} else {
			right = helpKey.Render("j/k") + helpDim.Render(" nav ") +
				helpKey.Render("enter") + helpDim.Render(" open ") +
				helpKey.Render("tab") + helpDim.Render(" type ") +
				helpKey.Render("/") + helpDim.Render(" search ") +
				helpKey.Render("r") + helpDim.Render(" refresh ") +
				helpKey.Render("q") + helpDim.Render(" quit")
		}
	default:
		right = helpKey.Render("ctrl+c") + helpDim.Render(" quit")
	}

	width := m.width
	if width < 1 {
		width = 80
	}
	pad := width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return statusBar.Render(left + strings.Repeat(" ", pad) + right)
}
