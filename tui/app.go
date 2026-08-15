package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SaiArnav/ZetaChat/adapters/telegram"
	"github.com/SaiArnav/ZetaChat/config"
	"github.com/SaiArnav/ZetaChat/core"
	"github.com/SaiArnav/ZetaChat/storage"
)

// stage is the high-level UI state.
type stage int

const (
	stageBooting  stage = iota // connecting to Telegram
	stageAuth                  // interactive auth prompt
	stageLoading               // fetching chats after auth
	stageReady                 // full app: sidebar + chat + composer
)

// Model is the root Bubble Tea model for ZetaChat.
type Model struct {
	cfg     *config.Config
	adapter *telegram.Adapter
	store   *storage.Store
	ctx     context.Context
	cancel  context.CancelFunc

	quitting bool
	width    int
	height   int

	stage   stage
	spinner spinner.Model
	status  string

	authPrompt *authPromptState

	chats    []core.Chat
	chatIdx  int
	messages []core.Message
	cached   bool

	viewport      messagesViewport
	focusComposer bool
	composer      textinput.Model

	searchMode    bool
	search        textinput.Model
	searchResults []core.Message
	searchIdx     int

	flashUntil time.Time
	selfName   string
}

// NewModel builds the root Bubble Tea model.
func NewModel(cfg *config.Config, adapter *telegram.Adapter, store *storage.Store) Model {
	ctx, cancel := context.WithCancel(context.Background())

	sp := spinner.New()
	sp.Style = spinnerStyle()

	composer := textinput.New()
	composer.Prompt = "❯ "
	composer.PromptStyle = baseText.Bold(true).Foreground(lime)
	composer.Placeholder = "Message…"
	composer.PlaceholderStyle = dimStyle
	composer.CharLimit = 4096
	composer.TextStyle = baseText

	search := textinput.New()
	search.Prompt = "/ "
	search.PromptStyle = baseText.Bold(true).Foreground(cyan)
	search.Placeholder = "Search Telegram…"
	search.PlaceholderStyle = dimStyle
	search.CharLimit = 200
	search.TextStyle = baseText

	return Model{
		cfg:      cfg,
		adapter:  adapter,
		store:    store,
		ctx:      ctx,
		cancel:   cancel,
		stage:    stageBooting,
		spinner:  sp,
		composer: composer,
		search:   search,
		viewport: newMessagesViewport(),
		status:   "Connecting to Telegram…",
	}
}

// Init starts background work.
func (m Model) Init() tea.Cmd {
	m.adapter.Connect(m.ctx)
	return tea.Batch(
		m.spinner.Tick,
		waitReady(m.adapter),
		watchUpdates(m.adapter),
	)
}

// waitReady resolves when the adapter is authed (or failed).
func waitReady(a *telegram.Adapter) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-a.Ready():
			if err := a.AuthErr(); err != nil {
				return connErrMsg{err: err}
			}
			return connReadyMsg{}
		case <-time.After(120 * time.Second):
			return connErrMsg{err: errors.New("connection timed out after 120s")}
		}
	}
}

// watchUpdates relays live messages from the adapter into the UI.
func watchUpdates(a *telegram.Adapter) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-a.Updates():
			return liveMsg{msg: msg}
		case <-time.After(30 * time.Second):
			return nil
		}
	}
}

// UIPrompter implements telegram.Prompter by surfacing prompts through the
// Bubble Tea program and waiting for the user's answer.
type UIPrompter struct {
	Program *tea.Program
}

func (p *UIPrompter) ask(kind authPromptKind, title, body string) (string, error) {
	reply := make(chan string, 1)
	p.Program.Send(authPromptMsg{
		kind:  kind,
		title: title,
		body:  body,
		reply: reply,
	})
	select {
	case v := <-reply:
		return v, nil
	case <-time.After(3 * time.Minute):
		return "", errors.New("authentication timed out")
	}
}

func (p *UIPrompter) AskPhone() (string, error) {
	return p.ask(authPhone, "Phone number",
		"Enter your phone number in international format, e.g. +15551234567")
}

func (p *UIPrompter) AskCode() (string, error) {
	return p.ask(authCode, "Login code",
		"Enter the code Telegram sent you (or check your other sessions).")
}

func (p *UIPrompter) AskPassword() (string, error) {
	return p.ask(authPassword, "Two-factor password",
		"Your account has 2FA enabled. Enter your password:")
}

func (p *UIPrompter) AskTerms(title, text string) (bool, error) {
	body := "Telegram requires accepting their Terms of Service first.\n\n" +
		truncate(text, 600) + "\n\nAccept? [y/N]"
	ans, err := p.ask(authTerms, title, body)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(ans), "y"), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ---------------------------------------------------------------------------
// message types

type connReadyMsg struct{}
type connErrMsg struct{ err error }

type cachedChatsMsg struct{ chats []core.Chat }
type chatsLoadedMsg struct {
	chats []core.Chat
	err   error
}
type messagesLoadedMsg struct {
	chatID string
	msgs   []core.Message
	err    error
}
type cachedMessagesMsg struct {
	chatID string
	msgs   []core.Message
}
type sendResultMsg struct {
	chatID string
	text   string
	err    error
}
type searchResultMsg struct {
	query string
	msgs  []core.Message
	err   error
}
type liveMsg struct{ msg core.Message }

// authPromptMsg asks the user for an auth input (phone/code/password/terms).
type authPromptMsg struct {
	kind  authPromptKind
	title string
	body  string
	reply chan string
}

type authPromptKind int

const (
	authPhone authPromptKind = iota
	authCode
	authPassword
	authTerms
)

// ---------------------------------------------------------------------------
// Update

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.refreshMessagesView()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case connReadyMsg:
		m.stage = stageLoading
		m.status = "Connected — loading your chats…"
		m.selfName = m.adapter.SelfName()
		return m, tea.Batch(
			loadCachedChatsCmd(m.store),
			loadChatsCmd(m.adapter, m.store),
		)

	case connErrMsg:
		m.stage = stageAuth
		m.status = "Connection failed: " + msg.err.Error()
		m.flash("Connection failed", true)
		return m, nil

	case authPromptMsg:
		m.stage = stageAuth
		return m, m.startAuthPrompt(msg)

	case cachedChatsMsg:
		if len(msg.chats) == 0 {
			return m, nil
		}
		m.chats = msg.chats
		m.stage = stageLoading
		if m.chatIdx >= len(m.chats) {
			m.chatIdx = 0
		}
		return m, openCachedChatCmd(m)

	case chatsLoadedMsg:
		if msg.err != nil {
			m.status = "Chats failed: " + msg.err.Error()
			m.flash("Failed to load chats", true)
			return m, nil
		}
		m.chats = msg.chats
		m.stage = stageReady
		// Keep current selection when possible.
		keep := ""
		if m.chatIdx < len(m.chats) {
			keep = m.chats[m.chatIdx].ID
		}
		m.chatIdx = 0
		for i, c := range m.chats {
			if c.ID == keep {
				m.chatIdx = i
				break
			}
		}
		m.status = fmt.Sprintf("%d chats on Telegram", len(m.chats))
		cmds := []tea.Cmd{refreshUnread(m)}
		if len(m.chats) > 0 {
			cmds = append(cmds, openChatCmd(m))
		}
		return m, tea.Batch(cmds...)

	case messagesLoadedMsg:
		if msg.err != nil {
			m.flash("Failed to load messages: "+msg.err.Error(), true)
			return m, nil
		}
		if msg.chatID != m.currentChatID() {
			return m, nil // stale response from a previous selection
		}
		m.messages = msg.msgs
		m.status = fmt.Sprintf("%d messages", len(m.messages))
		m.cached = false
		m = m.refreshMessagesView()
		return m, refreshUnread(m)

	case cachedMessagesMsg:
		if msg.chatID != m.currentChatID() {
			return m, nil
		}
		m.messages = msg.msgs
		m.cached = true
		if len(msg.msgs) > 0 {
			m.status = fmt.Sprintf("Showing %d cached messages…", len(msg.msgs))
		}
		m = m.refreshMessagesView()
		return m, refreshUnread(m)

	case sendResultMsg:
		if msg.err != nil {
			m.flash("Send failed: "+msg.err.Error(), true)
			return m, nil
		}
		msgOut := core.Message{
			ID:        fmt.Sprintf("local-%d", time.Now().UnixNano()),
			Platform:  core.PlatformTelegram,
			ChatID:    msg.chatID,
			Text:      msg.text,
			Timestamp: time.Now(),
			Out:       true,
		}
		m.messages = append(m.messages, msgOut)
		m.composer.SetValue("")
		m.flash("Sent", false)
		m = m.refreshMessagesView()
		return m, refreshUnread(m)

	case searchResultMsg:
		if msg.err != nil {
			m.flash("Search failed: "+msg.err.Error(), true)
			return m, nil
		}
		m.searchResults = msg.msgs
		m.searchIdx = 0
		m.status = fmt.Sprintf("%d results for %q", len(msg.msgs), msg.query)
		return m, nil

	case liveMsg:
		return m.handleLive(msg.msg)
	}
	return m, nil
}

// handleLive routes an incoming live message.
func (m Model) handleLive(msg core.Message) (tea.Model, tea.Cmd) {
	// Only surface notifications when not focused on that chat.
	focusedChat := ""
	if !m.searchMode && len(m.chats) > 0 && m.chatIdx < len(m.chats) {
		focusedChat = m.chats[m.chatIdx].ID
	}

	if msg.ChatID == focusedChat {
		// Append if not a duplicate of the last message.
		if len(m.messages) == 0 || m.messages[len(m.messages)-1].ID != msg.ID {
			m.messages = append(m.messages, msg)
		}
		m.status = fmt.Sprintf("New message from %s", msg.Sender.DisplayName)
		m = m.refreshMessagesView()
		return m, refreshUnread(m)
	}

	// Elsewhere: bump unread + notify.
	for i, c := range m.chats {
		if c.ID == msg.ChatID {
			m.chats[i].UnreadCount++
			break
		}
	}
	sender := msg.Sender.DisplayName
	if sender == "" {
		sender = "Unknown"
	}
	m.flash("[TG] "+sender+": "+msg.Text, false)
	return m, refreshUnread(m)
}

// flash sets a transient notification message.
func (m *Model) flash(text string, isErr bool) {
	if isErr {
		m.status = text
		return
	}
	m.status = text
	m.flashUntil = time.Now().Add(6 * time.Second)
}

// refreshUnread persists state; kept as a trivial cmd so the UI re-renders.
func refreshUnread(m Model) tea.Cmd {
	return func() tea.Msg { return nil }
}
