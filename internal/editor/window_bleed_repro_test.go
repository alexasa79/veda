package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/buffer"
)

// buildLargeContent returns many long lines so a single-window render fills
// every screen row and column with non-space characters.
func buildLargeContent(numLines int) string {
	var b strings.Builder
	for i := 0; i < numLines; i++ {
		// Make lines much wider than any test screen to ensure content area fills
		b.WriteString(fmt.Sprintf("LEFT %03d ", i))
		b.WriteString(strings.Repeat("L", 400))
		if i != numLines-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// buildSmallContent returns a few short lines that intentionally do NOT fill
// the window vertically.
func buildSmallContent() string {
	lines := []string{
		"module veda",
		"go 1.24.0",
		"require github.com/some/dependency v0.0.0",
	}
	return strings.Join(lines, "\n")
}

// wrappedCountForWindowLocal computes wrapped line count for a specific window
// by temporarily binding the editor to that window's region/state.
func wrappedCountForWindowLocal(t *testing.T, te *TestEditor, idx int) int {
	t.Helper()
	if idx < 0 || idx >= len(te.windows) {
		t.Fatalf("invalid window index %d", idx)
	}
	win := te.windows[idx]
	sub := newWindowScreen(te.mockScreen, win.X, win.Y, win.Width, win.Height, false)
	realScreen := te.screen
	realBuffer := te.buffer
	te.screen = sub
	// Map window state
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
	te.buffer = win.Buffer
	wrapped := te.CalculateWrappedLines()
	count := len(wrapped)
	// Restore
	win.Buffer.ViewIterator = savedView
	win.Buffer.CursorIterator = savedCursor
	win.Buffer.CursorX = savedX
	te.screen = realScreen
	te.buffer = realBuffer
	return count
}

// TestRightWindowBottomShouldBeEmpty_Repro constructs the user scenario:
//  1. Render a large buffer full-screen (fills the grid with non-spaces)
//  2. Split vertically (new right window becomes active)
//  3. Put a small buffer in the right window
//  4. Expect the bottom of the right window to be empty (spaces) — this is the
//     intended behavior. The current implementation leaves stale cells, so this
//     test should FAIL, serving as a reproduction of the bleed-through bug.
func TestRightWindowBottomShouldBeEmpty_Repro(t *testing.T) {
	// Use a moderately large screen; right window will get at least the
	// configured minimum active width, left will shrink.
	te := NewTestEditorWithContent(120, 30, buildLargeContent(200))
	// Ensure the sole window uses the editor buffer we just created.
	te.windows[0].Buffer = te.buffer

	// Initial render in single-window mode; this paints the entire grid with
	// non-space content from the left buffer.
	te.render()

	// Split vertically — the new right window becomes active.
	te.TypeKeys("space v")
	te.render()

	if len(te.windows) != 2 {
		t.Fatalf("expected 2 windows after split, got %d", len(te.windows))
	}

	// Populate the right window with a small buffer (few short lines).
	right := te.windows[1]
	rightBuffer := right.Buffer
	rightBuffer.Rope = buffer.NewRopeFromString(buildSmallContent())
	rightBuffer.LineCount = strings.Count(rightBuffer.Rope.String(), "\n") + 1
	rightBuffer.UpdateLineNumberWidth()
	right.ViewLine = 0
	right.CursorLine = 0
	right.CursorX = 0

	// Make sure editor points to the active window buffer and render.
	te.buffer = rightBuffer
	te.render()

	// Determine how many visual lines the right buffer actually draws.
	wrapped := wrappedCountForWindowLocal(t, te, 1)
	lineNumberWidth := rightBuffer.LineNumberWidth

	// For each row below the last drawn wrapped line, assert the content area
	// of the right window (beyond the gutter) is empty (spaces).
	startY := right.Y + wrapped
	endY := right.Y + right.Height - 1
	if startY > endY {
		t.Fatalf("unexpected geometry: startY %d > endY %d", startY, endY)
	}

	for y := startY; y <= endY; y++ {
		for x := right.X + lineNumberWidth; x < right.X+right.Width; x++ {
			r, _ := te.mockScreen.GetCell(x, y)
			if r != ' ' {
				t.Fatalf("expected right window bottom area to be empty; found '%c' at (%d,%d)", r, x, y)
			}
		}
	}
}
