package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/SaiArnav/ZetaChat/core"
)

// Layout constants.
const (
	sidebarWidth = 30
	statusHeight = 1
	titleHeight  = 1
	composerH    = 2
)

type messagesViewport struct {
	vp     viewport.Model
	chatID string
	dirty  bool
}

func newMessagesViewport() messagesViewport {
	return messagesViewport{vp: viewport.New(0, 0), dirty: true}
}

// chatPaneWidth/Height compute the message viewport size from window size.
func (m Model) chatPaneWidth() int {
	return m.width - sidebarWidth - 2
}

func (m Model) chatPaneHeight() int {
	return m.height - statusHeight
}

func (m Model) currentChatID() string {
	if m.chatIdx < 0 || m.chatIdx >= len(m.chats) {
		return ""
	}
	return m.chats[m.chatIdx].ID
}

// refreshMessagesView re-renders the viewport when the chat or messages change.
// It re-renders on every call so new messages appear immediately, but only
// jumps to the bottom when the chat was just opened or the user was already
// following the conversation (so scrolling up to read history isn't disturbed).
func (m Model) refreshMessagesView() Model {
	if m.stage != stageReady {
		return m
	}
	v := m.viewport
	v.vp.Width = m.chatPaneWidth()
	v.vp.Height = m.chatPaneHeight() - titleHeight - composerH

	cur := m.currentChatID()
	switched := v.chatID != cur || v.dirty
	if switched {
		v.chatID = cur
		v.dirty = false
	}
	v.vp.SetContent(m.renderMessages(m.messages))
	if switched || v.vp.AtBottom() {
		v.vp.GotoBottom()
	}
	m.viewport = v
	return m
}

// renderMessages renders a message list as viewport content.
func (m Model) renderMessages(msgs []core.Message) string {
	if len(msgs) == 0 {
		return dimStyle.Render("No messages yet.")
	}
	var b strings.Builder
	for i, msg := range msgs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(m.renderMessage(msg))
	}
	return b.String()
}

func (m Model) renderMessage(msg core.Message) string {
	ts := msg.Timestamp.Format("15:04")
	sender := msg.Sender.DisplayName
	if sender == "" {
		sender = "?"
	}
	text := strings.ReplaceAll(msg.Text, "\n", "\n    ")

	if msg.Out {
		head := dimStyle.Render("  " + ts + "   you")
		body := lipgloss.NewStyle().Foreground(textColor).Render("❯ " + text)
		return head + "\n" + body
	}
	head := lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true).
		Render(sender) + "  " + dimStyle.Render(ts)
	return head + "\n" + baseText.Render(text)
}

// chatTitle renders the header line of the chat pane.
func (m Model) chatTitle() string {
	cur := m.currentChatID()
	name := ""
	for _, c := range m.chats {
		if c.ID == cur {
			name = c.Name
			break
		}
	}
	if name == "" {
		name = cur
	}
	left := chatTitleStyle.Render("❯ " + name)
	if m.cached {
		left += dimStyle.Render("  (cached)")
	}
	if m.searchMode {
		left = searchHeader
	}
	right := ""
	if m.focusComposer {
		right = dimStyle.Render("[composer]")
	}
	width := m.chatPaneWidth()
	line := left
	if right != "" {
		pad := width - lipgloss.Width(left) - lipgloss.Width(right)
		if pad < 1 {
			pad = 1
		}
		line += strings.Repeat(" ", pad) + right
	}
	return line
}

var searchHeader = headerStyle.Render("Search")

// sidebar renders the chat list pane.
func (m Model) sidebarView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("CHATS"))
	b.WriteString("\n\n")

	if len(m.chats) == 0 {
		b.WriteString(dimStyle.Render("no chats yet"))
		return sidebarStyle.Render(b.String())
	}

	for i, c := range m.chats {
		name := c.Name
		if name == "" {
			name = c.ID
		}
		unread := c.UnreadCount
		selected := i == m.chatIdx

		title := truncateRight(name, 24)
		badge := ""
		if unread > 0 {
			if selected {
				badge = selectedBadge.Render(" " + strconv.Itoa(unread) + " ")
			} else {
				badge = unreadBadge.Render(" " + strconv.Itoa(unread) + " ")
			}
		}

		style := chatItemStyle
		if selected {
			style = chatItemSelected
		}
		line := style.Render(title)
		if badge != "" {
			pad := style.GetWidth() - lipgloss.Width(title) - lipgloss.Width(badge)
			if pad < 1 {
				pad = 1
			}
			line = strings.TrimRight(line, " ") + strings.Repeat(" ", pad) + badge
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Pad to full pane height so selection bar reaches the bottom.
	h := m.chatPaneHeight() - 3
	rendered := sidebarStyle.Render(b.String())
	pad := h - lipgloss.Height(rendered)
	if pad > 0 {
		rendered += "\n" + strings.Repeat("\n", pad-1)
	}
	return rendered
}

func truncateRight(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
