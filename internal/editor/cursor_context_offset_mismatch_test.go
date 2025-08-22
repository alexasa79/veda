package editor

import (
	"fmt"
	"testing"
	"veda/internal/buffer"
)

// Test reproduces a desync where the physical cursor Y includes the context
// offset while the highlighted "current line" is drawn without it, so they
// appear on different rows when context is shown.
func TestCursorHighlightDesyncWithContextLines(t *testing.T) {
	// Build a small Go file with one function and a long body so that when the
	// view is scrolled into the body, the function declaration line is outside
	// the visible region and becomes a context line.
	var body string
	for i := 0; i < 60; i++ {
		body += fmt.Sprintf("\t_ = %d\n", i)
	}
	content := "package main\n\nfunc f() {\n" + body + "}\n"

	te := NewTestEditorWithContent(100, 20, content)
	// Ensure tree-sitter chooses Go grammar
	te.buffer.FilePath = "main.go"

	// Scroll the view down so the function declaration is above the visible area.
	// This should produce at least one context line (the function signature).
	te.buffer.ViewIterator = buffer.NewBufferIteratorAtLine(te.buffer, 10)

	// Place cursor somewhere in the function body
	te.SetCursorPosition(0, 15)

	// Render once to populate context and draw
	te.render()

	// Require that context is actually shown; otherwise this test isn't meaningful
    if len(te.buffer.ContextCache) == 0 {
		t.Fatalf("expected contextCache > 0 to reproduce bug, got 0")
	}

	// Find the screen row where the current line highlight background is applied.
	// The highlight is applied to the content cells (not the gutter), so probe at
	// the first content column.
	lineBg := te.config.Theme.GetStyle(te.config.Theme.CurrentLine).Bg
	contentX := te.buffer.LineNumberWidth

	_, height := te.screen.Size()
	highlightedY := -1
	for y := 0; y < height; y++ {
		_, style := te.mockScreen.GetCell(contentX, y)
		if style.Bg == lineBg {
			highlightedY = y
			break
		}
	}
	if highlightedY == -1 {
		t.Fatalf("could not locate current-line highlight on screen")
	}

	// Read the physical cursor position placed by updateCursor()
	_, cursorY := te.GetScreenCursorPosition()

	// After fix: the physical cursor Y must match the highlighted line Y.
	diff := cursorY - highlightedY
	if diff != 0 {
        t.Fatalf("expected cursorY and highlightedY to match (diff=0), got diff=%d (cursorY=%d, highlightedY=%d, contextLines=%d)", diff, cursorY, highlightedY, len(te.buffer.ContextCache))
	}
}
