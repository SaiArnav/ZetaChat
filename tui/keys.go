package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SaiArnav/ZetaChat/core"
)

// handleKey routes keys by current stage and focus.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.stage {
	case stageAuth:
		return m.handleAuthKey(msg)
	case stageBooting, stageLoading:
		return m.handleGlobalKey(msg)
	case stageDashboard:
		return m.handleDashboardKey(msg)
	case stageQR:
		return m.handleQRKey(msg)
	case stageReady:
		if m.searchMode {
			return m.handleSearchKey(msg)
		}
		if m.focusViewport {
			return m.handleViewportKey(msg)
		}
		if m.focusComposer {
			return m.handleComposerKey(msg)
		}
		return m.handleSidebarKey(msg)
	}
	return m, nil
}

// handleDashboardKey navigates the platform cards.
func (m Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "down", "j", "right", "l", "tab":
		if len(m.platOrder) > 0 {
			m.dashIdx = (m.dashIdx + 1) % len(m.platOrder)
		}
	case "up", "k", "left", "h", "shift+tab":
		if len(m.platOrder) > 0 {
			m.dashIdx = (m.dashIdx - 1 + len(m.platOrder)) % len(m.platOrder)
		}
	case "enter":
		if m.dashIdx >= 0 && m.dashIdx < len(m.platOrder) {
			p := m.platOrder[m.dashIdx]
			return m.selectPlatform(p)
		}
	}
	return m, nil
}

// selectPlatform enters the chat view for one platform.
func (m Model) selectPlatform(p core.Platform) (tea.Model, tea.Cmd) {
	st := m.platState[p]
	a := m.adapters[p]

	if st == nil || a == nil {
		return m, nil
	}

	if !st.ready {
		// Not linked yet: show the pairing screen (QR or waiting spinner).
		m.activePlat = p
		m.stage = stageQR
		return m, nil
	}

	if st.err != nil {
		m.flash("["+metaName(p)+"] "+st.err.Error(), true)
		return m, nil
	}

	m.activePlat = p
	m.stage = stageLoading
	m.chats = nil
	m.messages = nil
	m.chatIdx = 0
	m.viewport.dirty = true
	m.status = "[" + metaName(p) + "] loading chats…"

	return m, tea.Batch(
		loadCachedChatsCmd(m.store),
		loadChatsCmd(a, m.store),
	)
}

// handleGlobalKey: quit keys everywhere; esc opens the dashboard early.
func (m Model) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		if len(m.platOrder) > 0 {
			m.stage = stageDashboard
		}
		return m, nil
	}
	return m, nil
}

// handleQRKey lets the user bail back to the dashboard mid-pairing.
func (m Model) handleQRKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.stage = stageDashboard
		return m, nil
	}
	return m, nil
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		dash := m
		dash.openDashboard()
		return dash, nil
	case "down", "j":
		if len(m.chats) > 0 {
			m.chatIdx = (m.chatIdx + 1) % len(m.chats)
			return m, openChatCmd(m)
		}
	case "up", "k":
		if len(m.chats) > 0 {
			m.chatIdx = (m.chatIdx - 1 + len(m.chats)) % len(m.chats)
			return m, openChatCmd(m)
		}
	case "g":
		if len(m.chats) > 0 {
			m.chatIdx = 0
			return m, openChatCmd(m)
		}
	case "G":
		if len(m.chats) > 0 {
			m.chatIdx = len(m.chats) - 1
			return m, openChatCmd(m)
		}
	case "enter":
		return m, openChatCmd(m)
	case "tab":
		m.focusComposer = true
		m.composer.Focus()
	case "/":
		m.searchMode = true
		m.search.Focus()
		m.search.SetValue("")
	case "s":
		if len(m.chats) > 0 {
			m.focusViewport = true
		}
	case "r":
		return m, loadChatsCmd(m.activeAdapter(), m.store)
	}
	return m, nil
}

func (m Model) handleViewportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc", "s":
		m.focusViewport = false
		return m, nil
	case "j", "down":
		m.viewport.vp.LineDown(1)
	case "k", "up":
		m.viewport.vp.LineUp(1)
	case "g":
		m.viewport.vp.GotoTop()
	case "G":
		m.viewport.vp.GotoBottom()
	case "ctrl+d":
		m.viewport.vp.HalfViewDown()
	case "ctrl+u":
		m.viewport.vp.HalfViewUp()
	case "pgdown":
		m.viewport.vp.PageDown()
	case "pgup":
		m.viewport.vp.PageUp()
	}
	return m, nil
}

func (m Model) handleComposerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		text := m.composer.Value()
		if len(m.chats) > 0 && text != "" {
			chatID := m.currentChatID()
			m.composer.SetValue("")

			m.messages = append(m.messages, core.Message{
				ID:        fmt.Sprintf("local-%d", time.Now().UnixNano()),
				Platform:  core.PlatformTelegram,
				ChatID:    chatID,
				Text:      text,
				Timestamp: time.Now(),
				Out:       true,
			})
			m = m.refreshMessagesView()

			return m, sendCmd(m.activeAdapter(), chatID, text)
		}
	case "esc", "tab":
		m.focusComposer = false
		m.composer.Blur()
		return m, nil
	default:
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.searchMode = false
		m.search.Blur()
		m.searchResults = nil
		m.searchIdx = 0
		m.search.SetValue("")
		return m, nil
	case "enter":
		if len(m.searchResults) == 0 {
			q := m.search.Value()
			if q != "" {
				return m, searchCmd(m.activeAdapter(), q)
			}
			return m, nil
		}
		// Open the selected result's chat.
		res := m.searchResults[m.searchIdx]
		return m.openChatFromSearch(res)
	case "down", "j":
		if len(m.searchResults) > 0 {
			m.searchIdx = (m.searchIdx + 1) % len(m.searchResults)
		}
	case "up", "k":
		if len(m.searchResults) > 0 {
			m.searchIdx = (m.searchIdx - 1 + len(m.searchResults)) % len(m.searchResults)
		}
	default:
		if len(m.searchResults) > 0 {
			// Editing the query clears stale results.
			m.searchResults = nil
			m.searchIdx = 0
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}
	return m, nil
}

// openChatFromSearch opens the chat a search result belongs to.
func (m Model) openChatFromSearch(res core.Message) (tea.Model, tea.Cmd) {
	cur := m.currentChatID()
	if res.ChatID != cur {
		found := false
		for i, c := range m.chats {
			if c.ID == res.ChatID {
				m.chatIdx = i
				found = true
				break
			}
		}
		if !found {
			m.chats = append(m.chats, core.Chat{
				ID:   res.ChatID,
				Name: res.Sender.DisplayName,
			})
			m.chatIdx = len(m.chats) - 1
		}
	}
	m.searchMode = false
	m.search.Blur()
	m.searchResults = nil
	m.searchIdx = 0
	m.search.SetValue("")
	m.status = "Opened chat from search"
	return m, tea.Batch(openChatCmd(m), refreshUnread(m))
}
