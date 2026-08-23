package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/SaiArnav/ZetaChat/core"
)

// dashListWidth is the fixed width of the left selection column.
const dashListWidth = 34

// dashboardView renders a master-detail platform screen: a selection
// rail on the left, a detail panel for the highlighted entry on the
// right, and the live connection status underneath.
func (m Model) dashboardView() string {
	cards := dashCards(m.platOrder)

	// Left rail -------------------------------------------------------
	var rail strings.Builder
	rail.WriteString(sectionStyle.Render("PLATFORMS"))
	rail.WriteString("\n\n")

	for i, p := range cards {
		selected := i == m.dashIdx
		name := metaName(p)
		dot, label := m.cardStatus(p)

		row := "  " + name +
			strings.Repeat(" ", max(0, 16-lipgloss.Width(name))) +
			dot + " " + dimStyle.Render(label)

		if selected {
			row = selectedRow.Render("▌ ") + selectedRow.Render(row)
		}
		rail.WriteString(row)
		rail.WriteString("\n\n")
	}

	railBox := paneBorder(false).
		Width(dashListWidth).
		Render(strings.TrimRight(rail.String(), "\n"))

	// Right detail ----------------------------------------------------
	detail := m.detailPanel(cards[m.dashIdx])

	body := lipgloss.JoinHorizontal(lipgloss.Top, railBox, detail)

	head := titleStyle.Render("ZETACHAT") +
		dimStyle.Render("  //  multi-platform relay")

	rule := mutedSty.Render(strings.Repeat("─", max(20, min(m.width, 72))))

	statusLine := m.spinner.View() + "  " + baseText.Render(m.bootSummary())
	hints := dimStyle.Render(m.help.ShortHelpView(shortHelpFor(m)))

	panel := lipgloss.JoinVertical(lipgloss.Left,
		head,
		rule,
		"",
		body,
		"",
		statusLine,
		" ",
		hints,
	)

	return m.centerView(panel)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// cardStatus returns the dot + label for one dashboard row.
func (m Model) cardStatus(p core.Platform) (string, string) {
	if p == core.PlatformAll {
		return m.aggregateDot()
	}
	st := m.platState[p]
	if st == nil {
		return dimStyle.Render("◌"), "LINKING…"
	}
	return st.statusDot(), st.statusLabel()
}

// detailPanel renders the right-hand description of the selection.
func (m Model) detailPanel(p core.Platform) string {
	meta := platformRegistry[p]
	accent := lipgloss.Color(meta.accent)

	var b strings.Builder

	iconLines := make([]string, len(meta.icon))
	for i, ln := range meta.icon {
		style := lipgloss.NewStyle().Foreground(accent)
		if _, label := m.cardStatus(p); label == "OFFLINE" {
			style = style.Faint(true)
		}
		iconLines[i] = style.Render(ln)
	}

	b.WriteString(lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Render(meta.name))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(meta.tagline))
	b.WriteString("\n\n")
	b.WriteString(strings.Join(iconLines, "\n"))
	b.WriteString("\n\n")

	switch p {
	case core.PlatformAll:
		online := 0
		totalChats := 0
		for _, q := range m.platOrder {
			if st := m.platState[q]; st != nil && st.ready && st.err == nil {
				online++
				totalChats += st.chatN
			}
		}
		b.WriteString(baseText.Render(
			pluralize(online, "platform") + " online"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(pluralize(totalChats, "chat") +
			" in merged inbox"))
	default:
		st := m.platState[p]
		account := meta.tagline
		chats := ""
		if st != nil {
			dot, label := m.cardStatus(p)
			line := dot + " " + label
			if st.err != nil {
				line += " · " + truncateRight(st.err.Error(), 28)
			} else if st.selfName != "" {
				line += " · " + st.selfName
			}
			b.WriteString(line)
			b.WriteString("\n")
			chats = pluralize(st.chatN, "chat")
		}
		_ = account
		if chats != "" {
			b.WriteString(dimStyle.Render(chats))
		}
	}

	return paneBorder(true).Padding(1, 3).Render(b.String())
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
	case m.allResolved():
		return "ready — pick a destination"
	default:
		return "establishing uplinks…"
	}
}

// inboxCardStats is kept for tests / future reuse.
func (m Model) inboxCardStats() (int, int) {
	online, total := 0, 0
	for _, p := range m.platOrder {
		total++
		if st := m.platState[p]; st != nil && st.ready && st.err == nil {
			online++
		}
	}
	return online, total
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
