package editor

import (
	"fmt"
	"strings"
	"testing"
)

// TestTerminalScrollback verifies that lines scrolled off the top of the
// terminal emulator are preserved in scrollback and appear in Render().
func TestTerminalScrollback(t *testing.T) {
	cols, rows := 10, 5
	te := NewTerminalEmulator(cols, rows)

	// Write more lines than the height to force scrolling
	total := 8
	for i := 1; i <= total; i++ {
		line := fmt.Sprintf("L%02d\r\n", i)
		te.Write([]byte(line))
	}

	// Expect scrollback to contain lines scrolled past bottom margin.
	// First (rows-1) newlines move within the screen; subsequent newlines scroll.
	expectedSB := total - (rows - 1)
	if expectedSB < 0 {
		expectedSB = 0
	}
	if te.ScrollbackSize() != expectedSB {
		t.Fatalf("expected scrollback size %d, got %d", expectedSB, te.ScrollbackSize())
	}

	// Render and verify the earliest lines are present at the top
	r := te.Render()
	lines := strings.Split(r, "\n")
	if len(lines) < expectedSB {
		t.Fatalf("rendered lines %d shorter than expected scrollback %d", len(lines), expectedSB)
	}
	// Check the first scrollback line is L01
	if expectedSB > 0 && !strings.HasPrefix(lines[0], "L01") {
		t.Fatalf("expected first rendered line to start with L01, got %q", lines[0])
	}
}
