package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SaiArnav/ZetaChat/core"
)

// handleKey routes keys by current stage and focus. Every match goes
// through m.km bindings so the help overlay stays in sync by
// construction.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.helpOpen {
		return m.handleHelpKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys().Help) &&
		m.stage != stageAuth && !m.focusTyping():
		m.helpOpen = true
		return m, nil

	case key.Matches(msg, m.keys().Mouse):
		if m.mouseOn {
			m.mouseOn = false
			return m, tea.DisableMouse
		}
		m.mouseOn = true
		return m, tea.EnableMouseCellMotion
	}

	switch m.stage {
	case stageAuth:
		return m.handleAuthKey(msg)
	case stageBooting, stageLoading:
		return m.handleGlobalKey(msg)
	case stageDashboard:
		return m.handleDashboardKey(msg)
	case stageQR:
		return m.handleQRKey(msg)
	case stageError:
		return m.handleErrorKey(msg)
	case stageReady:
		if m.searchMode {
			return m.handleSearchKey(msg)
		}
		switch m.focus {
		case PaneMessages:
			return m.handleViewportKey(msg)
		case PaneComposer:
			next, cmd := m.handleComposerKey(msg)
			if nm, ok := next.(Model); ok {
				// Keep the input box height glued to its content
				// (one line idle, grows as text wraps).
				nm = nm.syncComposer()
				return nm, cmd
			}
			return next, cmd
		default:
			return m.handleSidebarKey(msg)
		}
	}
	return m, nil
}

// focusTyping reports whether a text input currently owns the keyboard.
func (m Model) focusTyping() bool {
	return m.focus == PaneComposer || m.searchMode || m.stage == stageAuth
}

// dashCardAt resolves the platform for the current dashboard selection.
func (m Model) dashCardAt() core.Platform {
	cards := dashCards(m.platOrder)
	if m.dashIdx < 0 || m.dashIdx >= len(cards) {
		return ""
	}
	return cards[m.dashIdx]
}

// handleDashboardKey navigates the platform rail.
func (m Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(dashCards(m.platOrder))
	switch {
	case key.Matches(msg, m.keys().Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys().Down):
		if n > 0 {
			m.dashIdx = (m.dashIdx + 1) % n
		}
	case key.Matches(msg, m.keys().Up):
		if n > 0 {
			m.dashIdx = (m.dashIdx - 1 + n) % n
		}
	case key.Matches(msg, m.keys().Open):
		if p := m.dashCardAt(); p != "" {
			return m.selectPlatform(p)
		}
	}
	return m, nil
}

// selectPlatform enters the chat view for one platform — or the merged
// unified inbox when p == core.PlatformAll.
func (m Model) selectPlatform(p core.Platform) (tea.Model, tea.Cmd) {
	if p == core.PlatformAll {
		return m.selectInbox()
	}

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
	m.focus = PaneSidebar
	m.viewport.dirty = true
	m.status = "[" + metaName(p) + "] loading chats…"

	return m, tea.Batch(
		loadCachedChatsCmd(m.store),
		loadChatsCmd(a, m.store),
	)
}

// selectInbox opens the unified inbox: chats from every ready platform,
// merged and sorted by recency.
func (m Model) selectInbox() (tea.Model, tea.Cmd) {
	ready := m.readyAdapters()
	if len(ready) == 0 {
		m.flash("No platform is connected yet.", true)
		return m, nil
	}
	if len(ready) == 1 {
		// Single platform: no point in the merged view.
		return m.selectPlatform(ready[0])
	}

	m.activePlat = core.PlatformAll
	m.stage = stageLoading
	m.chats = nil
	m.messages = nil
	m.chatIdx = 0
	m.focus = PaneSidebar
	m.viewport.dirty = true
	m.status = "Merging chats from " + strconv.Itoa(len(ready)) + " platforms…"

	return m, tea.Batch(
		loadCachedChatsCmd(m.store),
		loadAllChatsCmd(m),
	)
}

// handleGlobalKey: quit keys everywhere; esc opens the dashboard early.
func (m Model) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys().Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys().EscBack):
		if len(m.platOrder) > 0 {
			m.stage = stageDashboard
		}
		return m, nil
	}
	return m, nil
}

// handleQRKey lets the user bail back to the dashboard mid-pairing.
func (m Model) handleQRKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys().Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys().EscBack):
		m.stage = stageDashboard
		return m, nil
	}
	return m, nil
}

// handleErrorKey recovers from an app-level error screen.
func (m Model) handleErrorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys().Quit):
		m.quitting = true
		return m, tea.Quit
	case msg.String() == "enter", key.Matches(msg, m.keys().EscBack):
		m.openDashboard()
		return m, nil
	}
	return m, nil
}

// handleHelpKey dismisses the help overlay.
func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "?", "esc", "q":
		m.helpOpen = false
		return m, nil
	}
	return m, nil
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys().Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys().EscBack):
		dash := m
		dash.openDashboard()
		return dash, nil
	case key.Matches(msg, m.keys().Down):
		if len(m.chats) > 0 {
			m.chatIdx = (m.chatIdx + 1) % len(m.chats)
			m = m.markRead(m.chats[m.chatIdx].ID)
			m.status = "opening " + m.chats[m.chatIdx].Name + "…"
			return m, openChatCmd(m)
		}
	case key.Matches(msg, m.keys().Up):
		if len(m.chats) > 0 {
			m.chatIdx = (m.chatIdx - 1 + len(m.chats)) % len(m.chats)
			m = m.markRead(m.chats[m.chatIdx].ID)
			m.status = "opening " + m.chats[m.chatIdx].Name + "…"
			return m, openChatCmd(m)
		}
	case msg.String() == "g":
		if len(m.chats) > 0 {
			m.chatIdx = 0
			m = m.markRead(m.chats[0].ID)
			m.status = "opening " + m.chats[0].Name + "…"
			return m, openChatCmd(m)
		}
	case msg.String() == "G":
		if len(m.chats) > 0 {
			last := len(m.chats) - 1
			m.chatIdx = last
			m = m.markRead(m.chats[last].ID)
			m.status = "opening " + m.chats[last].Name + "…"
			return m, openChatCmd(m)
		}
	case key.Matches(msg, m.keys().Open),
		key.Matches(msg, m.keys().FocusCycle):
		// Enter or Tab enters the highlighted chat: load it and put
		// focus on the message pane so j/k scroll immediately.
		if len(m.chats) > 0 {
			m.focus = PaneMessages
			m = m.markRead(m.chats[m.chatIdx].ID)
			m.status = "opening " + m.chats[m.chatIdx].Name + "…"
			return m, openChatCmd(m)
		}
	case key.Matches(msg, m.keys().Search):
		m.searchMode = true
		m.search.Focus()
		m.search.SetValue("")
	case key.Matches(msg, m.keys().FocusMsgs):
		if len(m.chats) > 0 {
			m.focus = PaneMessages
		}
	case key.Matches(msg, m.keys().Refresh):
		if m.activePlat == core.PlatformAll {
			return m, loadAllChatsCmd(m)
		}
		return m, loadChatsCmd(m.activeAdapter(), m.store)
	}
	return m, nil
}

func (m Model) handleViewportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys().FocusCycle):
		m.focus = PaneComposer
		m.composer.Focus()
	case key.Matches(msg, m.keys().CloseOrBack), msg.String() == "s":
		m.focus = PaneSidebar
	case key.Matches(msg, m.keys().Down):
		m.viewport.vp.LineDown(1)
	case key.Matches(msg, m.keys().Up):
		m.viewport.vp.LineUp(1)
	case msg.String() == "g":
		m.viewport.vp.GotoTop()
	case msg.String() == "G":
		m.viewport.vp.GotoBottom()
	case msg.String() == "ctrl+d":
		m.viewport.vp.HalfViewDown()
	case msg.String() == "ctrl+u":
		m.viewport.vp.HalfViewUp()
	case msg.String() == "pgdown":
		m.viewport.vp.PageDown()
	case msg.String() == "pgup":
		m.viewport.vp.PageUp()
	}
	return m, nil
}

func (m Model) handleComposerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys().Send):
		// Opt-in confirmation gate: first enter arms the bar,
		// second enter confirms.
		if m.cfg != nil && m.cfg.ConfirmSend && !m.confirming &&
			m.composer.Value() != "" {
			m.confirming = true
			return m, nil
		}
		m.confirming = false

		text := strings.TrimSpace(m.composer.Value())
		if len(m.chats) > 0 && text != "" {
			chat := m.chats[m.chatIdx]
			chatID := chat.ID
			m.composer.Reset()

			m.messages = append(m.messages, core.Message{
				ID:        fmt.Sprintf("local-%d", time.Now().UnixNano()),
				Platform:  chat.Platform,
				ChatID:    chatID,
				Text:      text,
				Timestamp: time.Now(),
				Out:       true,
			})
			m = m.refreshMessagesView().followLatest()

			return m, sendCmd(m.adapterForChat(chat), chatID, text)
		}

	case key.Matches(msg, m.keys().Newline):
		// Insert a newline instead of sending.
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(
			tea.KeyMsg{Type: tea.KeyEnter, Alt: true},
		)
		return m, cmd

	case key.Matches(msg, m.keys().FocusCycle), key.Matches(msg, m.keys().EscBack):
		m.confirming = false
		m.focus = PaneSidebar
		m.composer.Blur()
		return m, nil

	default:
		if m.confirming {
			// Any edit disarms the confirmation.
			m.confirming = false
		}
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys().CloseOrBack):
		m.searchMode = false
		m.search.Blur()
		m.searchResults = nil
		m.searchIdx = 0
		m.search.SetValue("")
		return m, nil
	case key.Matches(msg, m.keys().Open):
		if len(m.searchResults) == 0 {
			q := m.search.Value()
			if q != "" {
				if m.activePlat == core.PlatformAll {
					return m, searchAllCmd(m, q)
				}
				return m, searchCmd(m.activeAdapter(), q)
			}
			return m, nil
		}
		// Open the selected result's chat.
		res := m.searchResults[m.searchIdx]
		return m.openChatFromSearch(res)
	case key.Matches(msg, m.keys().Down):
		if len(m.searchResults) > 0 {
			m.searchIdx = (m.searchIdx + 1) % len(m.searchResults)
		}
	case key.Matches(msg, m.keys().Up):
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
				ID:       res.ChatID,
				Name:     res.Sender.DisplayName,
				Platform: res.Platform,
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
