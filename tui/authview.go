package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// authPromptState holds the active interactive auth prompt.
type authPromptState struct {
	title string
	body  string
	input textinput.Model
	reply chan string
}

// startAuthPrompt builds the prompt UI for a pending authPromptMsg.
func (m *Model) startAuthPrompt(msg authPromptMsg) tea.Cmd {
	input := textinput.New()
	if msg.kind == authPassword {
		input.EchoMode = textinput.EchoPassword
	}
	input.Prompt = "❯ "
	input.PromptStyle = baseText.Bold(true).Foreground(lime)
	input.Placeholder = "…"
	input.PlaceholderStyle = dimStyle
	input.CharLimit = 512
	input.TextStyle = baseText
	input.Focus()

	m.authPrompt = &authPromptState{
		title: msg.title,
		body:  msg.body,
		input: input,
		reply: msg.reply,
	}
	return nil
}

func (m Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.authPrompt == nil {
		return m.handleGlobalKey(msg)
	}

	switch msg.Type {
	case tea.KeyEnter:
		ans := m.authPrompt.input.Value()
		m.authPrompt.reply <- ans
		m.authPrompt = nil
		m.stage = stageBooting
		m.status = "Authenticating…"
		return m, nil
	case tea.KeyEsc, tea.KeyCtrlC:
		m.authPrompt.reply <- ""
		m.authPrompt = nil
		m.quitting = true
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.authPrompt.input, cmd = m.authPrompt.input.Update(msg)
		return m, cmd
	}
}

// authView renders the auth prompt card, or the spinner when idle.
func (m Model) authView() string {
	if m.authPrompt == nil {
		return m.centerView(
			m.spinner.View() + "  " + baseText.Render(m.status),
		)
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan).
		Padding(1, 2).
		Width(60)

	content := titleStyle.Render("ZetaChat — " + m.authPrompt.title) +
		"\n\n" +
		baseText.Render(m.authPrompt.body) +
		"\n\n" +
		m.authPrompt.input.View() +
		"\n\n" +
		dimStyle.Render("enter: submit    esc: cancel")

	return m.centerView(card.Render(content))
}

// searchResultsView renders results while search mode is active.
func (m Model) searchResultsView() string {
	if len(m.searchResults) == 0 {
		return dimStyle.Render("Type a query and press enter to search Telegram.")
	}
	var b strings.Builder
	for i, res := range m.searchResults {
		sender := res.Sender.DisplayName
		if sender == "" {
			sender = "?"
		}
		ts := res.Timestamp.Format("01/02 15:04")
		line := dimStyle.Render(ts+" "+sender) + "  " + baseText.Render(res.Text)
		if i == m.searchIdx {
			line = lipgloss.NewStyle().
				Background(lime).
				Foreground(lipgloss.Color("#111827")).
				Render(ts+" "+sender+"  "+truncateRight(res.Text, 60))
		}
		b.WriteString(line)
		b.WriteString("\n\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
