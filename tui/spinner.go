package tui

import "github.com/charmbracelet/lipgloss"

func spinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lime).
		Bold(true)
}
