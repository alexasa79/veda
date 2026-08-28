package editor

import (
	"strings"
	"testing"
	"veda/internal/ui"
)

func TestTerminalEmulatorFishRightPromptStaysOnInputRow(t *testing.T) {
	te := NewTerminalEmulator(10, 4)
	te.Write([]byte("╰─> "))

	// Fish draws its right prompt through the final column, returns to the
	// beginning of the row, and redraws the syntax-highlighted command.
	te.Write([]byte("\r\x1b[8C─╯\r\x1b[4Cl"))
	te.Write([]byte("\r\x1b[8C─╯\r\x1b[5Cs"))

	curX, curY := te.Cursor()
	if curX != 6 || curY != 0 {
		t.Fatalf("cursor = (%d, %d), want (6, 0); render:\n%s", curX, curY, te.Render())
	}

	lines := strings.Split(te.Render(), "\n")
	if got, want := lines[0], "╰─> ls  ─╯"; got != want {
		t.Fatalf("input row = %q, want %q; render:\n%s", got, want, te.Render())
	}
}

func TestTerminalEmulatorWrapsBeforeNextPrintableCharacter(t *testing.T) {
	te := NewTerminalEmulator(4, 2)
	te.Write([]byte("abcd"))

	curX, curY := te.Cursor()
	if curX != 3 || curY != 0 {
		t.Fatalf("cursor after final column = (%d, %d), want (3, 0)", curX, curY)
	}

	te.Write([]byte("e"))
	curX, curY = te.Cursor()
	if curX != 1 || curY != 1 {
		t.Fatalf("cursor after wrapped character = (%d, %d), want (1, 1)", curX, curY)
	}

	if got, want := te.Render(), "abcd\ne"; got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

func TestTerminalEmulatorEraseClearsCellStyles(t *testing.T) {
	te := NewTerminalEmulator(4, 2)
	te.Write([]byte("\x1b[31;44;1mtext\r\x1b[2K"))

	for x := 0; x < 4; x++ {
		fg, bg, attrs := te.StyleAt(x, 0)
		if fg != ui.ColorDefault || bg != ui.ColorDefault || attrs != ui.AttrNone {
			t.Fatalf("style at (%d, 0) after erase = (%v, %v, %v), want defaults", x, fg, bg, attrs)
		}
	}
}
