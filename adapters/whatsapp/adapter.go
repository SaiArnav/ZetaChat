// Package whatsapp implements core.LiveMessenger for WhatsApp via whatsmeow.
//
// Login pairs this device with your phone by QR code (WhatsApp → Linked
// devices → Link a device). The session is stored locally, so scanning is
// a one-time step. Chat history comes from the phone's history sync plus
// live messages; WhatsApp has no server-side history API.
package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "modernc.org/sqlite"

	"github.com/SaiArnav/ZetaChat/core"
)

var (
	// ErrNotAuthorized is returned when an operation needs a paired session.
	ErrNotAuthorized = errors.New("whatsapp: not linked yet — scan the QR code")
)

const chatIDPrefix = "wa-"

// Adapter implements core.LiveMessenger for WhatsApp.
type Adapter struct {
	storePath string

	mu        sync.Mutex
	client    *whatsmeow.Client
	authed    bool
	paired    bool
	ready     chan struct{}
	runErr    error
	ctx       context.Context
	cancel    context.CancelFunc
	qrCodes   chan string
	updates   chan core.Message
	closed    bool
	selfName  string
	selfJID   types.JID
	chats     map[string]core.Chat      // jid -> chat metadata
	messages  map[string][]core.Message // jid -> newest-first history
	pushNames map[types.JID]string      // sender jid -> display name
}

// New creates a WhatsApp adapter. The session database lives at storePath.
func New(storePath string) *Adapter {
	return &Adapter{
		storePath: storePath,
		ready:     make(chan struct{}),
		qrCodes:   make(chan string, 4),
		updates:   make(chan core.Message, 256),
		chats:     map[string]core.Chat{},
		messages:  map[string][]core.Message{},
		pushNames: map[types.JID]string{},
	}
}

// PlatformName reports the platform served by this adapter.
func (a *Adapter) PlatformName() core.Platform { return core.PlatformWhatsApp }

// QRCodes streams pairing codes until the device is linked.
func (a *Adapter) QRCodes() <-chan string { return a.qrCodes }

// Updates returns the channel receiving live messages.
func (a *Adapter) Updates() <-chan core.Message { return a.updates }

// Ready is closed once pairing + connecting finished (success or error).
func (a *Adapter) Ready() <-chan struct{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ready
}

// AuthErr returns the connection error, if any.
func (a *Adapter) AuthErr() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runErr
}

// SelfName returns a display name for the linked account.
func (a *Adapter) SelfName() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.selfName != "" {
		return a.selfName
	}
	return "linked device"
}

// Connect opens the WebSocket to WhatsApp in the background. If no session
// exists yet it starts the QR pairing flow and streams codes on QRCodes().
func (a *Adapter) Connect(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	go a.run()
}

func (a *Adapter) run() {
	container, err := sqlstore.New(context.Background(), "sqlite",
		"file:"+a.storePath+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)",
		waLog.Stdout("store", "ERROR", true))
	if err != nil {
		a.markReady(fmt.Errorf("whatsapp store open: %w", err))
		return
	}
	device, err := container.GetFirstDevice(a.ctx)
	if err != nil {
		a.markReady(fmt.Errorf("whatsapp device load: %w", err))
		return
	}

	client := whatsmeow.NewClient(device, waLog.Stdout("client", "WARN", true))
	client.AddEventHandler(a.onEvent)

	a.mu.Lock()
	a.client = client
	unpaired := client.Store.ID == nil
	a.mu.Unlock()

	if unpaired {
		qrChan, err := client.GetQRChannel(a.ctx)
		if err != nil {
			a.markReady(fmt.Errorf("whatsapp qr channel: %w", err))
			return
		}
		if err := client.Connect(); err != nil {
			a.markReady(fmt.Errorf("whatsapp connect: %w", err))
			return
		}
		for evt := range qrChan {
			switch evt.Event {
			case "code":
				select {
				case a.qrCodes <- evt.Code:
				case <-a.ctx.Done():
					a.markReady(nil)
					return
				}
			case "success":
				// pairing succeeded; onEvent(PairSuccess/Connected) finishes up
			case "timeout":
				a.markReady(errors.New("whatsapp: QR expired — restart to try again"))
				return
			default:
			}
		}
		return
	}

	if err := client.Connect(); err != nil {
		a.markReady(fmt.Errorf("whatsapp connect: %w", err))
		return
	}
}

func (a *Adapter) markConnected() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.paired && !a.authed {
		a.authed = true
		close(a.ready)
	}
}

func (a *Adapter) markReady(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	select {
	case <-a.ready:
		return // already closed
	default:
	}
	if err != nil {
		a.runErr = err
	}
	a.authed = true // terminal state reached either way
	close(a.ready)
}

// onEvent processes connection lifecycle and message events.
func (a *Adapter) onEvent(evt interface{}) {
	switch e := evt.(type) {

	case *events.PairSuccess:
		a.mu.Lock()
		a.paired = true
		a.selfJID = e.ID
		a.selfName = "linked: +" + e.ID.User
		a.mu.Unlock()
		a.markConnected()
		a.refreshChats()

	case events.Connected:
		a.mu.Lock()
		a.paired = true
		a.mu.Unlock()
		a.markConnected()
		a.refreshChats()

	case *events.LoggedOut:
		a.markReady(errors.New("whatsapp session logged out — restart to re-link"))

	case *events.HistorySync:
		a.absorbHistory(e)

	case *events.Message:
		msg := a.mapMessage(e.Info, e.Message.GetConversation())
		a.pushMessage(msg)
		a.rememberChat(msg)
		a.rememberPushName(e.Info.Sender, e.Info.PushName)
	}
}

// refreshChats pulls the joined group list into the chat map.
func (a *Adapter) refreshChats() {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return
	}

	groups, err := client.GetJoinedGroups(a.ctx)
	if err != nil {
		return // best effort; DMs come from history/live events anyway
	}
	for _, g := range groups {
		name := g.Name
		if name == "" {
			name = "group-" + g.JID.User
		}
		a.mu.Lock()
		a.chats["wa-"+g.JID.String()] = core.Chat{
			ID:        "wa-" + g.JID.String(),
			Platform:  core.PlatformWhatsApp,
			Name:      name,
			IsGroup:   true,
			UpdatedAt: time.Now(),
		}
		a.mu.Unlock()
	}
}
