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
		return m.splashView(m.spinner.View() + "  " + baseText.Render(m.status))
	case stageAuth:
		return m.authView()
	case stageReady:
		return m.mainView()
	}
	return ""
}

func (m Model) splashView(content string) string {
	header := bannerStyle.Render(banner())
	return m.centerView(header + "\n\n\n" + content)
}

func (m Model) centerView(content string) string {
	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) mainView() string {
	header := titleStyle.Render("ZETACHAT")
	if m.selfName != "" {
		header += dimStyle.Render("   " + m.selfName)
	}
	header += "\n" + strings.Repeat("─", m.width)

	sidebar := m.sidebarView()
	chat := m.chatPaneView()

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chat)
	return header + "\n" + body + "\n" + m.statusView()
}

// chatPaneView renders the right-hand pane: title, content, composer.
func (m Model) chatPaneView() string {
	title := m.chatTitle()
	composer := m.composerView()

	var content string
	vw := m.chatPaneWidth()
	vh := m.chatPaneHeight() - titleHeight - composerH

	if m.searchMode {
		content = m.search.View() + "\n\n" + truncateLines(m.searchResultsView(), vh-3)
		content = scrollPane(content, vw, vh)
	} else {
		content = m.viewport.vp.View()
	}

	return lipgloss.JoinVertical(lipgloss.Top, title, content, composer)
}

func (m Model) composerView() string {
	if m.focusComposer {
		return m.composer.View()
	}
	return dimStyle.Render("❯ ") + dimStyle.Render("press tab to type")
}

// scrollPane clips rendered content to the pane dimensions.
func scrollPane(content string, w, h int) string {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) > h {
		lines = lines[len(lines)-h:]
	}
	for i, ln := range lines {
		lines[i] = truncateRight(ln, w)
	}
	return strings.Join(lines, "\n")
}

func truncateLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
