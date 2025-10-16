package editor

import (
	"testing"
)

// TestTerminalClearWithScrollingRegion tests the behavior when bash uses
// a scrolling region and then clears. This is what actually happens with Ctrl+L.
func TestTerminalClearWithScrollingRegion(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create and open a terminal
	te.ExecuteCommand("term")

	buf := te.CurrentBuffer()
	if buf == nil {
		t.Fatal("No buffer after opening terminal")
	}

	ts := te.terminalSessions[buf.FilePath]
	if ts == nil || ts.term == nil {
		t.Fatal("No terminal session or emulator")
	}

	// Simulate what bash does: it sets up a scrolling region
	// and then fills it with content
	// Bash typically uses the whole screen as scroll region: ESC[1;24r (rows 1-24)
	ts.term.Write([]byte("\x1b[1;24r"))

	// Fill with content
	for i := 0; i < 100; i++ {
		ts.term.Write([]byte("Line\r\n"))
	}

	// Now the screen is full and has scrollback
	scrollbackBefore := ts.term.ScrollbackSize()
	if scrollbackBefore == 0 {
		t.Fatal("Expected scrollback after writing 100 lines")
	}

	// Now simulate what bash does for "clear" command or Ctrl+L
	// It moves cursor to 1,1 and clears from cursor to end of screen
	ts.term.Write([]byte("\x1b[H\x1b[J"))

	// The cursor should be at top
	curX, curY := ts.term.Cursor()
	if curX != 0 || curY != 0 {
		t.Errorf("After clear, expected cursor at (0, 0), got (%d, %d)", curX, curY)
	}

	// The visible screen should be clear
	render := ts.term.Render()
	_, height := ts.term.Dimensions()
	totalRows := ts.term.TotalRows()
	lines := splitLines(render)

	// Get the visible portion (last `height` rows)
	visibleStart := totalRows - height
	if visibleStart < 0 {
		visibleStart = 0
	}

	allBlank := true
	for i := visibleStart; i < len(lines) && i < totalRows; i++ {
		line := lines[i]
		for _, ch := range line {
			if ch != ' ' {
				allBlank = false
				t.Logf("Line %d not blank: %q", i, line)
				break
			}
		}
	}

	if !allBlank {
		t.Error("Expected visible screen to be blank after clear")
		t.Logf("Total lines in render: %d", len(lines))
		t.Logf("Total rows: %d, height: %d, visible start: %d", totalRows, height, visibleStart)
	}
}

func TestTerminalCtrlLClearsScreen(t *testing.T) {
	te := NewTestEditor(80, 24)

	// Create and open a terminal
	te.ExecuteCommand("term")

	// Get the terminal emulator from the terminal session
	buf := te.CurrentBuffer()
	if buf == nil {
		t.Fatal("No buffer after opening terminal")
	}

	ts := te.terminalSessions[buf.FilePath]
	if ts == nil || ts.term == nil {
		t.Fatal("No terminal session or emulator")
	}

	// Simulate terminal output with many lines (simulating `make test` output)
	for i := 0; i < 100; i++ {
		ts.term.Write([]byte("Line " + string(rune('0'+i%10)) + "\r\n"))
	}

	// At this point, the terminal should have scrolled
	// Check that we have scrollback
	scrollbackBefore := ts.term.ScrollbackSize()
	if scrollbackBefore == 0 {
		t.Fatal("Expected scrollback after writing 100 lines to 24-line terminal")
	}

	// Simulate Ctrl+L: bash typically sends ESC[H (move cursor to 1,1) followed by ESC[2J (clear display)
	// Or it might send ESC[H ESC[J (clear from cursor to end)
	ts.term.Write([]byte("\x1b[H\x1b[2J"))

	// After Ctrl+L:
	// 1. The cursor should be at the top (0, 0)
	curX, curY := ts.term.Cursor()
	if curX != 0 || curY != 0 {
		t.Errorf("After Ctrl+L clear, expected cursor at (0, 0), got (%d, %d)", curX, curY)
	}

	// 2. The screen should be clear (all spaces)
	render := ts.term.Render()
	// The render should be mostly empty (just scrollback + empty visible area)
	// After a clear, when we look at the visible area, it should be blank
	width, height := ts.term.Dimensions()
	totalRows := ts.term.TotalRows()

	// Check that the visible portion (last `height` rows) is empty
	lines := splitLines(render)
	visibleStart := totalRows - height
	if visibleStart < 0 {
		visibleStart = 0
	}

	for i := visibleStart; i < len(lines) && i < totalRows; i++ {
		line := lines[i]
		for _, ch := range line {
			if ch != ' ' {
				t.Errorf("After Ctrl+L clear, expected visible screen to be blank, but line %d contains: %q", i, line)
				break
			}
		}
	}

	// 3. Scrollback should still exist (history is preserved)
	scrollbackAfter := ts.term.ScrollbackSize()
	if scrollbackAfter == 0 {
		t.Error("Expected scrollback to be preserved after Ctrl+L")
	}

	// The scrollback might be different because of how the clear is implemented,
	// but it should still contain historical data
	_ = width // use to avoid unused variable error
}

func splitLines(s string) []string {
	var lines []string
	var current string
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
