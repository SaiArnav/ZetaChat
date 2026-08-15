package tui

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Vishvas77/zetachat/adapters/telegram"
)

type loginResultMsg struct {
	adapter *telegram.Adapter
	err     error
}

// Model is the root Bubble Tea model for ZetaChat.
type Model struct {
	quitting bool
	status   string
	telegram *telegram.Adapter
}

func NewModel() Model {
	return Model{
		status: "No accounts connected",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "a":
			return m, loginCmd()
		}

	case loginResultMsg:
		if msg.err != nil {
			m.status = "Telegram login failed: " + msg.err.Error()
			return m, nil
		}

		// Keep the authenticated adapter alive.
		m.telegram = msg.adapter

		// Test the Telegram connection by fetching chats.
		chats, err := msg.adapter.Chats()
		if err != nil {
			m.status = "Telegram connected, but chats failed: " + err.Error()
			return m, nil
		}

		m.status = fmt.Sprintf(
			"Telegram connected! %d chats found.",
			len(chats),
		)
	}

	return m, nil
}

type telegramLoginCommand struct {
	adapter *telegram.Adapter
}

func (c telegramLoginCommand) Run() error {
	return c.adapter.Login()
}

func (c telegramLoginCommand) SetStdin(_ io.Reader) {}

func (c telegramLoginCommand) SetStdout(_ io.Writer) {}

func (c telegramLoginCommand) SetStderr(_ io.Writer) {}

func loginCmd() tea.Cmd {
	adapter := telegram.New("./session/telegram")

	return tea.Exec(
		telegramLoginCommand{
			adapter: adapter,
		},
		func(err error) tea.Msg {
			return loginResultMsg{
				adapter: adapter,
				err:     err,
			}
		},
	)
}

var boxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	Padding(1, 4).
	Align(lipgloss.Center)

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	content := "ZETACHAT\n\n" +
		m.status +
		"\n\n[A] Add account\n[Q] Quit"

	return boxStyle.Render(content)
}
