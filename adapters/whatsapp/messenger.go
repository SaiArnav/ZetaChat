package whatsapp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/SaiArnav/ZetaChat/core"
)

// Login waits until pairing + connecting finished.
func (a *Adapter) Login() error {
	if a.ctx == nil {
		return ErrNotAuthorized
	}
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.Ready():
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runErr != nil {
		return a.runErr
	}
	if !a.paired {
		return ErrNotAuthorized
	}
	return nil
}

// Logout disconnects and deletes the local session (the phone keeps the
// linked-device entry until it expires; remove it from Linked Devices too).
func (a *Adapter) Logout() error {
	a.mu.Lock()
	client, paired := a.client, a.paired
	a.mu.Unlock()
	if !paired || client == nil {
		return ErrNotAuthorized
	}
	if err := client.Logout(a.ctx); err != nil {
		return fmt.Errorf("whatsapp logout: %w", err)
	}
	a.mu.Lock()
	a.paired = false
	a.authed = false
	a.mu.Unlock()
	return nil
}

// Chats returns the known conversation list: joined groups, DMs seen in
// history sync or live messages.
func (a *Adapter) Chats() ([]core.Chat, error) {
	if !a.isPaired() {
		return nil, ErrNotAuthorized
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]core.Chat, 0, len(a.chats))
	for _, c := range a.chats {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Messages returns locally accumulated history for one chat. WhatsApp has
// no server-side history API — content arrives via phone history sync and
// live events, which the adapter records continuously.
func (a *Adapter) Messages(chatID string) ([]core.Message, error) {
	if !a.isPaired() {
		return nil, ErrNotAuthorized
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	msgs := a.messages[strings.TrimPrefix(chatID, chatIDPrefix)]
	out := make([]core.Message, len(msgs))
	copy(out, msgs)
	reverse(out) // oldest-first for the viewport
	return out, nil
}

// SendMessage delivers a text message to a chat JID.
func (a *Adapter) SendMessage(chatID string, text string) error {
	a.mu.Lock()
	client, paired := a.client, a.paired
	a.mu.Unlock()
	if !paired || client == nil {
		return ErrNotAuthorized
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("whatsapp: empty message")
	}

	jid, err := parseJID(strings.TrimPrefix(chatID, chatIDPrefix))
	if err != nil {
		return err
	}

	ts, err := client.SendMessage(a.ctx, jid, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		return fmt.Errorf("whatsapp send: %w", err)
	}

	// Record our own send so it shows up without waiting for an echo.
	self := core.Message{
		ID:        "wa-" + ts.ID,
		Platform:  core.PlatformWhatsApp,
		ChatID:    "wa-" + jid.String(),
		Text:      text,
		Timestamp: time.Now().UTC(),
		Out:       true,
		Sender:    core.User{DisplayName: a.SelfName()},
	}
	a.pushMessage(self)
	return nil
}

// Search filters all locally accumulated messages by query.
func (a *Adapter) Search(query string) ([]core.Message, error) {
	if !a.isPaired() {
		return nil, ErrNotAuthorized
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	var out []core.Message
	for _, msgs := range a.messages {
		for _, m := range msgs {
			if strings.Contains(strings.ToLower(m.Text), q) {
				out = append(out, m)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out, nil
}

// Close disconnects from WhatsApp.
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	if a.cancel != nil {
		a.cancel()
	}
	if a.client != nil {
		a.client.Disconnect()
	}
	return nil
}

// ---------------------------------------------------------------------------
// internals

func (a *Adapter) isPaired() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	select {
	case <-a.ready:
	default:
	}
	return a.paired && a.runErr == nil
}

// absorbHistory ingests a history-sync payload delivered on first link.
func (a *Adapter) absorbHistory(e *events.HistorySync) {
	for _, conv := range e.Data.GetConversations() {
		jid, err := parseJID(conv.GetID())
		if err != nil {
			continue
		}

		name := conv.GetDisplayName()
		for _, msg := range conv.GetMessages() {
			key := msg.GetMessage().GetKey()
			wrapped := msg.GetMessage().GetMessage()
			text := wrapped.GetConversation()
			if text == "" {
				continue
			}
			ts := timeFromSeconds(int64(msg.GetMessage().GetMessageTimestamp()))
			m := core.Message{
				ID:        chatIDPrefix + key.GetID(),
				Platform:  core.PlatformWhatsApp,
				ChatID:    chatIDPrefix + jid.String(),
				Text:      text,
				Timestamp: ts,
				Sender:    core.User{DisplayName: name},
			}
			if key.GetFromMe() {
				m.Out = true
				m.Sender.DisplayName = a.SelfName()
			}
			a.pushMessage(m)
		}

		a.mu.Lock()
		id := chatIDPrefix + jid.String()
		c, ok := a.chats[id]
		if !ok {
			c = core.Chat{ID: id, Platform: core.PlatformWhatsApp}
		}
		c.Name = chatName(name, jid)
		c.IsGroup = jid.Server == types.GroupServer
		c.UpdatedAt = time.Now().UTC()
		a.chats[id] = c
		a.mu.Unlock()
	}
}

func (a *Adapter) mapMessage(info types.MessageInfo, text string) core.Message {
	senderName := info.PushName
	if senderName == "" {
		senderName = a.displayNameFor(info.Sender)
	}

	chatID := info.Chat.String()

	return core.Message{
		ID:        chatIDPrefix + info.ID,
		Platform:  core.PlatformWhatsApp,
		ChatID:    chatIDPrefix + chatID,
		Text:      text,
		Timestamp: info.Timestamp.UTC(),
		Out:       info.IsFromMe,
		Sender: core.User{
			ID:          info.Sender.String(),
			DisplayName: senderName,
		},
	}
}

// pushMessage appends to the in-memory history (newest last).
func (a *Adapter) pushMessage(msg core.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	jid := strings.TrimPrefix(msg.ChatID, chatIDPrefix)

	msgs := a.messages[jid]
	// Skip duplicates.
	if len(msgs) > 0 && msgs[len(msgs)-1].ID == msg.ID {
		return
	}
	a.messages[jid] = append(msgs, msg)

	// Keep memory bounded: newest 200 per chat.
	if n := len(a.messages[jid]); n > 200 {
		a.messages[jid] = a.messages[jid][n-200:]
	}
}

// rememberChat registers a chat entry for any message we see.
func (a *Adapter) rememberChat(msg core.Message) {
	jid := strings.TrimPrefix(msg.ChatID, chatIDPrefix)
	name := msg.Sender.DisplayName

	a.mu.Lock()
	defer a.mu.Unlock()
	id := chatIDPrefix + jid
	c, ok := a.chats[id]
	if !ok {
		c = core.Chat{ID: id, Platform: core.PlatformWhatsApp}
	}
	if c.Name == "" || (!msg.Out && name != "") {
		c.Name = chatName(name, mustParseJID(jid))
	}
	c.IsGroup = strings.Contains(jid, "@g.us")
	if msg.Timestamp.After(c.UpdatedAt) {
		c.UpdatedAt = msg.Timestamp
	}
	a.chats[id] = c
}

func (a *Adapter) rememberPushName(jid types.JID, pushName string) {
	if pushName == "" {
		return
	}
	a.mu.Lock()
	a.pushNames[jid] = pushName
	a.mu.Unlock()
}

func (a *Adapter) displayNameFor(jid types.JID) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if n, ok := a.pushNames[jid]; ok && n != "" {
		return n
	}
	return "+" + jid.User
}

func chatName(name string, jid types.JID) string {
	if name != "" {
		return name
	}
	if jid.Server == types.GroupServer {
		return "group-" + jid.User
	}
	return "+" + jid.User
}

func parseJID(s string) (types.JID, error) {
	j, err := types.ParseJID(s)
	if err != nil {
		return types.JID{}, fmt.Errorf("whatsapp: bad jid %q: %w", s, err)
	}
	return j, nil
}

func mustParseJID(s string) types.JID {
	j, _ := types.ParseJID(s)
	return j
}

// reverse flips a message slice into chronological order.
func reverse(msgs []core.Message) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}

// timeFromSeconds converts a unix-seconds timestamp into UTC.
func timeFromSeconds(sec int64) time.Time {
	if sec <= 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}
