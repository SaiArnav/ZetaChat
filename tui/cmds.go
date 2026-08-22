package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SaiArnav/ZetaChat/core"
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
func loadChatsCmd(a core.LiveMessenger, store *storage.Store) tea.Cmd {
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
	a, s := m.activeAdapter(), m.store
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
	store := m.store
	return func() tea.Msg {
		msgs, err := store.CachedMessages(chatID)
		if err != nil {
			msgs = nil
		}
		return cachedMessagesMsg{chatID: chatID, msgs: msgs}
	}
}

// sendCmd delivers a message through the active platform's adapter.
func sendCmd(a core.LiveMessenger, chatID, text string) tea.Cmd {
	return func() tea.Msg {
		err := a.SendMessage(chatID, text)
		return sendResultMsg{chatID: chatID, text: text, err: err}
	}
}

// searchCmd runs a search on the active platform.
func searchCmd(a core.LiveMessenger, q string) tea.Cmd {
	return func() tea.Msg {
		msgs, err := a.Search(q)
		return searchResultMsg{query: q, msgs: msgs, err: err}
	}
}

// retryConnectCmd reconnects one platform.
func retryConnectCmd(p core.Platform, a core.LiveMessenger) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-a.Ready():
		default:
			return connMsg{plat: p} // still connecting
		}
		if err := a.AuthErr(); err != nil {
			return connMsg{plat: p, err: err}
		}
		return connMsg{plat: p}
	}
}

// qrMsg carries a pairing code that should be rendered as a QR code.
type qrMsg struct {
	plat core.Platform
	code string
}

// watchQRCmd relays streamed QR codes from an adapter into the UI.
func watchQRCmd(p core.Platform, a core.LiveMessenger) tea.Cmd {
	qs, ok := a.(core.QRStreamer)
	if !ok {
		return nil
	}
	return func() tea.Msg {
		code, ok := <-qs.QRCodes()
		if !ok {
			return nil
		}
		return qrMsg{plat: p, code: code}
	}
}
