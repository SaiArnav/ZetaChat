package telegram

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestMapSearchResultsMessages(t *testing.T) {
	msg := &tg.MessagesMessages{
		Messages: []tg.MessageClass{
			&tg.MessageEmpty{ID: 1},
		},
		Users: []tg.UserClass{
			&tg.UserEmpty{ID: 10},
		},
		Chats: []tg.ChatClass{
			&tg.ChatEmpty{ID: 20},
		},
	}

	messages, users, chats := mapSearchResults(msg)

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
}

func TestMapSearchResultsMessagesSlice(t *testing.T) {
	msg := &tg.MessagesMessagesSlice{
		Messages: []tg.MessageClass{
			&tg.MessageEmpty{ID: 2},
		},
		Users: []tg.UserClass{
			&tg.UserEmpty{ID: 11},
		},
		Chats: []tg.ChatClass{
			&tg.ChatEmpty{ID: 21},
		},
	}

	messages, users, chats := mapSearchResults(msg)

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
}

func TestMapSearchResultsChannelMessages(t *testing.T) {
	msg := &tg.MessagesChannelMessages{
		Messages: []tg.MessageClass{
			&tg.MessageEmpty{ID: 3},
		},
		Users: []tg.UserClass{
			&tg.UserEmpty{ID: 12},
		},
		Chats: []tg.ChatClass{
			&tg.ChatEmpty{ID: 22},
		},
	}

	messages, users, chats := mapSearchResults(msg)

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
}

func TestMapSearchResultsNotModified(t *testing.T) {
	msg := &tg.MessagesMessagesNotModified{}

	messages, users, chats := mapSearchResults(msg)

	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}

	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}

	if len(chats) != 0 {
		t.Fatalf("expected 0 chats, got %d", len(chats))
	}
}

func TestMapSearchResultsUnknown(t *testing.T) {
	var msg tg.MessagesMessagesClass = nil

	messages, users, chats := mapSearchResults(msg)

	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}

	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}

	if len(chats) != 0 {
		t.Fatalf("expected 0 chats, got %d", len(chats))
	}
}
