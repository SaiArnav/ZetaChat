package core

import "time"

// Chat represents a conversation thread on any platform (DM, group, channel).
type Chat struct {
	ID          string
	Platform    Platform
	Name        string
	IsGroup     bool
	UnreadCount int
	LastMessage *Message
	UpdatedAt   time.Time
}
