package tui

import (
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
	case stageReady:
		if m.searchMode {
			return m.handleSearchKey(msg)
		}
		if m.focusComposer {
			return m.handleComposerKey(msg)
		}
		return m.handleSidebarKey(msg)
	}
	return m, nil
}

// handleGlobalKey: quit keys that work everywhere.
func (m Model) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit
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
	case "r":
		return m, loadChatsCmd(m.adapter, m.store)
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
			return m, sendCmd(m.adapter, chatID, text)
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
				return m, searchCmd(m.adapter, q)
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
