package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/skip2/go-qrcode"
)

// lipglossAccent builds a bold accent-colored style.
func lipglossAccent(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true)
}

// qrQuietZone is the number of light module rings rendered around the code
// (scanners need it to find the symbol edges).
const qrQuietZone = 2

// qrView renders the pairing screen: a native-resolution QR sized exactly
// to its module count, centered in the terminal.
func (m Model) qrView() string {
	accent := platformRegistry[m.activePlat].accent
	name := metaName(m.activePlat)

	title := lipglossAccent(accent).Render(name) +
		dimStyle.Render("  //  link your phone")

	if m.qrCode == "" {
		body := dimStyle.Render("waiting for pairing code…") +
			"\n\n" + m.spinner.View()
		return m.splashView(title + "\n\n" + body)
	}

	grid, err := qrGrid(m.qrCode)
	if err != nil || len(grid) == 0 {
		return m.splashView(title + "\n\n" +
			dimStyle.Render("failed to render QR — press esc and retry"))
	}

	qrText := drawQR(grid)
	qrLines := strings.Split(qrText, "\n")
	qrW := len(grid[0])
	qrH := len(qrLines)

	needH := qrH + 3 // title + gap + steps line
	needW := qrW + 2

	if m.width < needW || m.height < needH {
		return m.centerPlain(fmt.Sprintf(
			"%s\n\n%s %dx%d %s %dx%d\n\n%s",
			title,
			"QR needs", qrW, qrH, "rows — enlarge this window",
			m.width, m.height,
			dimStyle.Render("then press esc and pick WHATSAPP again"),
		))
	}

	topPad := (m.height - needH) / 2
	leftPad := (m.width - qrW) / 2
	indent := strings.Repeat(" ", leftPad)

	var b strings.Builder
	b.WriteString(strings.Repeat("\n", topPad))
	b.WriteString(centerLine(title, m.width))
	b.WriteString("\n\n")
	for _, ln := range qrLines {
		b.WriteString(indent)
		b.WriteString(ln)
		b.WriteString("\n")
	}
	steps := dimStyle.Render("WhatsApp → Settings → Linked devices → Link a device")
	b.WriteString("\n")
	b.WriteString(centerLine(steps+"   "+dimStyle.Render("esc back"), m.width))

	return b.String()
}

// centerLine pads a styled line so it sits centered for the given width.
func centerLine(s string, width int) string {
	pad := (width - lipgloss.Width(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

// centerPlain centers plain multi-line content on the full screen.
func (m Model) centerPlain(content string) string {
	w, h := m.width, m.height
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 24
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

// qrGrid encodes s into a boolean matrix plus a light quiet-zone border.
// A module is true when it is dark.
func qrGrid(s string) ([][]bool, error) {
	q, err := qrcode.New(s, qrcode.Low)
	if err != nil {
		return nil, err
	}
	q.DisableBorder = true
	src := q.Bitmap()

	n := qrQuietZone
	size := len(src) + n*2
	grid := make([][]bool, size)
	for i := range grid {
		grid[i] = make([]bool, size)
	}
	for y, row := range src {
		copy(grid[y+n][n:], row)
	}
	return grid, nil
}

// drawQR renders a boolean matrix as half-block ASCII at native
// resolution: two module rows per text line, inverted so dark modules use
// the terminal background (light modules become foreground blocks).
func drawQR(grid [][]bool) string {
	var b strings.Builder
	for y := 0; y < len(grid); y += 2 {
		for x := 0; x < len(grid[y]); x++ {
			top := grid[y][x]
			var bottom bool
			if y+1 < len(grid) {
				bottom = grid[y+1][x]
			}
			switch {
			case top && bottom:
				b.WriteString(" ")
			case top:
				b.WriteString("▄")
			case bottom:
				b.WriteString("▀")
			default:
				b.WriteString("█")
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
