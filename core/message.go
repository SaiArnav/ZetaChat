package core

import "time"

// Platform identifies which messaging service a message/chat belongs to.
type Platform string

const (
	PlatformTelegram Platform = "telegram"
	PlatformWhatsApp Platform = "whatsapp"
	PlatformDiscord  Platform = "discord"
)

// Attachment represents a file, image, or media item attached to a message.
type Attachment struct {
	ID        string
	FileName  string
	MimeType  string
	URL       string // local cache path or remote URL, adapter-defined
	SizeBytes int64
}

// Message is the platform-agnostic representation every adapter must produce.
type Message struct {
	ID          string
	Platform    Platform
	ChatID      string
	Sender      User
	Text        string
	Timestamp   time.Time
	Attachments []Attachment
}
