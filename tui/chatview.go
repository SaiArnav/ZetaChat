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
	w := m.width - m.sidebarWidth
	if w < 1 {
		return 1
	}
	return w
}

func (m Model) chatPaneHeight() int {
	// The main view contains:
	//   2 lines for the header
	//   chat body
	//   1 line for the status bar
	//
	// Therefore the body gets total height - 3.
	h := m.height - statusHeight - 2
	if h < 1 {
		return 1
	}
	return h
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

	height := m.chatPaneHeight() - titleHeight - composerH
	if height < 1 {
		height = 1
	}
	v.vp.Height = height

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
		body := lipgloss.NewStyle().
			Foreground(textColor).
			Render("❯ " + text)

		return head + "\n" + body
	}

	head := lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true).
		Render(sender) +
		"  " +
		dimStyle.Render(ts)

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
		return sidebarStyle.Width(m.sidebarWidth).Render(b.String())
	}

	itemWidth := m.sidebarWidth - 4
	if itemWidth < 10 {
		itemWidth = 10
	}

	for i, c := range m.chats {
		name := c.Name
		if name == "" {
			name = c.ID
		}

		unread := c.UnreadCount
		selected := i == m.chatIdx

		title := truncateRight(name, itemWidth-2)

		badge := ""

		if unread > 0 {
			if selected {
				badge = selectedBadge.Render(" " + strconv.Itoa(unread) + " ")
			} else {
				badge = unreadBadge.Render(" " + strconv.Itoa(unread) + " ")
			}
		}

		style := lipgloss.NewStyle().
			Padding(0, 1).
			Width(itemWidth)

		if selected {
			style = lipgloss.NewStyle().
				Background(lime).
				Foreground(lipgloss.Color(colDark)).
				Bold(true).
				Padding(0, 1).
				Width(itemWidth)
		}

		line := style.Render(title)

		if badge != "" {
			pad := style.GetWidth() -
				lipgloss.Width(title) -
				lipgloss.Width(badge)

			if pad < 1 {
				pad = 1
			}

			line = strings.TrimRight(line, " ") +
				strings.Repeat(" ", pad) +
				badge
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Pad to full pane height so selection bar reaches the bottom.
	h := m.chatPaneHeight() - 3

	rendered := sidebarStyle.Width(m.sidebarWidth).Render(b.String())

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

	if n <= 1 {
		return "…"
	}

	return string(runes[:n-1]) + "…"
}
