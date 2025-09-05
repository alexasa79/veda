package editor

import (
	"fmt"
	"testing"
)

// Ensure that ShowInfo does not change visible height or trigger scrolling
// when the cursor is near the bottom of the viewport.
func TestShowInfoDoesNotAffectLayoutOrScroll(t *testing.T) {
	// Small terminal to make the threshold easy to reason about
	te := NewTestEditorWithContent(80, 24, makeNLines(100))

	// Establish initial layout
	initialVisible := te.getVisibleHeight()
	if initialVisible <= 0 {
		t.Fatalf("unexpected initial visible height: %d", initialVisible)
	}

	// Place cursor just below the bottom-scroll threshold so that prior behavior
	// (reserving an extra line for info) would have triggered a scroll.
	// Threshold: topLine + visibleHeight - ScrollOffset
	topLine := te.CurrentBuffer().ViewIterator.LineNumber()
	targetLine := topLine + initialVisible - te.config.ScrollOffset - 1
	if targetLine < 0 {
		targetLine = 0
	}
	if targetLine >= te.CurrentBuffer().LineCount {
		targetLine = te.CurrentBuffer().LineCount - 1
	}

	// Move cursor to targetLine, X=0
	te.CurrentBuffer().CursorIterator = te.CurrentBuffer().ViewIterator
	for te.CurrentBuffer().CursorIterator.LineNumber() < targetLine && !te.CurrentBuffer().CursorIterator.IsEOF() {
		te.CurrentBuffer().CursorIterator = te.CurrentBuffer().CursorIterator.Next()
	}
	te.CurrentBuffer().CursorX = 0

	// Let scroll settle
	te.scroll()
	startView := te.CurrentBuffer().ViewIterator.LineNumber()

	// Show info banner and scroll again
	te.ShowInfo("File saved: %s", "test.txt")
	te.scroll()

	// Visible height should remain the same
	afterVisible := te.getVisibleHeight()
	if afterVisible != initialVisible {
		t.Fatalf("visible height changed after ShowInfo: before=%d after=%d", initialVisible, afterVisible)
	}

	// View top should not have moved
	afterView := te.CurrentBuffer().ViewIterator.LineNumber()
	if afterView != startView {
		t.Fatalf("view scrolled after ShowInfo: before=%d after=%d (visible=%d)", startView, afterView, afterVisible)
	}
}

// Helper to generate N numbered lines
func makeNLines(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += "\n"
		}
		s += fmt.Sprintf("line %d", i+1)
	}
	return s
}
