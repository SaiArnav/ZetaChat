package storage

import (
	"database/sql"

	"github.com/Vishvas77/zetachat/core"
)

// Store wraps a local SQLite database used to cache chats and messages
// so the TUI is fast on startup and can show history offline.
type Store struct {
	db *sql.DB
}

// Open opens (and if needed, initializes) the SQLite database at path.
func Open(path string) (*Store, error) {
	// TODO:
	// db, err := sql.Open("sqlite", path) // modernc.org/sqlite driver
	// run migrations (create chats, messages, users tables)
	return &Store{}, nil
}

func (s *Store) SaveChats(chats []core.Chat) error {
	// TODO: upsert into chats table
	return nil
}

func (s *Store) SaveMessages(messages []core.Message) error {
	// TODO: upsert into messages table
	return nil
}

func (s *Store) CachedChats() ([]core.Chat, error) {
	// TODO: read from chats table
	return nil, nil
}

func (s *Store) CachedMessages(chatID string) ([]core.Message, error) {
	// TODO: read from messages table filtered by chat_id
	return nil, nil
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
