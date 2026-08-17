package core

// Messenger is the contract every platform adapter must satisfy.
// The TUI, CLI, and AI-agent tools only ever talk to this interface —
// never to a platform-specific client directly.
type Messenger interface {
	// Login authenticates with the platform (may trigger an interactive
	// flow, e.g. QR code or phone code, handled by the adapter itself).
	Login() error

	// Chats returns the list of conversations visible to the account.
	Chats() ([]Chat, error)

	// Messages returns message history for a given chat.
	Messages(chatID string) ([]Message, error)

	// SendMessage sends a text message to a chat.
	// Callers (CLI/TUI/agent) are responsible for confirming intent
	// with the user before calling this — see SPEC.md section 9.
	SendMessage(chatID string, text string) error

	// Search returns messages matching a query within this platform.
	Search(query string) ([]Message, error)

	// Logout ends the session and clears any cached credentials.
	Logout() error
}
