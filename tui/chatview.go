package tui

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/SaiArnav/ZetaChat/core"
)

// Layout constants.
const (
	statusHeight = 1
	titleHeight  = 1

	// minComposerLines..maxComposerLines bound the dynamic composer.
	minComposerLines = 1
	maxComposerLines = 5
)

type messagesViewport struct {
	vp     viewport.Model
	chatID string
	dirty  bool
	// cache memoizes rendered message blocks keyed by message ID so a
	// single new message doesn't re-style the whole history.
	cache map[string]string
}

func newMessagesViewport() messagesViewport {
	return messagesViewport{
		vp:    viewport.New(0, 0),
		dirty: true,
		cache: map[string]string{},
	}
}

// chatPaneWidth is the inner width of the message/composer panes.
func (m Model) chatPaneWidth() int {
	w := m.width - m.sidebarWidth - 2 // inside the pane borders
	if w < 1 {
		return 1
	}
	return w
}

// composerBoxHeight is the total height of the bordered composer box.
func (m Model) composerBoxHeight() int {
	return m.composerHeight() + 2 // borders
}

// composerHeight is the number of visible input rows: one when empty,
// growing as the text wraps or gains lines, capped at maxComposerLines.
func (m Model) composerHeight() int {
	h := m.composer.LineCount()
	if h < minComposerLines {
		h = minComposerLines
	}
	if h > maxComposerLines {
		h = maxComposerLines
	}
	return h
}

// syncComposer keeps the textarea's rendered size in step with its
// content and the current pane width. Call after any resize or edit.
func (m Model) syncComposer() Model {
	w := m.chatPaneWidth() - 4
	if w < 10 {
		w = 10
	}
	m.composer.SetWidth(w)
	m.composer.SetHeight(m.composerHeight())
	return m
}

func (m Model) chatPaneHeight() int {
	// titlebar + panes + status bar.
	h := m.height - statusHeight - titleHeight
	if h < 1 {
		return 1
	}
	return h
}

// messageVPHeight is the viewport height inside the bordered message
// pane, below its one-line header and above the bordered composer box.
func (m Model) messageVPHeight() int {
	h := m.chatPaneHeight() - 2 - // message pane borders
		1 - // pane header line
		(m.composerHeight() + 2) // composer box incl. its borders
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

// refreshMessagesView re-renders the viewport when the chat or messages change,
// reusing cached per-message blocks, and only jumps to the bottom when the chat
// was just opened or the user was already following the conversation.
func (m Model) refreshMessagesView() Model {
	if m.stage != stageReady {
		return m
	}

	v := m.viewport
	v.vp.Width = m.chatPaneWidth()
	v.vp.Height = m.messageVPHeight()

	cur := m.currentChatID()
	switched := v.chatID != cur || v.dirty

	if switched {
		v.chatID = cur
		v.dirty = false
		// Fresh chat: drop the previous chat's rendered blocks.
		v.cache = map[string]string{}
	}

	v.vp.SetContent(renderMessages(m.messages, v.cache, m.chatPaneWidth()-2))

	if switched || v.vp.AtBottom() {
		v.vp.GotoBottom()
	}

	m.viewport = v
	return m
}

// renderMessages renders a message list as bubble blocks, reusing cached
// blocks for messages that haven't changed. width is the drawable area,
// used for wrapping and right-alignment of outgoing bubbles.
func renderMessages(msgs []core.Message, cache map[string]string, width int) string {
	if len(msgs) == 0 {
		return dimStyle.Render("No messages yet.")
	}

	var b strings.Builder

	for i, msg := range msgs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(cachedMessageBlock(msg, cache, width))
	}

	return b.String()
}

// cachedMessageBlock renders one bubble or reuses its cached rendering.
// The cache key includes text, direction and width so edits and
// resizes invalidate naturally.
func cachedMessageBlock(msg core.Message, cache map[string]string, width int) string {
	key := msg.ID + "|" + msg.Text + "|" +
		map[bool]string{true: "o", false: "i"}[msg.Out] +
		"|" + strconv.Itoa(width)
	if s, ok := cache[key]; ok {
		return s
	}
	s := renderBubble(msg, width)
	cache[key] = s
	return s
}

// renderBubble draws one message as a clean two-party exchange.
// Incoming sit left; outgoing are right-aligned. The bubble always
// hugs its content — no filler columns inside the box.
func renderBubble(msg core.Message, width int) string {
	ts := msg.Timestamp.Format("15:04")

	wrapW := width * 3 / 5
	if wrapW < 18 {
		wrapW = 18
	}
	if width > 0 && wrapW > width-6 {
		wrapW = width - 6
	}
	text := wrapWords(msg.Text, wrapW)

	if msg.Out {
		head := dimStyle.Render("you · " + ts)
		body := bubbleOut.Render(text)

		hpad := width - lipgloss.Width(head) - 1
		bpad := width - lipgloss.Width(body) - 1
		if hpad < 0 {
			hpad = 0
		}
		if bpad < 0 {
			bpad = 0
		}
		hind := strings.Repeat(" ", hpad)
		bind := strings.Repeat(" ", bpad)
		// Indent every line of the bordered block, not just the first.
		body = strings.ReplaceAll(body, "\n", "\n"+bind)
		return hind + head + "\n" + bind + body
	}

	sender := msg.Sender.DisplayName
	if sender == "" {
		sender = "?"
	}
	head := inSender.Render(sender) + dimStyle.Render(" · "+ts)
	body := bubbleIn.Render(text)
	return head + "\n" + body
}

// wrapWords soft-wraps text at word boundaries to w columns WITHOUT
// padding short lines — unlike lipgloss Width(), which would force
// every line to full width and stretch the bubble with dead space.
func wrapWords(s string, w int) string {
	if w < 1 {
		return s
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := words[0]
		cw := utf8.RuneCountInString(cur)
		for _, wd := range words[1:] {
			ww := utf8.RuneCountInString(wd)
			if cw+1+ww <= w {
				cur += " " + wd
				cw += 1 + ww
			} else {
				out = append(out, cur)
				cur = wd
				cw = ww
			}
		}
		out = append(out, cur)
	}
	return strings.Join(out, "\n")
}

// paneHeader renders the title strip inside the message pane border.
func (m Model) paneHeader() string {
	name := m.currentChatName()

	left := sectionStyle.Render("❯ " + name)

	if m.activePlat == core.PlatformAll {
		for _, c := range m.chats {
			if c.ID == m.currentChatID() {
				left += "  " + dimStyle.Render(metaName(c.Platform))
				break
			}
		}
	}

	if m.cached {
		left += dimStyle.Render("  (cached)")
	}

	if m.searchMode {
		left = sectionStyle.Render("/ search")
	}

	right := ""
	switch m.focus {
	case PaneComposer:
		right = dimStyle.Render("[composer]")
	case PaneMessages:
		right = flashStyle.Render("[messages]")
	default:
		right = ""
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

	return truncateRight(line, width)
}

func (m Model) currentChatName() string {
	cur := m.currentChatID()
	for _, c := range m.chats {
		if c.ID == cur {
			return c.Name
		}
	}
	if cur == "" {
		return "no chat selected"
	}
	return cur
}

var searchHeader = headerStyle.Render("Search")

// sidebarView renders the bordered chat-list pane.
func (m Model) sidebarView() string {
	var b strings.Builder

	header := ""
	if m.activePlat == core.PlatformAll {
		header += "◈ INBOX "
	} else {
		header += "◈ " + metaName(m.activePlat) + " "
	}

	totalUnread := 0
	for _, c := range m.chats {
		totalUnread += c.UnreadCount
	}
	if totalUnread > 0 {
		header += unreadBadge.Render(" " + strconv.Itoa(totalUnread) + " ")
	}
	b.WriteString(sectionStyle.Render(header))
	b.WriteString("\n")

	if len(m.chats) == 0 {
		b.WriteString("\n" + dimStyle.Render("no chats yet"))
	} else {
		itemWidth := m.sidebarWidth - 6
		if itemWidth < 10 {
			itemWidth = 10
		}

		for i, c := range m.chats {
			name := c.Name
			if name == "" {
				name = c.ID
			}

			unread := c.UnreadCount
			selected := i == m.chatIdx && m.focus == PaneSidebar

			title := truncateRight(name, itemWidth-2)

			// Merged inbox rows carry a platform chip.
			tag := ""
			if m.activePlat == core.PlatformAll {
				tag = chatTag(c.Platform) + " "
			}

			badge := ""

			if unread > 0 {
				if selected {
					badge = selectedBadge.Render(strconv.Itoa(unread))
				} else {
					badge = unreadBadge.Render(strconv.Itoa(unread))
				}
			}

			marker := "  "
			if selected {
				marker = selectedRow.Render("▌ ") +
					selectedRow.Render(title)
			}

			line := marker
			if !selected {
				line = lipgloss.NewStyle().Width(itemWidth).Render(tag + title)
			}

			if badge != "" {
				pad := itemWidth -
					lipgloss.Width(tag) -
					lipgloss.Width(title) -
					lipgloss.Width(badge)

				if pad < 1 {
					pad = 1
				}

				line += strings.Repeat(" ", pad) + badge
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	box := paneBorder(m.focus == PaneSidebar).
		Width(m.sidebarWidth - 2).
		Render(b.String())

	// Pad to full pane height so the border reaches the bottom.
	h := m.chatPaneHeight()
	pad := h - lipgloss.Height(box)
	if pad > 0 {
		box += "\n" + strings.Repeat("\n", pad-1)
	}
	return box
}

// followLatest jumps the message viewport to the newest message. Used
// when a message arrives or is sent, so the conversation always shows
// the latest turn regardless of where the user had scrolled.
func (m Model) followLatest() Model {
	v := m.viewport
	v.vp.GotoBottom()
	m.viewport = v
	return m
}

// markRead clears the unread badge for chatID once its messages were
// viewed, persisting the cleared count so it survives a restart.
func (m Model) markRead(chatID string) Model {
	if chatID == "" {
		return m
	}
	if m.readCleared == nil {
		m.readCleared = map[string]bool{}
	}
	m.readCleared[chatID] = true

	changed := false
	for i := range m.chats {
		if m.chats[i].ID == chatID && m.chats[i].UnreadCount > 0 {
			m.chats[i].UnreadCount = 0
			changed = true
		}
	}
	if changed && m.store != nil {
		_ = m.store.SaveChats(m.chats)
	}
	return m
}

// applyReadCleared squashes unread counts for chats already viewed this
// session when a fresh (server or cached) chat list is ingested.
func (m Model) applyReadCleared() {
	if m.readCleared == nil {
		return
	}
	m.applyReadClearedTo(m.chats)
}

// applyReadClearedTo is the slice-target variant of applyReadCleared.
func (m Model) applyReadClearedTo(chats []core.Chat) {
	if m.readCleared == nil {
		return
	}
	for i := range chats {
		if m.readCleared[chats[i].ID] {
			chats[i].UnreadCount = 0
		}
	}
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
