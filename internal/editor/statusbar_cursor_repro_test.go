package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestStatusBarCursor_Repro reproduces the "cursor appears over status bar"
// issue in a normal (non-diff) single-window view. It constructs content
// where a single very long line fills the entire visible content area, and
// then places the cursor on the empty logical line created by a trailing
// newline. The current GetCursorVisualPosition mapping can return a visual
// row one past the last wrapped row, which updateCursor then draws on the
// status bar row.
func TestStatusBarCursor_Repro(t *testing.T) {
	// Create a reasonably small screen so the content area height is limited.
	// Height includes the status bar line; content area = height - 1.
	width := 80
	height := 16
	te := NewTestEditor(width, height)

	// Build a single very long line that will wrap across many visual rows
	// in a normal single-pane view, followed by a trailing newline to create
	// an empty logical line that is initially off-screen.
	long := strings.Repeat("A", 4000)
	content := long + "\n"

	// Install the content into a regular (non-diff) buffer and attach it to the window.
	buf := buffer.NewBuffer(content, "repro.txt", "repro.txt", config.IndentTabs, 4)
	te.buffers[buf.FilePath] = buf
	te.bufferOrder = append(te.bufferOrder, buf.FilePath)
	te.windows[te.activeWindow].Buffer = buf
	te.SetCurrentBuffer(buf)
	te.CurrentBuffer().ViewIterator = buffer.NewBufferIterator(te.CurrentBuffer())
	te.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(te.CurrentBuffer())

	// Place the cursor on the empty logical line after the trailing newline.
	// Logical line index 1 corresponds to the empty line (line 2, 1-based).
	te.CurrentBuffer().CursorIterator = buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), 1)
	te.CurrentBuffer().CursorX = 0

	// Render once to place the cursor based on current view/cursor state.
	te.render()

	// Read the physical cursor position from the mock screen.
	_, cursorY := te.mockScreen.GetCursorPosition()
	_, screenH := te.mockScreen.Size()

	// The cursor must not be drawn on the status bar row (screenH-1).
	if cursorY == screenH-1 {
		t.Fatalf("cursor incorrectly placed on status bar (y=%d, statusbar=%d)", cursorY, screenH-1)
	}

	// Also sanity-check that the cursor is visible and within the content area bounds.
	if cursorY < 0 || cursorY >= screenH-1 {
		t.Fatalf("cursor Y out of content bounds: y=%d, content max=%d", cursorY, screenH-2)
	}
}
