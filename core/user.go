package core

// User represents a participant in a chat, normalized across platforms.
type User struct {
	ID          string
	Platform    Platform
	DisplayName string
	Handle      string // @username, phone number, or platform-specific ID
	AvatarURL   string
}
