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
		} else if m.focusViewport {
			right = helpKey.Render("j/k") + helpDim.Render(" scroll ") +
				helpKey.Render("g/G") + helpDim.Render(" top/bot ") +
				helpKey.Render("esc") + helpDim.Render(" back")
		} else if m.focusComposer {
			right = helpKey.Render("enter") + helpDim.Render(" send ") +
				helpKey.Render("esc") + helpDim.Render(" sidebar")
		} else {
			right = helpKey.Render("j/k") + helpDim.Render(" nav ") +
				helpKey.Render("enter") + helpDim.Render(" open ") +
				helpKey.Render("tab") + helpDim.Render(" type ") +
				helpKey.Render("s") + helpDim.Render(" scroll ") +
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

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	// statusBar has horizontal padding of 1 on both sides.
	// Therefore only width-2 columns are available for actual content.
	contentWidth := width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Leave at least one space between the left and right sections.
	pad := contentWidth - leftWidth - rightWidth

	if pad < 1 {
		// There is not enough room for both sections.
		// Keep the status visible and hide the key hints rather
		// than allowing them to overflow past the terminal edge.
		right = ""
		rightWidth = 0
		pad = contentWidth - leftWidth
	}

	if pad < 1 {
		pad = 1
	}

	return statusBar.Render(
		left + strings.Repeat(" ", pad) + right,
	)
}
