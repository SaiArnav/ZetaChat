package telegram

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gotd/td/tg"

	"github.com/SaiArnav/ZetaChat/core"
)

// entityIndex holds the user/chat/channel maps built from an API response.
type entityIndex struct {
	users    map[int64]*tg.User
	basicG   map[int64]*tg.Chat
	channels map[int64]*tg.Channel
}

func newEntityIndex() entityIndex {
	return entityIndex{
		users:    make(map[int64]*tg.User),
		basicG:   make(map[int64]*tg.Chat),
		channels: make(map[int64]*tg.Channel),
	}
}

func (e entityIndex) indexUsers(users []tg.UserClass) {
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			e.users[user.ID] = user
		}
	}
}

func (e entityIndex) indexChats(chats []tg.ChatClass) {
	for _, c := range chats {
		switch c := c.(type) {
		case *tg.Chat:
			e.basicG[c.ID] = c
		case *tg.Channel:
			e.channels[c.ID] = c
		}
	}
}

// buildIndex updates the adapter's shared name index from raw entity lists.
func (a *Adapter) buildIndex(users []tg.UserClass, chats []tg.ChatClass) {
	idx := newEntityIndex()
	idx.indexUsers(users)
	idx.indexChats(chats)

	a.mu.Lock()
	defer a.mu.Unlock()
	for id, u := range idx.users {
		a.users[id] = userName(u)
	}
	for id, c := range idx.basicG {
		a.basicG[id] = c.Title
	}
	for id, c := range idx.channels {
		a.channels[id] = c.Title
	}
}

// userName renders a display name for a user.
func userName(u *tg.User) string {
	if u.FirstName != "" && u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.LastName != "" {
		return u.LastName
	}
	if u.Username != "" {
		return "@" + u.Username
	}
	return fmt.Sprintf("user %d", u.ID)
}

// chatIDFor produces the stable core chatID for a peer.
func chatIDFor(p tg.PeerClass) string {
	switch p := p.(type) {
	case *tg.PeerUser:
		return "u" + fmt.Sprint(p.UserID)
	case *tg.PeerChat:
		return "g" + fmt.Sprint(p.ChatID)
	case *tg.PeerChannel:
		return "s" + fmt.Sprint(p.ChannelID)
	}
	return ""
}

// senderName resolves a peer (sender) to a display name.
func (a *Adapter) senderName(p tg.PeerClass) (core.User, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch p := p.(type) {
	case *tg.PeerUser:
		name, ok := a.users[p.UserID]
		if !ok {
			name = fmt.Sprintf("user %d", p.UserID)
		}
		return core.User{
			ID:          fmt.Sprint(p.UserID),
			Platform:    core.PlatformTelegram,
			DisplayName: name,
		}, true
	case *tg.PeerChat:
		return core.User{
			ID:          fmt.Sprint(p.ChatID),
			Platform:    core.PlatformTelegram,
			DisplayName: a.basicG[p.ChatID],
		}, true
	case *tg.PeerChannel:
		return core.User{
			ID:          fmt.Sprint(p.ChannelID),
			Platform:    core.PlatformTelegram,
			DisplayName: a.channels[p.ChannelID],
		}, true
	}
	return core.User{}, false
}

// messageSummary describes a message, falling back to a media label.
func messageSummary(msg *tg.Message) string {
	if msg.Message != "" {
		return msg.Message
	}
	switch media := msg.Media.(type) {
	case *tg.MessageMediaPhoto:
		return "[photo]"
	case *tg.MessageMediaDocument:
		if d, ok := media.Document.AsNotEmpty(); ok {
			for _, attr := range d.Attributes {
				if f, ok := attr.(*tg.DocumentAttributeFilename); ok && f.FileName != "" {
					return "[file: " + f.FileName + "]"
				}
			}
		}
		return "[document]"
	case *tg.MessageMediaWebPage:
		return "[link]"
	case *tg.MessageMediaGeo:
		return "[location]"
	case *tg.MessageMediaPoll:
		return "[poll]"
	case *tg.MessageMediaContact:
		return "[contact]"
	default:
		if msg.Media != nil {
			return "[media]"
		}
		return ""
	}
}

// mapMessage converts a tg.MessageClass to core.Message.
func (a *Adapter) mapMessage(cls tg.MessageClass) (core.Message, bool) {
	msg, ok := cls.AsNotEmpty()
	if !ok {
		return core.Message{}, false
	}
	tm, ok := msg.(*tg.Message)
	if !ok {
		return core.Message{}, false
	}

	chatID := chatIDFor(tm.PeerID)
	if chatID == "" {
		return core.Message{}, false
	}

	// Channel posts have no FromID: sender is the channel itself.
	from := tm.PeerID
	if tm.FromID != nil {
		from = tm.FromID
	}
	sender, _ := a.senderName(from)

	return core.Message{
		ID:        fmt.Sprint(tm.ID),
		Platform:  core.PlatformTelegram,
		ChatID:    chatID,
		Sender:    sender,
		Text:      messageSummary(tm),
		Timestamp: time.Unix(int64(tm.Date), 0),
		Out:       tm.Out,
	}, true
}

// chatFromDialog maps a dialog + top message to core.Chat.
func (a *Adapter) chatFromDialog(dialog *tg.Dialog, msgs []tg.MessageClass, idx entityIndex) (core.Chat, bool) {
	chatID := chatIDFor(dialog.Peer)
	if chatID == "" {
		return core.Chat{}, false
	}

	var (
		title   string
		isGroup bool
	)
	switch p := dialog.Peer.(type) {
	case *tg.PeerUser:
		if u, ok := idx.users[p.UserID]; ok {
			title = userName(u)
		}
	case *tg.PeerChat:
		if c, ok := idx.basicG[p.ChatID]; ok {
			title = c.Title
		}
		isGroup = true
	case *tg.PeerChannel:
		if c, ok := idx.channels[p.ChannelID]; ok {
			title = c.Title
			isGroup = !c.Broadcast
		}
	}
	if title == "" {
		title = chatID
	}

	// Find the top message for preview.
	var last *core.Message
	for _, cls := range msgs {
		m, ok := cls.AsNotEmpty()
		if !ok || m.GetID() != dialog.TopMessage {
			continue
		}
		mm, ok := a.mapMessage(cls)
		if ok {
			last = &mm
		}
		break
	}

	updated := time.Now()
	if last != nil {
		updated = last.Timestamp
	}

	return core.Chat{
		ID:          chatID,
		Platform:    core.PlatformTelegram,
		Name:        title,
		IsGroup:     isGroup,
		UnreadCount: dialog.UnreadCount,
		LastMessage: last,
		UpdatedAt:   updated,
	}, true
}

// sortChats orders chats by most recent activity first.
func sortChats(chats []core.Chat) {
	sort.SliceStable(chats, func(i, j int) bool {
		return chats[i].UpdatedAt.After(chats[j].UpdatedAt)
	})
}

func parseID(s string) int64 {
	var id int64
	fmt.Sscanf(s, "%d", &id)
	return id
}

// refreshPeers rebuilds the chatID -> peer resolver map.
func (a *Adapter) refreshPeers(users []tg.UserClass, chats []tg.ChatClass) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, cls := range users {
		u, ok := cls.(*tg.User)
		if !ok {
			continue
		}
		a.peers["u"+fmt.Sprint(u.ID)] = peerInfo{
			kind:       "user",
			id:         u.ID,
			accessHash: u.AccessHash,
			username:   u.Username,
		}
	}
	for _, cls := range chats {
		switch c := cls.(type) {
		case *tg.Chat:
			a.peers["g"+fmt.Sprint(c.ID)] = peerInfo{kind: "chat", id: c.ID, title: c.Title}
		case *tg.Channel:
			a.peers["s"+fmt.Sprint(c.ID)] = peerInfo{
				kind:       "channel",
				id:         c.ID,
				accessHash: c.AccessHash,
				title:      c.Title,
				username:   c.Username,
			}
		}
	}
}

// resolvePeerOrUsername resolves a core chatID or "@username" to an input peer.
func (a *Adapter) resolvePeerOrUsername(chatID string) (tg.InputPeerClass, error) {
	if strings.HasPrefix(chatID, "@") {
		return a.resolveUsername(a.ctx, chatID)
	}
	return a.resolvePeer(chatID)
}

// resolvePeer looks up the input peer for a core chatID.
func (a *Adapter) resolvePeer(chatID string) (tg.InputPeerClass, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	p, ok := a.peers[chatID]
	if !ok {
		return nil, ErrChatNotFound
	}
	return p.inputPeer(), nil
}

// Chats returns the user's dialog list, most recent first.
func (a *Adapter) Chats() ([]core.Chat, error) {
	a.mu.Lock()
	authed := a.authed
	raw := a.raw
	a.mu.Unlock()
	if !authed {
		return nil, ErrNotAuthorized
	}

	dialogs, err := raw.MessagesGetDialogs(a.ctx, &tg.MessagesGetDialogsRequest{
		OffsetID:   0,
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: get dialogs: %w", err)
	}

	var (
		dialogList []tg.DialogClass
		msgList    []tg.MessageClass
		users      []tg.UserClass
		chatEnts   []tg.ChatClass
	)
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		dialogList = d.Dialogs
		msgList = d.Messages
		users = d.Users
		chatEnts = d.Chats
	case *tg.MessagesDialogsSlice:
		dialogList = d.Dialogs
		msgList = d.Messages
		users = d.Users
		chatEnts = d.Chats
	default:
		return nil, fmt.Errorf("telegram: unexpected dialogs response %T", dialogs)
	}

	a.buildIndex(users, chatEnts)
	a.refreshPeers(users, chatEnts)
	idx := newEntityIndex()
	idx.indexUsers(users)
	idx.indexChats(chatEnts)

	chats := make([]core.Chat, 0, len(dialogList))
	for _, cls := range dialogList {
		d, ok := cls.(*tg.Dialog)
		if !ok {
			continue
		}
		if c, ok := a.chatFromDialog(d, msgList, idx); ok {
			chats = append(chats, c)
		}
	}

	sortChats(chats)
	return chats, nil
}

// Messages returns message history for a chat, oldest first.
func (a *Adapter) Messages(chatID string) ([]core.Message, error) {
	a.mu.Lock()
	authed := a.authed
	raw := a.raw
	a.mu.Unlock()
	if !authed {
		return nil, ErrNotAuthorized
	}

	peer, err := a.resolvePeerOrUsername(chatID)
	if err != nil {
		return nil, err
	}

	hist, err := raw.MessagesGetHistory(a.ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: get history: %w", err)
	}

	messages, users, chats := mapSearchResults(hist)
	a.buildIndex(users, chats)

	// History is newest-first; flip to oldest-first.
	out := make([]core.Message, 0, len(messages))
	for _, cls := range messages {
		if m, ok := a.mapMessage(cls); ok {
			out = append(out, m)
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// SendMessage sends a text message to a chat.
func (a *Adapter) SendMessage(chatID string, text string) error {
	a.mu.Lock()
	authed := a.authed
	sender := a.sender
	a.mu.Unlock()
	if !authed {
		return ErrNotAuthorized
	}

	peer, err := a.resolvePeerOrUsername(chatID)
	if err != nil {
		return err
	}

	if _, err := sender.To(peer).Text(a.ctx, text); err != nil {
		return fmt.Errorf("telegram: send message: %w", err)
	}
	return nil
}

// Search searches all chats and returns matching messages.
func (a *Adapter) Search(query string) ([]core.Message, error) {
	a.mu.Lock()
	authed := a.authed
	raw := a.raw
	a.mu.Unlock()
	if !authed {
		return nil, ErrNotAuthorized
	}

	res, err := raw.MessagesSearchGlobal(a.ctx, &tg.MessagesSearchGlobalRequest{
		Q:     query,
		Limit: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: search: %w", err)
	}

	messages, users, chats := mapSearchResults(res)

	a.buildIndex(users, chats)
	a.refreshPeers(users, chats)

	out := make([]core.Message, 0, len(messages))
	for _, cls := range messages {
		if m, ok := a.mapMessage(cls); ok {
			out = append(out, m)
		}
	}
	return out, nil
}
func mapSearchResults(
	res tg.MessagesMessagesClass,
) (
	messages []tg.MessageClass,
	users []tg.UserClass,
	chats []tg.ChatClass,
) {
	switch v := res.(type) {
	case *tg.MessagesMessages:
		return v.GetMessages(), v.GetUsers(), v.GetChats()

	case *tg.MessagesMessagesSlice:
		return v.GetMessages(), v.GetUsers(), v.GetChats()

	case *tg.MessagesChannelMessages:
		return v.GetMessages(), v.GetUsers(), v.GetChats()

	case *tg.MessagesMessagesNotModified:
		return nil, nil, nil

	default:
		return nil, nil, nil
	}
}

// resolveUsername resolves a username like "username" or "@username" to an
// input peer, so `zeta send @arnav "hi"` works even for unknown chats.
func (a *Adapter) resolveUsername(ctx context.Context, username string) (tg.InputPeerClass, error) {
	uname := strings.TrimPrefix(strings.TrimSpace(username), "@")

	a.mu.Lock()
	authed := a.authed
	raw := a.raw
	a.mu.Unlock()
	if !authed {
		return nil, ErrNotAuthorized
	}

	resolved, err := raw.ContactsResolveUsername(a.ctx, &tg.ContactsResolveUsernameRequest{Username: uname})
	if err != nil {
		return nil, fmt.Errorf("telegram: resolve username: %w", err)
	}

	for _, cls := range resolved.Users {
		if u, ok := cls.(*tg.User); ok {
			a.mu.Lock()
			a.peers["u"+fmt.Sprint(u.ID)] = peerInfo{
				kind:       "user",
				id:         u.ID,
				accessHash: u.AccessHash,
				username:   u.Username,
			}
			a.users[u.ID] = userName(u)
			a.mu.Unlock()
			return &tg.InputPeerUser{UserID: u.ID, AccessHash: u.AccessHash}, nil
		}
	}
	for _, cls := range resolved.Chats {
		if ch, ok := cls.(*tg.Channel); ok {
			a.mu.Lock()
			a.peers["s"+fmt.Sprint(ch.ID)] = peerInfo{
				kind:       "channel",
				id:         ch.ID,
				accessHash: ch.AccessHash,
				title:      ch.Title,
				username:   ch.Username,
			}
			a.channels[ch.ID] = ch.Title
			a.mu.Unlock()
			return &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}, nil
		}
	}
	return nil, ErrChatNotFound
}
