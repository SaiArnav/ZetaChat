package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"

	"github.com/SaiArnav/ZetaChat/core"
)

var (
	// ErrNotAuthorized is returned when an operation requires an authenticated session.
	ErrNotAuthorized = errors.New("telegram: not authorized")
	// ErrChatNotFound is returned when a chatID cannot be resolved to a peer.
	ErrChatNotFound = errors.New("telegram: chat not found, refresh chats first")
)

// Prompter is the interface the adapter uses to ask the user for
// interactive authentication inputs (phone, code, 2FA password).
// The TUI implements this with on-screen prompts; the CLI uses stdin.
type Prompter interface {
	AskPhone() (string, error)
	AskCode() (string, error)
	AskPassword() (string, error)
	AskTerms(title, text string) (bool, error)
}

// Adapter implements core.Messenger for Telegram over MTProto (gotd/td).
type Adapter struct {
	appID       int
	appHash     string
	sessionPath string
	prompter    Prompter

	mu     sync.Mutex
	client *telegram.Client
	raw    *tg.Client
	sender *message.Sender
	self   *tg.User
	authed bool
	ready  chan struct{}
	runErr error
	ctx    context.Context
	cancel context.CancelFunc

	peers    map[string]peerInfo // core chatID -> telegram peer
	users    map[int64]string    // userID -> display name
	basicG   map[int64]string    // chatID -> title (basic groups)
	channels map[int64]string    // channelID -> title

	// updates receives live incoming messages.
	updates chan core.Message
}

// peerInfo is the resolved information needed to build an InputPeer.
type peerInfo struct {
	kind       string // "user" | "chat" | "channel"
	id         int64
	accessHash int64
	title      string
	username   string
}

func (p peerInfo) inputPeer() tg.InputPeerClass {
	switch p.kind {
	case "user":
		return &tg.InputPeerUser{UserID: p.id, AccessHash: p.accessHash}
	case "chat":
		return &tg.InputPeerChat{ChatID: p.id}
	default:
		return &tg.InputPeerChannel{ChannelID: p.id, AccessHash: p.accessHash}
	}
}

// New creates a Telegram adapter. prompter may be nil (stdin prompts used).
func New(appID int, appHash, sessionPath string, prompter Prompter) *Adapter {
	if prompter == nil {
		prompter = stdinPrompter{}
	}
	return &Adapter{
		appID:       appID,
		appHash:     appHash,
		sessionPath: sessionPath,
		prompter:    prompter,
		peers:       make(map[string]peerInfo),
		users:       make(map[int64]string),
		basicG:      make(map[int64]string),
		channels:    make(map[int64]string),
	}
}

// SetPrompter replaces the interactive auth prompter (used by the TUI).
func (a *Adapter) SetPrompter(p Prompter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.prompter = p
}

// Connect starts the MTProto client in the background. It authenticates
// (interactively if needed) and unblocks Ready(). Returns immediately.
func (a *Adapter) Connect(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.ready = make(chan struct{})
	a.initUpdates()

	a.client = telegram.NewClient(a.appID, a.appHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: a.sessionPath},
		UpdateHandler:  telegram.UpdateHandlerFunc(a.onUpdate),
	})
	a.raw = a.client.API()
	a.sender = message.NewSender(a.raw)

	go func() {
		err := a.client.Run(a.ctx, func(ctx context.Context) error {
			status, err := a.client.Auth().Status(ctx)
			if err != nil {
				return err
			}
			if !status.Authorized {
				flow := auth.NewFlow(
					interactiveAuth{a: a},
					auth.SendCodeOptions{},
				)
				if err := a.client.Auth().IfNecessary(ctx, flow); err != nil {
					return err
				}
				status, err = a.client.Auth().Status(ctx)
				if err != nil {
					return err
				}
			}

			a.mu.Lock()
			a.self = status.User
			a.authed = true
			a.mu.Unlock()
			a.markReady(nil)
			<-ctx.Done()
			return ctx.Err()
		})
		if err != nil {
			a.markReady(err)
		}
	}()
}

func (a *Adapter) markReady(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	select {
	case <-a.ready:
		return // already marked
	default:
	}
	if err != nil {
		a.runErr = err
	}
	close(a.ready)
}

// Ready returns a channel closed when the adapter is connected+authed.
func (a *Adapter) Ready() <-chan struct{} {
	return a.ready
}

// AuthErr returns the connection/auth error, if any.
func (a *Adapter) AuthErr() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runErr
}

// Self returns the authenticated user (or nil before auth).
func (a *Adapter) Self() *tg.User {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.self
}

// SelfName returns a display name for the authenticated user.
func (a *Adapter) SelfName() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.self == nil {
		return ""
	}
	u := a.self
	switch {
	case u.FirstName != "" && u.LastName != "":
		return u.FirstName + " " + u.LastName
	case u.FirstName != "":
		return u.FirstName
	case u.Username != "":
		return "@" + u.Username
	default:
		return fmt.Sprintf("user %d", u.ID)
	}
}

// Updates returns the channel receiving live messages.
func (a *Adapter) Updates() <-chan core.Message {
	return a.updates
}

// Close disconnects from Telegram.
func (a *Adapter) Close() error {
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

// Login authenticates if needed and waits until the session is ready.
func (a *Adapter) Login() error {
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.ready:
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runErr != nil {
		return a.runErr
	}
	if !a.authed {
		return ErrNotAuthorized
	}
	return nil
}

// Logout terminates the session.
func (a *Adapter) Logout() error {
	a.mu.Lock()
	authed := a.authed
	a.mu.Unlock()
	if !authed {
		return ErrNotAuthorized
	}
	if _, err := a.raw.AuthLogOut(a.ctx); err != nil {
		return fmt.Errorf("telegram logout: %w", err)
	}
	a.mu.Lock()
	a.authed = false
	a.mu.Unlock()
	return nil
}
