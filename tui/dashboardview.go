package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/SaiArnav/ZetaChat/core"
)

// dashboardView renders the platform selection screen: a row of glowing
// connection cards, one per configured platform.
func (m Model) dashboardView() string {
	cards := make([]string, 0, len(m.platOrder))
	for i, p := range m.platOrder {
		st := m.platState[p]
		if st == nil {
			st = &platformState{}
		}
		cards = append(cards, m.platformCard(p, st, i == m.dashIdx))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Center, cards...)
	if len(cards) == 1 {
		row = lipgloss.NewStyle().Padding(0, 2).Render(row)
	}

	title := titleStyle.Render("ZETACHAT") +
		dimStyle.Render("  //  multi-platform relay")

	hint := dimStyle.Render("↑↓ select  ") +
		helpKey.Render("enter") + helpDim.Render(" connect  ") +
		helpKey.Render("q") + helpDim.Render(" quit")

	status := ""
	if m.status != "" {
		status = "\n\n" + m.status
	}

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(gray).
		Padding(1, 3).
		Render(strings.Join([]string{
			m.spinner.View() + " " + baseText.Render(m.bootSummary()),
			"",
			row,
			status,
			"",
			hint,
		}, "\n"))

	return m.centerView(title + "\n\n" + panel)
}

// bootSummary describes the overall connection state.
func (m Model) bootSummary() string {
	online, total := 0, len(m.platOrder)
	for _, p := range m.platOrder {
		if st := m.platState[p]; st != nil && st.ready && st.err == nil {
			online++
		}
	}
	switch {
	case online == total && total > 0:
		return "all platforms online"
	default:
		return "establishing uplinks…"
	}
}

// platformCard renders one platform tile with icon, status and account.
func (m Model) platformCard(p core.Platform, st *platformState, selected bool) string {
	meta, ok := platformRegistry[p]
	if !ok {
		return ""
	}

	accent := lipgloss.Color(meta.accent)

	iconLines := make([]string, len(meta.icon))
	for i, ln := range meta.icon {
		style := lipgloss.NewStyle().Foreground(accent)
		if !st.ready || st.err != nil {
			style = style.Faint(true)
		}
		iconLines[i] = style.Render(ln)
	}

	head := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Render(meta.name) +
		"  " + st.statusDot() + dimStyle.Render(" "+st.statusLabel())

	account := st.selfName
	if st.err != nil {
		account = truncateRight(st.err.Error(), 24)
	}
	if account == "" {
		account = meta.tagline
	}

	chats := dimStyle.Render(pluralize(st.chatN, "chat"))

	body := strings.Join(iconLines, "\n") + "\n" +
		head + "\n" +
		dimStyle.Render(account) + "\n" +
		chats

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Render(body)

	if selected {
		card = lipgloss.NewStyle().
			Foreground(lime).
			Bold(true).
			Render(card)
	} else {
		card = lipgloss.NewStyle().Faint(true).Render(card)
	}

	return card
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return itoa(n) + " " + noun + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
