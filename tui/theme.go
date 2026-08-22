package tui

import "github.com/charmbracelet/lipgloss"

// Neon palette — bold, high-contrast terminal design.
const (
	colLime   = "#BBF351"
	colCyan   = "#00BCFF"
	colDanger = "#DC2626"
	colWarn   = "#D97706"
	colDim    = "#8A8F98"
	colGray   = "#3C4048"
	colBg     = "#0F1012"
	colText   = "#EDEDEF"
	colDark   = "#111827"
	colMuted  = "#52575F"
	colBarTxt = "#D1D5DB"
)

var (
	// Fixed colors are used instead of AdaptiveColor so the UI stays
	// consistent across Windows terminals and different terminal themes.
	textColor = lipgloss.Color(colText)
	dimColor  = lipgloss.Color(colDim)
	muted     = lipgloss.Color(colMuted)

	lime  = lipgloss.Color(colLime)
	cyan  = lipgloss.Color(colCyan)
	red   = lipgloss.Color(colDanger)
	amber = lipgloss.Color(colWarn)
	gray  = lipgloss.Color(colGray)
	bg    = lipgloss.Color(colBg)

	// baseText is the default text style used across the app.
	baseText = lipgloss.NewStyle().
			Foreground(textColor)

	bannerStyle = lipgloss.NewStyle().
			Foreground(lime).
			Bold(true)

	titleStyle = lipgloss.NewStyle().
			Foreground(lime).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Used for the left chat list.
	sidebarStyle = lipgloss.NewStyle().
			Background(bg).
			Padding(0, 1)

	// Used for the main chat pane background.
	chatPaneStyle = lipgloss.NewStyle().
			Background(bg)

	// Used for the top application header.
	headerBarStyle = lipgloss.NewStyle().
			Background(bg)

	chatItemStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Width(28)

	chatItemSelected = lipgloss.NewStyle().
				Background(lime).
				Foreground(lipgloss.Color(colDark)).
				Bold(true).
				Padding(0, 1).
				Width(28)

	unreadBadge = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	selectedBadge = lipgloss.NewStyle().
			Background(lime).
			Foreground(lipgloss.Color(colDark)).
			Bold(true)

	statusBar = lipgloss.NewStyle().
			Background(gray).
			Foreground(lipgloss.Color(colBarTxt)).
			Padding(0, 1)

	helpKey = lipgloss.NewStyle().
		Foreground(lime).
		Bold(true)

	helpDim = lipgloss.NewStyle().
		Foreground(dimColor)

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

// amberStyle highlights pending/pairing states.
var amberStyle = lipgloss.NewStyle().Foreground(amber).Bold(true)
