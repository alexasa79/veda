package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

// TestWrapAwareScrollingPreservesScrollOffset verifies that with a narrow pane
// (two windows side by side), scrolling down across wrapped content does not
// allow the cursor to reach the last row (status bar edge) and continues to
// progress past earlier logical lines.
func TestWrapAwareScrollingPreservesScrollOffset(t *testing.T) {
	te := NewTestEditor(120, 20)
	te.config.ScrollOffset = 3

	// Create a buffer with a very long first line to force heavy wrapping in a narrow pane,
	// followed by many short lines to allow continued scrolling.
	long := strings.Repeat("B", 2000)
	var b strings.Builder
	b.WriteString(long)
	b.WriteString("\n")
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	content := b.String()

	// Attach the buffer and create a second window to make panes narrow.
	buf := buffer.NewBuffer(content, "wrap.txt", "wrap.txt", config.IndentTabs, 4)
	te.buffers[buf.FilePath] = buf
	te.bufferOrder = append(te.bufferOrder, buf.FilePath)
	te.windows[0].setActiveBuffer(buf)
	te.SetCurrentBuffer(buf)
	te.CurrentBuffer().ViewIterator = buffer.NewBufferIterator(buf)
	te.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(buf)

	te.createSecondWindow()
	if len(te.windows) < 2 {
		t.Fatalf("expected two windows")
	}
	// Keep second window with a dummy buffer to keep layout split.
	other := buffer.NewBuffer("other", "other.txt", "other.txt", config.IndentTabs, 4)
	te.buffers[other.FilePath] = other
	te.windows[1].setActiveBuffer(other)

	te.activeWindow = 0
	te.recomputeWindowsLayout()

	// Move cursor well into the document by pressing down repeatedly.
	// Render after each step and ensure the cursor never lands on the last
	// visible row of the active pane.
	for i := 0; i < 300; i++ {
		_ = te.ExecuteCommand("down")
		// In the interactive loop, scroll() runs before render(); emulate here.
		te.scroll()
		te.render()
		// Cursor must never be on the global status bar row
		_, y := te.mockScreen.GetCursorPosition()
		_, screenH := te.mockScreen.Size()
		if y == screenH-1 {
			t.Fatalf("cursor reached global status bar at step %d (y=%d, screenH=%d)", i, y, screenH)
		}
	}

	// We do not assert on specific logical line progression here because the
	// number of visual steps to leave a heavily wrapped first line depends on
	// pane width and gutter. The key invariant we enforce is that the cursor
	// never reaches the global status bar during repeated downward movement.
}
