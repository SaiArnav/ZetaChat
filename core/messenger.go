package core

import "context"

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

// LiveMessenger extends Messenger for adapters that stream live updates
// and manage their own connection lifecycle. The TUI drives platforms
// exclusively through this interface.
type LiveMessenger interface {
	Messenger

	// PlatformName reports which platform this adapter serves.
	PlatformName() Platform

	// Connect starts the client in the background and returns immediately.
	Connect(ctx context.Context)

	// Ready is closed once the adapter finished connecting (success or error).
	Ready() <-chan struct{}

	// AuthErr returns the connection/auth error, if any.
	AuthErr() error

	// SelfName returns a display name for the authenticated account.
	SelfName() string

	// Updates yields incoming messages until the adapter is closed.
	Updates() <-chan Message

	// Close shuts the connection down.
	Close() error
}

// QRStreamer is implemented by adapters that pair by scanning a QR code
// (e.g. WhatsApp). The TUI renders each streamed code on screen until the
// user scans it.
type QRStreamer interface {
	QRCodes() <-chan string
}

// ChatReadMarker is an optional adapter capability: sending a read
// receipt so the platform's own unread counter clears (otherwise the
// count resurrects on every Chats() refresh).
type ChatReadMarker interface {
	// MarkChatRead tells the platform every message in chatID up to
	// lastMsgID has been seen by the user.
	MarkChatRead(chatID, lastMsgID string) error
}

// sendGate is the process-wide confirmation gate consulted before any
// adapter delivers a message. Nil means sends are allowed.
var sendGate func(chatID, text string) error

// ErrSendDenied is returned by a gate that refuses a delivery.
var ErrSendDenied = errSendDenied{}

type errSendDenied struct{}

func (errSendDenied) Error() string { return "send denied by confirmation gate" }

// SetSendGate installs the global send gate — the single choke point
// every SendMessage passes through (SPEC.md section 9: "no silent
// sends, ever"). Agent/automation layers MUST install a gate that asks
// a human before returning nil.
func SetSendGate(g func(chatID, text string) error) { sendGate = g }

// GateSend applies the installed gate. Adapters call this at the top of
// SendMessage; a non-nil error aborts the delivery.
func GateSend(chatID, text string) error {
	if sendGate == nil {
		return nil
	}
	return sendGate(chatID, text)
}
