package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

// Ember - a warm terminal palette. One dominant accent (ember orange),
// earth-tone support colors, parchment text on warm charcoal-brown.
// All colors are fixed (non-adaptive) for cross-terminal consistency.
//
// Every style name from the previous theme exists here remapped onto
// Ember: this file is a drop-in replacement and no call site needs to
// change for the build to stay green. New semantic helpers used by the
// layout rework (paneBorder, bubbleIn/out, modeChip, ...) are additive.

const (
	colBg      = "#16130F" // canvas: warm near-black brown
	colSurface = "#221D16" // raised surfaces: bubbles, inputs
	colBorder  = "#3B3327" // unfocused pane borders
	colText    = "#EDE4D3" // parchment text
	colDim     = "#9C8F7B" // secondary text
	colMuted   = "#6B6252" // faint text, rules

	colAccent   = "#E8853D" // ember orange: primary accent, focused border
	colGold     = "#D9A441" // headings, badges, pairing states
	colSage     = "#93A978" // online / success
	colClay     = "#CF6A5D" // errors
	colOnAccent = "#16130F" // text on accent fills

	colBarBg  = "#2E271E" // title + status bar fill
	colSelBg  = "#3A3125" // selected rows / outgoing bubbles
	colDark   = colOnAccent
	colViolet = colAccent // legacy
	colBarTxt = colText
)

var (
	textColor = lipgloss.Color(colText)
	dimColor  = lipgloss.Color(colDim)
	mutedClr  = lipgloss.Color(colMuted)
	accent    = lipgloss.Color(colAccent)
	gold      = lipgloss.Color(colGold)
	sage      = lipgloss.Color(colSage)
	clay      = lipgloss.Color(colClay)
	bg        = lipgloss.Color(colBg)
	surface   = lipgloss.Color(colSurface)
	barBg     = lipgloss.Color(colBarBg)
	selBg     = lipgloss.Color(colSelBg)
	focusCol  = lipgloss.Color(colAccent) // focused pane border
	idleCol   = lipgloss.Color(colBorder) // unfocused pane border
	onAccent  = lipgloss.Color(colOnAccent)
)

// Legacy color names mapped onto Ember so existing call sites compile.
var (
	lime   = accent // was neon lime
	cyan   = gold   // was electric cyan
	red    = clay
	amber  = gold
	gray   = barBg
	violet = accent // legacy
	muted  = mutedClr
)

// Text -------------------------------------------------------------------
var (
	baseText = lipgloss.NewStyle().Foreground(textColor)
	dimStyle = lipgloss.NewStyle().Foreground(dimColor)

	titleStyle = lipgloss.NewStyle(). // wordmark
			Foreground(accent).
			Bold(true)

	bannerStyle = titleStyle.Copy()

	headerStyle = lipgloss.NewStyle(). // pane headers, sections
			Foreground(gold).
			Bold(true)

	flashStyle = lipgloss.NewStyle().Foreground(sage).Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(clay).Bold(true)

	helpKey = lipgloss.NewStyle().Foreground(accent).Bold(true)
	helpDim = lipgloss.NewStyle().Foreground(dimColor)
)

// Bars -------------------------------------------------------------------
var (
	headerBarStyle = lipgloss.NewStyle().
			Background(barBg)

	statusBar = lipgloss.NewStyle().
			Background(barBg).
			Foreground(textColor).
			Padding(0, 1)

	modeChip = lipgloss.NewStyle().
			Background(selBg).
			Foreground(accent).
			Bold(true).
			Padding(0, 1)
)

// Chat panes ---------------------------------------------------------------
var (
	sidebarStyle = lipgloss.NewStyle().
			Background(bg).
			Padding(0, 1)

	chatPaneStyle = lipgloss.NewStyle().
			Background(bg)

	chatItemStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Width(28)

	chatItemSelected = lipgloss.NewStyle().
				Background(accent).
				Foreground(onAccent).
				Bold(true).
				Padding(0, 1).
				Width(28)

	unreadBadge = lipgloss.NewStyle().
			Background(gold).
			Foreground(onAccent).
			Bold(true)

	selectedBadge = unreadBadge.Copy()

	chatTitleStyle = lipgloss.NewStyle().
			Foreground(gold).
			Bold(true).
			Margin(0, 0, 0, 1)

	chatTitleDimStyle = lipgloss.NewStyle().
				Foreground(dimColor).
				Bold(true).
				Margin(0, 0, 0, 1)
)

// Messages -----------------------------------------------------------------
var (
	inSender = lipgloss.NewStyle(). // incoming sender name
			Foreground(sage).
			Bold(true)

	bubbleIn = lipgloss.NewStyle(). // incoming bubble: left side
			Border(lipgloss.RoundedBorder()).
			BorderForeground(idleCol).
			Background(surface).
			Foreground(textColor).
			Padding(0, 1)

	bubbleOut = lipgloss.NewStyle(). // outgoing bubble: right-aligned
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5A4A33")).
			Background(selBg).
			Foreground(textColor).
			Padding(0, 1)

	outMark = lipgloss.NewStyle(). // outgoing side marker
		Foreground(accent).
		Bold(true)

	composerFocusedPrompt = lipgloss.NewStyle().
				Foreground(accent).
				Bold(true)

	sectionStyle = lipgloss.NewStyle(). // small caps section headers
			Foreground(gold).
			Bold(true)

	selectedRow = lipgloss.NewStyle(). // highlighted list row
			Foreground(onAccent).
			Background(accent).
			Bold(true)

	mutedSty = lipgloss.NewStyle().Foreground(mutedClr)

	titleBarStyle = lipgloss.NewStyle(). // slim inverted app bar
			Background(barBg)
)

// amberStyle kept for pairing/pending states.
var amberStyle = lipgloss.NewStyle().Foreground(gold).Bold(true)

// Pane borders: the visual focus indicator. Focused panes glow ember,
// unfocused ones stay quiet.
func paneBorder(focused bool) lipgloss.Style {
	c := idleCol
	if focused {
		c = focusCol
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c)
}

// helpStyles returns Ember-tinted bubbles/help styles.
func helpStyles() help.Styles {
	s := help.New().Styles
	s.ShortKey = helpKey
	s.ShortDesc = helpDim
	s.ShortSeparator = lipgloss.NewStyle().Foreground(mutedClr)
	s.FullKey = helpKey
	s.FullDesc = baseText
	s.FullSeparator = lipgloss.NewStyle().Foreground(mutedClr)
	s.Ellipsis = lipgloss.NewStyle().Foreground(mutedClr)
	return s
}

// helpStyles (method form) lets views style help output from the model.
func (m Model) helpStyles() help.Styles { return helpStyles() }
