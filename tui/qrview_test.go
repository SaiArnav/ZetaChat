package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/SaiArnav/ZetaChat/core"
)

func TestQRGridAddsQuietZone(t *testing.T) {
	grid, err := qrGrid("https://example.com/link")
	if err != nil {
		t.Fatalf("qrGrid: %v", err)
	}
	if len(grid) == 0 || len(grid[0]) != len(grid) {
		t.Fatalf("qr grid is not square: %dx%d", len(grid[0]), len(grid))
	}
	n := qrQuietZone
	for i := 0; i < n; i++ {
		edge := len(grid) - 1 - i
		for x := range grid[i] {
			if grid[i][x] || grid[edge][x] {
				t.Fatal("quiet zone rows contain dark modules")
			}
			if grid[x][i] || grid[x][edge] {
				t.Fatal("quiet zone columns contain dark modules")
			}
		}
	}
}

func TestDrawQRShape(t *testing.T) {
	grid := [][]bool{
		{true, false},
		{false, true},
		{true, false},
	}
	out := drawQR(grid)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines for a 3-row grid, got %d", len(lines))
	}
	for i, ln := range lines {
		if n := utf8.RuneCountInString(ln); n != 2 {
			t.Fatalf("line %d has width %d, want 2", i, n)
		}
	}
}

func TestQRViewRendersNativeQR(t *testing.T) {
	m := Model{
		width:      100,
		height:     40,
		activePlat: core.PlatformWhatsApp,
		qrCode:     "2@AbCdEf1234567890abcdef,AbCdEf12345678901234567890abcdef==",
	}
	view := m.qrView()
	if !strings.Contains(view, "WHATSAPP") {
		t.Fatal("view missing platform title")
	}
	if !strings.Contains(view, "█") {
		t.Fatal("view missing QR blocks")
	}
	if strings.Contains(view, "enlarge") {
		t.Fatal("unexpected too-small message on a large terminal")
	}
}

func TestQRViewTooSmallMessage(t *testing.T) {
	m := Model{
		width:      24,
		height:     8,
		activePlat: core.PlatformWhatsApp,
		qrCode:     "2@AbCdEf1234567890abcdef,AbCdEf12345678901234567890abcdef==",
	}
	if view := m.qrView(); !strings.Contains(view, "enlarge") {
		t.Fatal("expected too-small message on a tiny terminal")
	}
}

func TestQRViewWaitingState(t *testing.T) {
	m := Model{
		width:      100,
		height:     40,
		activePlat: core.PlatformWhatsApp,
	}
	if view := m.qrView(); !strings.Contains(view, "waiting for pairing code") {
		t.Fatal("expected waiting message when no QR code yet")
	}
}
