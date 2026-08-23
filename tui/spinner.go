package tui

import "github.com/charmbracelet/lipgloss"

func spinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)
}
