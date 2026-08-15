package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SaiArnav/ZetaChat/adapters/telegram"
	"github.com/SaiArnav/ZetaChat/storage"
)

// loadCachedChatsCmd returns the locally cached chat list (fast first paint).
func loadCachedChatsCmd(store *storage.Store) tea.Cmd {
	return func() tea.Msg {
		chats, err := store.CachedChats()
		if err != nil {
			return cachedChatsMsg{}
		}
		return cachedChatsMsg{chats: chats}
	}
}

// loadChatsCmd fetches the live chat list from the network.
func loadChatsCmd(a *telegram.Adapter, store *storage.Store) tea.Cmd {
	return func() tea.Msg {
		chats, err := a.Chats()
		if err == nil && store != nil {
			_ = store.SaveChats(chats)
		}
		return chatsLoadedMsg{chats: chats, err: err}
	}
}

// openChatCmd loads the currently selected chat's messages from the network.
func openChatCmd(m Model) tea.Cmd {
	chatID := m.chats[m.chatIdx].ID
	a, s := m.adapter, m.store
	return func() tea.Msg {
		msgs, err := a.Messages(chatID)
		if err == nil {
			_ = s.SaveMessages(msgs)
		}
		return messagesLoadedMsg{chatID: chatID, msgs: msgs, err: err}
	}
}

// openCachedChatCmd loads the selected chat's cached messages immediately.
func openCachedChatCmd(m Model) tea.Cmd {
	chatID := m.chats[m.chatIdx].ID
	return func() tea.Msg {
		msgs, err := m.store.CachedMessages(chatID)
		if err != nil {
			msgs = nil
		}
		return cachedMessagesMsg{chatID: chatID, msgs: msgs}
	}
}

// sendCmd delivers a message through the adapter.
func sendCmd(a *telegram.Adapter, chatID, text string) tea.Cmd {
	return func() tea.Msg {
		err := a.SendMessage(chatID, text)
		return sendResultMsg{chatID: chatID, text: text, err: err}
	}
}

// searchCmd runs a global search.
func searchCmd(a *telegram.Adapter, q string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := a.Search(q)
		return searchResultMsg{query: q, msgs: msgs, err: err}
	}
}
