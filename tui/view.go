package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the whole UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var out string

	switch m.stage {
	case stageBooting, stageLoading:
		out = m.splashView(
			m.spinner.View() + "  " + baseText.Render(m.status),
		)

	case stageAuth:
		out = m.authView()

	case stageQR:
		out = m.qrView()

	case stageDashboard:
		out = m.dashboardView()

	case stageError:
		out = m.errorView()

	case stageReady:
		out = m.mainView()
	}

	if m.helpOpen {
		return renderHelpOverlay(m)
	}
	return out
}

func (m Model) splashView(content string) string {
	// Slim boot screen: a quiet wordmark instead of the old ASCII
	// banner wall, so launching lands you in the app immediately.
	head := titleStyle.Render("ZETACHAT") +
		dimStyle.Render("  //  multi-platform relay")
	hint := dimStyle.Render("esc dashboard · ? help · M mouse · ctrl+c quit")
	return m.centerView(head + "\n\n" + content + "\n\n" + hint)
}

func (m Model) centerView(content string) string {
	w, h := m.width, m.height

	if w < 1 {
		w = 80
	}

	if h < 1 {
		h = 24
	}

	return lipgloss.Place(
		w,
		h,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// mainView composes the chat workspace: a slim titlebar, the bordered
// sidebar + message pane + composer box in the middle, and the status
// bar at the bottom.
func (m Model) mainView() string {
	titlebar := m.titleBar()

	sidebar := m.sidebarView()
	messages := m.messagesPane()
	composer := m.composerBox()

	chatCol := lipgloss.JoinVertical(lipgloss.Left, messages, composer)

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		chatCol,
	)

	return titlebar + body + m.statusView()
}

// titleBar is the slim inverted app bar.
func (m Model) titleBar() string {
	left := titleBarStyle.Render(titleStyle.Render(" ZETACHAT")) +
		titleBarStyle.Render(dimStyle.Render("  //  "+metaName(m.activePlat)))

	right := ""
	if m.selfName != "" {
		right = titleBarStyle.Render(dimStyle.Render(m.selfName + " "))
	}

	width := m.width
	pad := width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 0 {
		pad = 0
	}

	return headerBarStyle.Render(left+strings.Repeat(" ", pad)+right) + "\n"
}

// messagesPane wraps viewport/search content in a focus-bordered pane
// with its own header strip.
func (m Model) messagesPane() string {
	var content string

	if m.searchMode {
		results := truncateLines(
			m.searchResultsView(),
			m.messageVPHeight(),
		)
		content = scrollPane(results, m.chatPaneWidth(), m.messageVPHeight())
	} else {
		content = m.viewport.vp.View()
	}

	inner := m.paneHeader() + "\n" + content

	box := paneBorder(m.focus == PaneMessages).
		Width(m.width - m.sidebarWidth).
		Render(inner)

	// Exact target height: header + viewport + two borders.
	target := m.messageVPHeight() + 3
	for lipgloss.Height(box) < target {
		box += "\n"
	}
	return box
}

// composerBox draws the input inside its own bordered pane; focused =
// ember border, unfocused = quiet.
func (m Model) composerBox() string {
	var inner string
	switch {
	case m.confirming:
		inner = flashStyle.Render("send? ") +
			baseText.Render(truncateRight(m.composer.Value(), 40))
	case m.focus == PaneComposer:
		inner = m.composer.View()
	default:
		inner = dimStyle.Render("❯ press tab to type")
	}

	box := paneBorder(m.focus == PaneComposer || m.confirming).
		Width(m.width-m.sidebarWidth).
		Padding(0, 0).
		Render(inner)

	// The composer box height is fixed: content + borders.
	target := m.composerHeight() + 2
	for lipgloss.Height(box) < target {
		box += "\n"
	}
	return box
}

// errorView renders the recoverable app-error screen (§2.5).
func (m Model) errorView() string {
	body := errorStyle.Render("Something broke") +
		"\n\n" + baseText.Render(m.fatal) +
		"\n\n" + dimStyle.Render("esc back to dashboard · ctrl+c quit")
	return m.centerView(body)
}

// padToHeight pads a block with trailing blank lines to exactly h rows.
func padToHeight(s string, h int) string {
	gap := h - lipgloss.Height(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat("\n", gap)
}

// scrollPane clips rendered content to the pane dimensions.
func scrollPane(content string, w, h int) string {
	if w < 1 {
		w = 1
	}

	if h < 1 {
		h = 1
	}

	lines := strings.Split(
		strings.TrimRight(content, "\n"),
		"\n",
	)

	if len(lines) > h {
		lines = lines[len(lines)-h:]
	}

	for i, ln := range lines {
		lines[i] = truncateRight(ln, w)
	}

	return strings.Join(lines, "\n")
}

func truncateLines(s string, n int) string {
	if n < 1 {
		return ""
	}

	lines := strings.Split(s, "\n")

	if len(lines) > n {
		lines = lines[:n]
	}

	return strings.Join(lines, "\n")
}
