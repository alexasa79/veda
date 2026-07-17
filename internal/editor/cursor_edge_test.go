package editor

import (
	"strings"
	"testing"
)

// TestCursorAtScreenEdge tests that typing text until it fills the screen width
// doesn't cause cursor position to go off-screen.
func TestCursorAtScreenEdge(t *testing.T) {
	// Create a narrow editor to make the test faster
	// Width 40 means ~35 chars available after gutter (assuming 5-char gutter)
	width := 40
	height := 10
	te := NewTestEditorWithContent(width, height, "")

	// Enter insert mode
	te.TypeKeys("i")

	// Type characters one by one until we fill the line
	// We'll type more than the available width to trigger the issue
	availableWidth := width - te.CurrentBuffer().LineNumberWidth
	t.Logf("Screen width: %d, gutter: %d, available: %d", width, te.CurrentBuffer().LineNumberWidth, availableWidth)

	// Type exactly enough characters to fill the available width
	for i := 0; i < availableWidth; i++ {
		te.TypeKeys("x")
	}

	t.Logf("After typing %d chars, buffer content length: %d", availableWidth, len(te.GetBufferContent()))
	t.Logf("Cursor position: line=%d, x=%d", te.CurrentBuffer().CursorIterator.LineNumber(), te.CurrentBuffer().CursorX)

	// Render should not panic - this is where the bug manifests
	te.render()

	// Type one more character - cursor should now be on the next line or still valid
	te.TypeKeys("y")

	t.Logf("After typing one more char, cursor position: line=%d, x=%d", te.CurrentBuffer().CursorIterator.LineNumber(), te.CurrentBuffer().CursorX)

	// Render again - should not panic
	te.render()

	// Verify cursor visual position is within screen bounds
	visualLine, visualCol := te.GetCursorVisualPosition()
	screenWidth, screenHeight := te.mockScreen.Size()

	t.Logf("Visual position: line=%d, col=%d, screen: %dx%d", visualLine, visualCol, screenWidth, screenHeight)

	if visualCol >= screenWidth {
		t.Errorf("Cursor visual column %d is off-screen (screen width %d)", visualCol, screenWidth)
	}
	if visualLine >= screenHeight-1 { // -1 for status bar
		t.Errorf("Cursor visual line %d is off-screen (content height %d)", visualLine, screenHeight-1)
	}
}

// TestCursorAtScreenEdgeLongLine tests with a longer line that definitely needs wrapping
func TestCursorAtScreenEdgeLongLine(t *testing.T) {
	width := 50
	height := 10
	te := NewTestEditorWithContent(width, height, "")

	// Enter insert mode
	te.TypeKeys("i")

	// Type a lot of characters - more than screen width
	numChars := width * 2
	for i := 0; i < numChars; i++ {
		te.TypeKeys("a")
		// Render periodically to catch the issue as it happens
		if i%10 == 0 {
			te.render()
		}
	}

	t.Logf("Typed %d chars", numChars)
	t.Logf("Buffer content: %q", te.GetBufferContent())
	t.Logf("Cursor: line=%d, x=%d", te.CurrentBuffer().CursorIterator.LineNumber(), te.CurrentBuffer().CursorX)

	// Final render
	te.render()

	// Check cursor position
	visualLine, visualCol := te.GetCursorVisualPosition()
	screenWidth, _ := te.mockScreen.Size()

	t.Logf("Final visual position: line=%d, col=%d", visualLine, visualCol)

	if visualCol >= screenWidth {
		t.Errorf("Cursor visual column %d >= screen width %d", visualCol, screenWidth)
	}
}

// TestCursorExactlyAtEdge tests the exact edge case where cursor is at position
// that would equal screen width
func TestCursorExactlyAtEdge(t *testing.T) {
	width := 40
	height := 10
	te := NewTestEditorWithContent(width, height, "")

	gutter := te.CurrentBuffer().LineNumberWidth
	availableWidth := width - gutter
	t.Logf("Width=%d, gutter=%d, available=%d", width, gutter, availableWidth)

	// Pre-fill the buffer with exactly (availableWidth) characters
	content := strings.Repeat("x", availableWidth)
	te.TypeKeys("i")
	for _, ch := range content {
		te.InsertChar(ch)
	}

	t.Logf("Buffer length: %d, CursorX: %d", len(te.GetBufferContent()), te.CurrentBuffer().CursorX)

	// Now the cursor should be at position availableWidth (after last char)
	// This is the edge case - cursor at end of line that fills the width
	// The insertion point after an exactly full line is at the start of the
	// next visual row.
	te.render()

	// Check the actual cursor position on screen.
	cursorX, cursorY := te.mockScreen.GetCursorPosition()
	t.Logf("Screen cursor position: x=%d, y=%d (screen width=%d)", cursorX, cursorY, width)

	if cursorX >= width {
		t.Errorf("BUG: Screen cursor X %d >= screen width %d", cursorX, width)
	}
	if cursorX < 0 {
		t.Errorf("BUG: Screen cursor X %d is negative", cursorX)
	}
	if cursorX != gutter || cursorY != 1 {
		t.Errorf("cursor at (%d,%d), want (%d,1)", cursorX, cursorY, gutter)
	}
}
