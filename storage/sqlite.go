package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/SaiArnav/ZetaChat/core"
)

const schema = `
CREATE TABLE IF NOT EXISTS chats (
	id          TEXT NOT NULL,
	platform    TEXT NOT NULL,
	name        TEXT NOT NULL DEFAULT '',
	is_group    INTEGER NOT NULL DEFAULT 0,
	unread      INTEGER NOT NULL DEFAULT 0,
	updated_at  INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (platform, id)
);

CREATE TABLE IF NOT EXISTS messages (
	id           TEXT NOT NULL,
	platform     TEXT NOT NULL,
	chat_id      TEXT NOT NULL,
	sender_id    TEXT NOT NULL DEFAULT '',
	sender_name  TEXT NOT NULL DEFAULT '',
	sender_handle TEXT NOT NULL DEFAULT '',
	text         TEXT NOT NULL DEFAULT '',
	ts           INTEGER NOT NULL DEFAULT 0,
	is_out       INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (platform, chat_id, id)
);

CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages (platform, chat_id, ts);
`

// Store wraps a local SQLite database used to cache chats and messages
// so the TUI is fast on startup and can show history offline.
type Store struct {
	db *sql.DB
}

// Open opens (and if needed, initializes) the SQLite database at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: single writer
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// SaveChats upserts chats into the cache.
func (s *Store) SaveChats(chats []core.Chat) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO chats (id, platform, name, is_group, unread, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(platform, id) DO UPDATE SET
			name = excluded.name,
			is_group = excluded.is_group,
			unread = excluded.unread,
			updated_at = excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range chats {
		var last int64
		if c.LastMessage != nil {
			last = c.LastMessage.Timestamp.Unix()
		}
		if _, err := stmt.Exec(c.ID, string(c.Platform), c.Name, boolInt(c.IsGroup), c.UnreadCount, last); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SaveMessages upserts messages into the cache.
func (s *Store) SaveMessages(messages []core.Message) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO messages
		(id, platform, chat_id, sender_id, sender_name, sender_handle, text, ts, is_out)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(platform, chat_id, id) DO UPDATE SET
			sender_name = excluded.sender_name,
			sender_handle = excluded.sender_handle,
			text = excluded.text,
			ts = excluded.ts,
			is_out = excluded.is_out`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range messages {
		if _, err := stmt.Exec(
			m.ID, string(m.Platform), m.ChatID,
			m.Sender.ID, m.Sender.DisplayName, m.Sender.Handle,
			m.Text, m.Timestamp.Unix(), boolInt(m.Out),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CachedChats reads all cached chats.
func (s *Store) CachedChats() ([]core.Chat, error) {
	rows, err := s.db.Query(`SELECT id, platform, name, is_group, unread, updated_at FROM chats ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := make([]core.Chat, 0, 32)
	for rows.Next() {
		var (
			c         core.Chat
			platform  string
			isGroup   int
			updatedAt int64
		)
		if err := rows.Scan(&c.ID, &platform, &c.Name, &isGroup, &c.UnreadCount, &updatedAt); err != nil {
			return nil, err
		}
		c.Platform = core.Platform(platform)
		c.IsGroup = isGroup != 0
		c.UpdatedAt = time.Unix(updatedAt, 0)
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

// CachedMessages reads cached messages for a chat, oldest first.
func (s *Store) CachedMessages(chatID string) ([]core.Message, error) {
	rows, err := s.db.Query(`SELECT id, sender_id, sender_name, sender_handle, text, ts, is_out
		FROM messages WHERE chat_id = ? ORDER BY ts ASC, rowid ASC`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := make([]core.Message, 0, 64)
	for rows.Next() {
		var (
			m          core.Message
			senderID   string
			senderName string
			handle     string
			ts         int64
			isOut      int
		)
		if err := rows.Scan(&m.ID, &senderID, &senderName, &handle, &m.Text, &ts, &isOut); err != nil {
			return nil, err
		}
		m.ChatID = chatID
		m.Platform = core.PlatformTelegram
		m.Sender = core.User{ID: senderID, DisplayName: senderName, Handle: handle}
		m.Timestamp = time.Unix(ts, 0)
		m.Out = isOut != 0
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// Close closes the database.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
