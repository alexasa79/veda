package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
)

// helper: return window widths as slice
func getWindowWidths(e *Editor) []int {
	widths := make([]int, len(e.windows))
	for i, w := range e.windows {
		widths[i] = w.Width
	}
	return widths
}

func TestActiveWindowMinWidthOnSwitch(t *testing.T) {
	te := NewTestEditor(180, 40) // total width 180; min active width is 100 by default
	// Start with a split
	te.TypeKeys("space v")
	te.render()

	if len(te.windows) != 2 {
		t.Fatalf("expected 2 windows after vsplit, got %d", len(te.windows))
	}
	// Active is the new right window (index 1). It should be at least minActiveWindowWidth
	widths := getWindowWidths(te.Editor)
	if widths[te.activeWindow] < te.minActiveWindowWidth {
		t.Fatalf("active window width too small after split: got %d, want >= %d (all widths=%v)", widths[te.activeWindow], te.minActiveWindowWidth, widths)
	}

	// Switch to left window and verify it gets widened
	te.TypeKeys("space 1")
	te.render()
	widths = getWindowWidths(te.Editor)
	if te.activeWindow != 0 {
		t.Fatalf("expected active window 0 after 'space 1', got %d", te.activeWindow)
	}
	if widths[0] < te.minActiveWindowWidth {
		t.Fatalf("left window not widened: got %d, want >= %d (widths=%v)", widths[0], te.minActiveWindowWidth, widths)
	}
}

// Compute wrapped line count for window idx by using the window's region
func wrappedCountForWindow(t *testing.T, te *TestEditor, idx int) int {
	t.Helper()
	if idx < 0 || idx >= len(te.windows) {
		t.Fatalf("invalid window index %d", idx)
	}
	win := te.windows[idx]
	// Create a sub-screen and temporarily bind editor to that region
	sub := newWindowScreen(te.mockScreen, win.X, win.Y, win.Width, win.Height, false)
	realScreen := te.screen
	realBuffer := te.CurrentBuffer()
	te.screen = sub
	// Map window state onto buffer
	savedView := win.Buffer.ViewIterator
	savedCursor := win.Buffer.CursorIterator
	savedX := win.Buffer.CursorX
	viewLine := win.ViewLine
	if viewLine < 0 {
		viewLine = 0
	}
	cursorLine := win.CursorLine
	if cursorLine < 0 {
		cursorLine = 0
	}
	win.Buffer.ViewIterator = buffer.NewBufferIteratorAtLine(win.Buffer, viewLine)
	win.Buffer.CursorIterator = buffer.NewBufferIteratorAtLine(win.Buffer, cursorLine)
	win.Buffer.CursorX = win.CursorX
	te.SetCurrentBuffer(win.Buffer)
	// Calculate wrapped lines for the current view
	wrapped := te.CalculateWrappedLines()
	count := len(wrapped)
	// Restore
	win.Buffer.ViewIterator = savedView
	win.Buffer.CursorIterator = savedCursor
	win.Buffer.CursorX = savedX
	te.screen = realScreen
	te.SetCurrentBuffer(realBuffer)
	return count
}

func TestActiveWindowDoesNotWrapLongLine(t *testing.T) {
	te := NewTestEditor(180, 40)
	// Split
	te.TypeKeys("space v")
	te.render()
	if len(te.windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(te.windows))
	}

	// Prepare a long line that fits in the active window (min width 100, ~96 text cols after gutter)
	// but will wrap in the narrower inactive window.
	long := strings.Repeat("X", 96)

	// Put content into both windows' buffers
	for _, w := range te.windows {
		if w.Buffer == nil {
			w.addBufferToWindow(buffer.NewBuffer("", "", "", te.config.DefaultIndentType, te.config.DefaultIndentWidth))
		}
		w.Buffer.Rope = buffer.NewRopeFromString(long)
		w.Buffer.LineCount = 1
		w.Buffer.UpdateLineNumberWidth()
		w.ViewLine = 0
		w.CursorLine = 0
		w.CursorX = 0
	}

	// Activate right and verify right does not wrap, left does
	te.TypeKeys("space 2")
	te.render()
	rightWrapped := wrappedCountForWindow(t, te, 1)
	leftWrapped := wrappedCountForWindow(t, te, 0)
	if rightWrapped != 1 {
		t.Fatalf("expected right window not to wrap (1 wrapped line), got %d", rightWrapped)
	}
	if leftWrapped <= 1 {
		t.Fatalf("expected left window to wrap (>1 wrapped lines), got %d", leftWrapped)
	}

	// Now activate left and verify left does not wrap
	te.TypeKeys("space 1")
	te.render()
	leftWrapped = wrappedCountForWindow(t, te, 0)
	if leftWrapped != 1 {
		t.Fatalf("expected left window not to wrap after activation, got %d", leftWrapped)
	}
}
