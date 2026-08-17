package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/SaiArnav/ZetaChat/core"
)

func (a *Adapter) initUpdates() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.updates == nil {
		a.updates = make(chan core.Message, 128)
	}
}

// onUpdate is wired into telegram.Options.UpdateHandler and forwards
// incoming messages to the TUI through a.updates.
func (a *Adapter) onUpdate(ctx context.Context, u tg.UpdatesClass) error {
	switch u := u.(type) {
	case *tg.UpdateShortMessage:
		idx := newEntityIndex()
		return a.pushUpdateMessage(&tg.Message{
			Out:     u.Out,
			ID:      u.ID,
			PeerID:  &tg.PeerUser{UserID: u.UserID},
			FromID:  &tg.PeerUser{UserID: u.UserID},
			Date:    u.Date,
			Message: u.Message,
		}, idx)

	case *tg.UpdateShortChatMessage:
		idx := newEntityIndex()
		return a.pushUpdateMessage(&tg.Message{
			Out:     u.Out,
			ID:      u.ID,
			PeerID:  &tg.PeerChat{ChatID: u.ChatID},
			FromID:  &tg.PeerUser{UserID: u.FromID},
			Date:    u.Date,
			Message: u.Message,
		}, idx)

	case *tg.Updates:
		idx := newEntityIndex()
		idx.indexUsers(u.Users)
		idx.indexChats(u.Chats)
		for _, upd := range u.Updates {
			if msg := updateMessage(upd); msg != nil {
				if err := a.pushUpdateMessage(msg, idx); err != nil {
					return err
				}
			}
		}

	case *tg.UpdatesCombined:
		idx := newEntityIndex()
		idx.indexUsers(u.Users)
		idx.indexChats(u.Chats)
		for _, upd := range u.Updates {
			if msg := updateMessage(upd); msg != nil {
				if err := a.pushUpdateMessage(msg, idx); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// updateMessage extracts a tg.Message from a single UpdateClass, if any.
func updateMessage(upd tg.UpdateClass) *tg.Message {
	switch upd := upd.(type) {
	case *tg.UpdateNewMessage:
		if m, ok := upd.Message.AsNotEmpty(); ok {
			if msg, ok := m.(*tg.Message); ok {
				return msg
			}
		}
	case *tg.UpdateNewChannelMessage:
		if m, ok := upd.Message.AsNotEmpty(); ok {
			if msg, ok := m.(*tg.Message); ok {
				return msg
			}
		}
	}
	return nil
}

func (a *Adapter) pushUpdateMessage(msg *tg.Message, idx entityIndex) error {
	// Seed the shared name index with any entities we just learned about.
	a.mu.Lock()
	for id, u := range idx.users {
		a.users[id] = userName(u)
	}
	for id, c := range idx.basicG {
		a.basicG[id] = c.Title
	}
	for id, c := range idx.channels {
		a.channels[id] = c.Title
	}
	a.mu.Unlock()

	m, ok := a.mapMessage(msg)
	if !ok {
		return nil
	}
	select {
	case a.updates <- m:
	default:
		// Drop if the UI is too slow; it will re-fetch history.
	}
	return nil
}
