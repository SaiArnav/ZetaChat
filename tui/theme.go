package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Neon palette — bold, high-contrast terminal design.
// primary lime / secondary cyan on an adaptive dark surface.
const (
	colLime   = "#BBF351"
	colCyan   = "#00BCFF"
	colDanger = "#DC2626"
	colWarn   = "#D97706"
	colDim    = "#8A8F98"
	colGray   = "#3C4048"
)

var (
	// text adapts to the terminal theme (light/dark).
	textColor = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#EDEDEF"}
	dimColor  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8A8F98"}
	muted     = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#52575F"}

	lime = lipgloss.Color(colLime)
	cyan = lipgloss.Color(colCyan)
	red  = lipgloss.Color(colDanger)
	amber = lipgloss.Color(colWarn)
	gray = lipgloss.Color(colGray)

	bg = lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#0F1012"}

	// baseText is the default text style used across the app.
	baseText = lipgloss.NewStyle().Foreground(textColor)

	bannerStyle = lipgloss.NewStyle().
			Foreground(lime).
			Bold(true)

	titleStyle = lipgloss.NewStyle().
			Foreground(lime).
			Bold(true)

	dimStyle = lipgloss.NewStyle().Foreground(dimColor)

	sidebarStyle = lipgloss.NewStyle().
			Background(bg).
			Padding(0, 1)

	chatItemStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Width(28)

	chatItemSelected = lipgloss.NewStyle().
				Background(lime).
				Foreground(lipgloss.Color("#111827")).
				Bold(true).
				Padding(0, 1).
				Width(28)

	unreadBadge = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	selectedBadge = lipgloss.NewStyle().
			Background(lime).
			Foreground(lipgloss.Color("#111827")).
			Bold(true)

	statusBar = lipgloss.NewStyle().
			Background(gray).
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#D1D5DB"}).
			Padding(0, 1)

	helpKey = lipgloss.NewStyle().
			Foreground(lime).
			Bold(true)

	helpDim = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#9CA3AF"})

	flashStyle = lipgloss.NewStyle().
			Foreground(lime).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true)

	chatTitleStyle = lipgloss.NewStyle().
			Foreground(lime).
			Bold(true).
			Margin(0, 0, 0, 1)
)
