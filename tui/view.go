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

	switch m.stage {
	case stageBooting, stageLoading:
		return m.splashView(
			m.spinner.View() + "  " + baseText.Render(m.status),
		)

	case stageAuth:
		return m.authView()

	case stageQR:
		return m.qrView()

	case stageDashboard:
		return m.dashboardView()

	case stageReady:
		return m.mainView()
	}

	return ""
}

func (m Model) splashView(content string) string {
	header := bannerStyle.Render(banner())
	hint := dimStyle.Render("esc dashboard · ctrl+c quit")
	return m.centerView(header + "\n\n\n" + content + "\n\n" + hint)
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

func (m Model) mainView() string {
	header := headerBarStyle.Render(
		titleStyle.Render("ZETACHAT"),
	)
	if m.selfName != "" {
		header += dimStyle.Render("   " + m.selfName)
	}

	// Render the divider with the dim style instead of the terminal's
	// default foreground. This prevents the bright white line.
	header += "\n" + headerBarStyle.Render(
		dimStyle.Render(strings.Repeat("─", m.width)),
	)

	sidebar := m.sidebarView()
	chat := m.chatPaneView()

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		chat,
	)

	return header + "\n" + body + "\n" + m.statusView()
}

// chatPaneView renders the right-hand pane: title, content, composer.
func (m Model) chatPaneView() string {
	title := m.chatTitle()
	composer := m.composerView()

	vw := m.chatPaneWidth()

	vh := m.chatPaneHeight() - titleHeight - composerH
	if vh < 1 {
		vh = 1
	}

	var content string

	if m.searchMode {
		searchContent := m.search.View() + "\n\n"

		results := truncateLines(
			m.searchResultsView(),
			vh-3,
		)

		content = scrollPane(
			searchContent+results,
			vw,
			vh,
		)
	} else {
		content = m.viewport.vp.View()
	}

	return chatPaneStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			title,
			content,
			composer,
		),
	)
}

func (m Model) composerView() string {
	if m.focusComposer {
		return m.composer.View()
	}

	return dimStyle.Render("❯ ") +
		dimStyle.Render("press tab to type")
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
