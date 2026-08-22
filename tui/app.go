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

	"github.com/SaiArnav/ZetaChat/config"
	"github.com/SaiArnav/ZetaChat/core"
	"github.com/SaiArnav/ZetaChat/storage"
)

// stage is the high-level UI state.
type stage int

const (
	stageBooting   stage = iota // connecting to all platforms
	stageAuth                   // interactive auth prompt
	stageDashboard              // platform dashboard
	stageLoading                // fetching chats after auth
	stageReady                  // full app: sidebar + chat + composer
	stageQR                     // scan-to-pair screen (WhatsApp)
)

// platformState tracks one platform's connection and account info.
type platformState struct {
	ready    bool
	pairing  bool // waiting for the user to scan a QR code
	err      error
	selfName string
	chatN    int
}

// Model is the root Bubble Tea model for ZetaChat.
type Model struct {
	cfg    *config.Config
	store  *storage.Store
	ctx    context.Context
	cancel context.CancelFunc

	adapters   map[core.Platform]core.LiveMessenger
	platOrder  []core.Platform
	platState  map[core.Platform]*platformState
	activePlat core.Platform
	dashIdx    int

	quitting bool
	width    int
	height   int

	sidebarWidth    int
	sidebarDragging bool

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
	focusViewport bool
	composer      textinput.Model

	searchMode    bool
	search        textinput.Model
	searchResults []core.Message
	searchIdx     int

	flashUntil time.Time
	selfName   string
	qrCode     string
}

// NewModel builds the root Bubble Tea model.
func NewModel(cfg *config.Config, adapters map[core.Platform]core.LiveMessenger, store *storage.Store) Model {
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
	search.Placeholder = "Search messages…"
	search.PlaceholderStyle = dimStyle
	search.CharLimit = 200
	search.TextStyle = baseText

	order := platOrder(keysOf(adapters))

	states := make(map[core.Platform]*platformState, len(order))
	for _, p := range order {
		states[p] = &platformState{}
	}

	active := core.PlatformTelegram
	if len(order) > 0 {
		active = order[0]
	}

	return Model{
		cfg:          cfg,
		store:        store,
		ctx:          ctx,
		cancel:       cancel,
		adapters:     adapters,
		platOrder:    order,
		platState:    states,
		activePlat:   active,
		stage:        stageBooting,
		spinner:      sp,
		sidebarWidth: 32,
		composer:     composer,
		search:       search,
		viewport:     newMessagesViewport(),
		status:       bootingStatus(order),
	}
}

func keysOf(adapters map[core.Platform]core.LiveMessenger) []core.Platform {
	out := make([]core.Platform, 0, len(adapters))
	for p := range adapters {
		out = append(out, p)
	}
	return out
}

func bootingStatus(order []core.Platform) string {
	if len(order) == 0 {
		return "No platforms configured."
	}
	names := make([]string, 0, len(order))
	for _, p := range order {
		if meta, ok := platformRegistry[p]; ok {
			names = append(names, meta.name)
		} else {
			names = append(names, string(p))
		}
	}
	return "Connecting to " + strings.Join(names, " + ") + "…"
}

// activeAdapter returns the adapter for the platform being browsed.
func (m Model) activeAdapter() core.LiveMessenger {
	return m.adapters[m.activePlat]
}

// Init starts background work: connects every configured platform.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick, bootGraceCmd()}
	for _, p := range m.platOrder {
		a := m.adapters[p]
		a.Connect(m.ctx)
		cmds = append(cmds,
			waitReady(p, a),
			watchUpdates(p, a),
			watchQRCmd(p, a),
		)
	}
	return tea.Batch(cmds...)
}

// bootGraceCmd opens the dashboard after a short grace period.
func bootGraceCmd() tea.Cmd {
	return tea.Tick(8*time.Second, func(time.Time) tea.Msg {
		return bootGraceMsg{}
	})
}

// waitReady resolves when one platform's adapter is authed (or failed).
func waitReady(p core.Platform, a core.LiveMessenger) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-a.Ready():
			if err := a.AuthErr(); err != nil {
				return connMsg{plat: p, err: err}
			}
			return connMsg{plat: p}
		case <-time.After(5 * time.Minute):
			return connMsg{plat: p, err: errors.New("connection timed out after 5 minutes")}
		}
	}
}

// watchUpdates relays live messages from one adapter into the UI. It blocks
// until a message arrives; handleLive re-arms it after every message so the
// UI keeps listening for the lifetime of the program.
func watchUpdates(p core.Platform, a core.LiveMessenger) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-a.Updates()
		if !ok {
			return nil
		}
		return liveMsg{msg: msg}
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
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// ---------------------------------------------------------------------------
// message types

type connMsg struct {
	plat core.Platform
	err  error
}

// bootGraceMsg forces the dashboard open shortly after launch, even if
// some platform never reaches a terminal state.
type bootGraceMsg struct{}

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

		// Keep the composer and search input inside the available
		// width of the chat pane.
		inputWidth := m.chatPaneWidth() - 2
		if inputWidth < 1 {
			inputWidth = 1
		}

		m.composer.Width = inputWidth
		m.search.Width = inputWidth

		m = m.refreshMessagesView()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case bootGraceMsg:
		if m.stage == stageBooting {
			m.stage = stageDashboard
		}
		return m, nil

	case connMsg:
		st := m.platState[msg.plat]
		if st == nil {
			return m, nil
		}
		st.ready = true
		st.err = msg.err
		name := metaName(msg.plat)

		if msg.err != nil {
			m.flash("["+name+"] "+msg.err.Error(), true)
		} else {
			st.selfName = m.adapters[msg.plat].SelfName()
			m.status = "[" + name + "] connected as " + st.selfName
		}

		if m.stage == stageQR && msg.plat == m.activePlat && msg.err == nil {
			m.qrCode = ""
			m.stage = stageDashboard
		}

		if m.allResolved() {
			if m.stage != stageQR {
				m.stage = stageDashboard
			}
			m.dashIdx = 0
		}
		return m, nil

	case qrMsg:
		if _, ok := m.adapters[msg.plat]; !ok {
			return m, nil
		}
		// Store the code but don't hijack the screen — the dashboard
		// stays in control; selecting the platform opens the QR view.
		if st := m.platState[msg.plat]; st != nil {
			st.pairing = true
		}
		m.activePlat = msg.plat
		m.qrCode = msg.code
		return m, watchQRCmd(msg.plat, m.adapters[msg.plat])

	case authPromptMsg:
		m.stage = stageAuth
		return m, m.startAuthPrompt(msg)

	case cachedChatsMsg:
		msg.chats = filterPlatform(msg.chats, m.activePlat)
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
			m.stage = stageDashboard
			return m, nil
		}

		m.chats = filterPlatform(msg.chats, m.activePlat)
		m.stage = stageReady

		if st := m.platState[m.activePlat]; st != nil {
			st.chatN = len(m.chats)
		}

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

		m.status = fmt.Sprintf(
			"%d chats on %s",
			len(m.chats),
			metaName(m.activePlat),
		)

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
			m.status = fmt.Sprintf(
				"Showing %d cached messages…",
				len(msg.msgs),
			)
		}

		m = m.refreshMessagesView()
		return m, refreshUnread(m)

	case sendResultMsg:
		// Ignore a delayed send response belonging to a chat
		// that is no longer selected.
		if msg.chatID != m.currentChatID() {
			return m, nil
		}

		if msg.err != nil {
			m.flash("Send failed: "+msg.err.Error(), true)
			return m, nil
		}

		m.flash("Sent", false)

		return m, refreshUnread(m)

	case searchResultMsg:
		if msg.err != nil {
			m.flash("Search failed: "+msg.err.Error(), true)
			return m, nil
		}

		m.searchResults = msg.msgs
		m.searchIdx = 0
		m.status = fmt.Sprintf(
			"%d results for %q",
			len(msg.msgs),
			msg.query,
		)

		return m, nil

	case liveMsg:
		return m.handleLive(msg.msg)

	case tea.MouseMsg:
		if m.stage != stageReady || m.searchMode {
			return m, nil
		}
		dividerX := m.sidebarWidth
		switch {
		case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
			if msg.X >= dividerX-1 && msg.X <= dividerX+1 {
				m.sidebarDragging = true
			}
		case msg.Action == tea.MouseActionRelease:
			m.sidebarDragging = false
		case msg.Action == tea.MouseActionMotion && m.sidebarDragging:
			newWidth := msg.X
			if newWidth < 15 {
				newWidth = 15
			}
			if newWidth > m.width-20 {
				newWidth = m.width - 20
			}
			m.sidebarWidth = newWidth
			m = m.refreshMessagesView()
		}
		return m, nil
	}

	return m, nil
}

// handleLive routes an incoming live message.
func (m Model) handleLive(msg core.Message) (tea.Model, tea.Cmd) {
	rearm := watchUpdates(msg.Platform, m.adapters[msg.Platform])
	tag := metaName(msg.Platform)

	// Only surface notifications when not focused on that chat.
	focusedChat := ""
	if !m.searchMode && m.stage == stageReady &&
		m.chatIdx >= 0 && m.chatIdx < len(m.chats) {
		cur := m.chats[m.chatIdx]
		if cur.Platform == msg.Platform || cur.Platform == "" {
			focusedChat = cur.ID
		}
	}

	if msg.ChatID == focusedChat {
		// The live echo of a message we just sent replaces our local echo,
		// so the sent message doesn't show up twice.
		if len(m.messages) > 0 {
			last := &m.messages[len(m.messages)-1]
			if msg.Out && strings.HasPrefix(last.ID, "local-") &&
				last.ChatID == msg.ChatID && last.Text == msg.Text &&
				time.Since(last.Timestamp) < time.Minute {
				*last = msg
				m.status = "Sent"
				m = m.refreshMessagesView()
				return m, tea.Batch(refreshUnread(m), rearm)
			}
		}
		// Append if not a duplicate of the last message.
		if len(m.messages) == 0 || m.messages[len(m.messages)-1].ID != msg.ID {
			m.messages = append(m.messages, msg)
		}
		m.status = fmt.Sprintf("New message from %s", msg.Sender.DisplayName)
		m = m.refreshMessagesView()
		return m, tea.Batch(refreshUnread(m), rearm)
	}

	// Elsewhere: bump unread + notify.
	for i, c := range m.chats {
		if c.ID == msg.ChatID && (c.Platform == msg.Platform || c.Platform == "") {
			m.chats[i].UnreadCount++
			break
		}
	}
	sender := msg.Sender.DisplayName
	if sender == "" {
		sender = "Unknown"
	}
	m.flash("["+tag+"] "+sender+": "+msg.Text, false)
	return m, tea.Batch(refreshUnread(m), rearm)
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

// allResolved reports whether every configured platform finished
// connecting — linked, errored, or waiting for a QR scan.
func (m Model) allResolved() bool {
	for _, p := range m.platOrder {
		st := m.platState[p]
		if st == nil {
			return false
		}
		if !st.ready && !st.pairing {
			return false
		}
	}
	return len(m.platOrder) > 0
}

// metaName returns the display name for a platform.
func metaName(p core.Platform) string {
	if meta, ok := platformRegistry[p]; ok {
		return meta.name
	}
	return strings.ToUpper(string(p))
}

// filterPlatform keeps only chats belonging to the platform (legacy rows
// with an empty Platform count as Telegram).
func filterPlatform(chats []core.Chat, p core.Platform) []core.Chat {
	out := chats[:0:0]
	for _, c := range chats {
		if c.Platform == p || c.Platform == "" && p == core.PlatformTelegram {
			out = append(out, c)
		}
	}
	return out
}

// openDashboard returns to the platform selection screen.
func (m *Model) openDashboard() {
	m.stage = stageDashboard
	m.chats = nil
	m.messages = nil
	m.chatIdx = 0
	m.searchMode = false
	m.search.Blur()
	m.focusComposer = false
	m.composer.Blur()
	m.viewport.dirty = true
}
