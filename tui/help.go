package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// keyMap is the single source of truth for every keybinding in the app.
// Update() dispatches through these bindings (key.Matches) and the help
// overlay renders from the same struct, so the two can never drift.
type keyMap struct {
	Quit        key.Binding
	Help        key.Binding
	Mouse       key.Binding
	EscBack     key.Binding
	Up          key.Binding
	Down        key.Binding
	TopBottom   key.Binding
	HalfPage    key.Binding
	Page        key.Binding
	Open        key.Binding
	Refresh     key.Binding
	Search      key.Binding
	FocusCycle  key.Binding
	FocusMsgs   key.Binding
	Send        key.Binding
	Newline     key.Binding
	CloseOrBack key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Mouse: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "toggle mouse (off = select/copy text)"),
		),
		EscBack: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back / dashboard"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k", "shift+tab", "h"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j", "l"),
			key.WithHelp("↓/j", "down"),
		),
		TopBottom: key.NewBinding(
			key.WithKeys("g", "G"),
			key.WithHelp("g/G", "first/last"),
		),
		HalfPage: key.NewBinding(
			key.WithKeys("ctrl+d", "ctrl+u"),
			key.WithHelp("ctrl+d/u", "half page"),
		),
		Page: key.NewBinding(
			key.WithKeys("pgup", "pgdown"),
			key.WithHelp("pgup/dn", "page"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open/select"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh chats"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search messages"),
		),
		FocusCycle: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "enter chat / next pane"),
		),
		FocusMsgs: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scroll messages"),
		),
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		Newline: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+enter", "new line"),
		),
		CloseOrBack: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
	}
}

// ShortHelp implements help.KeyMap (status-bar strip).
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Mouse, k.EscBack, k.Quit}
}

// FullHelp implements help.KeyMap (overlay columns).
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help, k.Mouse, k.EscBack, k.Quit},                   // global
		{k.Up, k.Down, k.Open, k.Refresh, k.Search},            // navigate
		{k.FocusCycle, k.FocusMsgs, k.TopBottom},               // panes+scroll
		{k.HalfPage, k.Page, k.Send, k.Newline, k.CloseOrBack}, // scroll/compose
	}
}

// shortHelpFor picks the context strip shown in the status bar right side.
func shortHelpFor(m Model) []key.Binding {
	k := m.keys()
	switch m.stage {
	case stageDashboard:
		return []key.Binding{k.Up, k.Down, k.Open, k.Help, k.Mouse, k.Quit}
	case stageQR, stageError, stageBooting, stageLoading:
		return []key.Binding{k.EscBack, k.Help, k.Quit}
	case stageReady:
		if m.searchMode {
			return []key.Binding{k.Open, k.Up, k.Down, k.CloseOrBack, k.Help}
		}
		if m.confirming {
			return []key.Binding{k.Send, k.CloseOrBack, k.Help}
		}
		switch m.focus {
		case PaneMessages:
			return []key.Binding{k.Up, k.Down, k.FocusCycle,
				k.TopBottom, k.Page, k.CloseOrBack, k.Help}
		case PaneComposer:
			return []key.Binding{k.Send, k.Newline, k.CloseOrBack,
				k.Mouse, k.Help}
		default:
			return []key.Binding{k.Up, k.Down, k.Open, k.FocusCycle,
				k.FocusMsgs, k.Search, k.Refresh, k.Help}
		}
	}
	return k.ShortHelp()
}

// helpModel is the bubbles renderer used by both bar and overlay.
func helpModel() help.Model {
	return help.New()
}

// renderHelpOverlay draws the full-help panel centered on screen. It is
// generated entirely from keyMap.FullHelp — no second binding list exists.
func renderHelpOverlay(m Model) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("KEYBINDINGS"))
	b.WriteString("\n\n")
	b.WriteString(m.helpStyles().FullSeparator.Render(""))
	body := m.help.FullHelpView(m.keys().FullHelp())

	panel := paneBorder(true).Padding(1, 3).Render(
		sectionStyle.Render("KEYBINDINGS") + "\n\n" + body + "\n\n" +
			dimStyle.Render(m.help.ShortHelpView(shortHelpFor(m))),
	)

	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, panel)
}
