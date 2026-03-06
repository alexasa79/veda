package editor

import (
	"fmt"
	"strings"
	"testing"
)

// TestScrollWithWrappedLongLines verifies that the editor can scroll to the end
// of a buffer where long lines wrap to more visual rows than the screen height.
// This is a regression test for the bug where maxTop used logical LineCount
// mixed with visual visibleHeight, preventing scrolling when lines wrap.
func TestScrollWithWrappedLongLines(t *testing.T) {
	// 80 columns wide, 10 rows tall.
	// With gutter ~5 chars, available width ~75.
	// Each 300-char line wraps to ~4 visual rows.
	// 16 lines × 4 rows = ~64 visual rows, well beyond visible height (~9).
	te := NewTestEditorWithContent(80, 10, makeLongLines(16, 300))

	// go-to-end should place cursor on the last logical line.
	err := te.ExecuteCommand("go-to-end")
	if err != nil {
		t.Fatalf("go-to-end failed: %v", err)
	}

	_, cursorY := te.GetCursorPosition()
	lastLine := te.CurrentBuffer().LineCount - 1
	if cursorY != lastLine {
		t.Errorf("go-to-end: expected cursor on line %d, got %d", lastLine, cursorY)
	}

	// The view must have scrolled past line 0 so the last line is visible.
	viewTop := te.CurrentBuffer().ViewIterator.LineNumber()
	if viewTop == 0 {
		t.Errorf("view did not scroll: viewTop is still 0 (last line is %d)", lastLine)
	}

	// Verify the cursor is within the visible area after scroll.
	te.scroll()
	visY, _ := te.GetCursorVisualPosition()
	visibleHeight := te.getVisibleHeight()
	if visY < 0 || visY >= visibleHeight {
		t.Errorf("cursor visual Y=%d is outside visible area [0, %d)", visY, visibleHeight)
	}
}

// TestScrollDownReachesLastLineWithWrapping verifies that repeated downward
// movement can reach the last line even when earlier lines wrap heavily.
func TestScrollDownReachesLastLineWithWrapping(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, makeLongLines(16, 300))
	// Use a small scroll offset so it doesn't exceed visible height.
	te.config.ScrollOffset = 3

	lastLine := te.CurrentBuffer().LineCount - 1

	// Press down enough times to reach the last line.
	for i := 0; i < 500; i++ {
		_ = te.ExecuteCommand("down")
		te.scroll()
		_, cursorY := te.GetCursorPosition()
		if cursorY == lastLine {
			return // success
		}
	}

	_, cursorY := te.GetCursorPosition()
	t.Errorf("could not reach last line %d after 500 downs; stuck at line %d", lastLine, cursorY)
}

// makeLongLines creates n lines, each approximately lineLen characters long.
func makeLongLines(n, lineLen int) string {
	lorem := "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua "
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		prefix := fmt.Sprintf("Line %d: ", i+1)
		var lb strings.Builder
		lb.WriteString(prefix)
		for lb.Len() < lineLen {
			lb.WriteString(lorem)
		}
		lines[i] = lb.String()[:lineLen]
	}
	return strings.Join(lines, "\n")
}
